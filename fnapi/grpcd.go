package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	builder "github.com/dosco/sanfran/builder/rpc"
	"github.com/dosco/sanfran/fnapi/data"
	"github.com/dosco/sanfran/fnapi/rpc"
	"github.com/dosco/sanfran/lib/clb"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"
	minio "github.com/minio/minio-go"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	codePath   = "/functions/%s"
	bucketName = "functions"
)

type server struct {
	fnstoreLB     clb.Balancer
	builderClient builder.BuilderClient
}

func initServer(clientset *kubernetes.Clientset, port int) {
	clbCfg := clb.Config{
		Namespace:  getNamespace(),
		HostPrefix: getHelmRelease(),
		Services: map[string]clb.Service{
			"builder": clb.Service{Host: "sf-builder", Port: "grpc"},
			"fnstore": clb.Service{Host: "fnstore", Port: "service"},
		},
	}
	lb := clb.NewClb(clientset, clbCfg)

	server := &server{
		builderClient: builder.NewBuilderClient(lb.ClientConn(clbCfg.Get("builder"))),
		fnstoreLB:     clb.HttpRoundRobin(lb),
	}

	if err := server.fnstoreLB.Start(clbCfg.Get("fnstore")); err != nil {
		glog.Fatalln(err.Error())
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) // RPC port
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}
	g := grpc.NewServer()

	rpc.RegisterFnAPIServer(g, server)

	glog.Infof("SanFran/FnAPI GRPC Service Listening on :%d\n", port)
	g.Serve(lis)
}

func (s *server) Create(ctx context.Context, req *rpc.CreateReq) (*rpc.CreateResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reqFn := req.GetFunction()

	builderReq := &builder.BuildReq{
		Name:    reqFn.GetName(),
		Lang:    reqFn.GetLang(),
		Code:    reqFn.GetCode(),
		Package: reqFn.GetPackage(),
		Version: 1,
	}

	_, err := s.builderClient.Build(ctx, builderReq)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	fn := functionFromReq(reqFn)
	fn.Version = 1

	if err := ds.CreateFn(&fn); err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	glog.Infof("[%s] Function created", fn.GetName())
	return &rpc.CreateResp{}, nil
}

func (s *server) Update(ctx context.Context, req *rpc.UpdateReq) (*rpc.UpdateResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reqFn := req.GetFunction()

	oldFn, err := ds.GetFn(reqFn.GetName())
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	builderReq := &builder.BuildReq{
		Name:    reqFn.GetName(),
		Lang:    reqFn.GetLang(),
		Code:    reqFn.GetCode(),
		Package: reqFn.GetPackage(),
		Version: oldFn.Version + 1,
	}

	_, err = s.builderClient.Build(ctx, builderReq)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	fn := functionFromReq(reqFn)

	if err := ds.UpdateFn(&fn); err == ErrKeyNotExists {
		return nil, grpc.Errorf(codes.NotFound, err.Error())
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}
	glog.Infof("[%s] Function updated", fn.GetName())

	return &rpc.UpdateResp{}, nil
}

func (s *server) Get(ctx context.Context, req *rpc.GetReq) (*rpc.GetResp, error) {
	fn, err := ds.GetFn(req.GetName())
	if fn == nil {
		return nil, grpc.Errorf(codes.NotFound, "Not Found")
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	resp := rpc.GetResp{
		Name:    fn.GetName(),
		Lang:    fn.GetLang(),
		Version: fn.GetVersion(),
	}

	glog.Infof("[%s] Function fetched", req.GetName())
	return &resp, nil
}

func (s *server) Delete(ctx context.Context, req *rpc.DeleteReq) (*rpc.DeleteResp, error) {
	fn, err := ds.DeleteFn(req.GetName())

	if err == ErrKeyNotExists {
		return nil, grpc.Errorf(codes.NotFound, err.Error())
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	client, err := getFnstoreClient(s.fnstoreLB)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	fileName := functionFilename(fn.Name, fn.Lang, fn.Version)

	if err = client.RemoveObject(bucketName, fileName); err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	glog.Infof("[%s] Function deleted", req.GetName())
	return &rpc.DeleteResp{}, nil
}

func (s *server) List(ctx context.Context, req *rpc.ListReq) (*rpc.ListResp, error) {
	fns, err := ds.ListFn()
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	var fnNames []string
	for i := range fns {
		fnNames = append(fnNames, fns[i].GetName())
	}

	return &rpc.ListResp{Names: fnNames}, nil
}

func functionFromReq(reqFn *rpc.Function) data.Function {
	return data.Function{
		Name:    reqFn.GetName(),
		Lang:    reqFn.GetLang(),
		Code:    reqFn.GetCode(),
		Package: reqFn.GetPackage(),
		Version: 0,
	}
}

func functionFilename(name, lang string, version int64) string {
	return strings.Join([]string{
		fmt.Sprintf("%s-%d", name, version), lang, "zip"}, ".")
}

func getFnstoreClient(fnstoreLB clb.Balancer) (*minio.Client, error) {
	addr, err := fnstoreLB.Get()
	if err != nil {
		return nil, err
	}

	minioClient, err := minio.New(addr.Addr,
		getFnstoreAccessKey(), getFnstoreSecretKey(), false)
	if err != nil {
		return nil, err
	}

	return minioClient, nil
}
