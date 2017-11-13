package main

import (
	"fmt"

	"github.com/dosco/sanfran/fnapi/data"
	"github.com/dosco/sanfran/fnapi/rpc"
	"github.com/golang/glog"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const codePath = "/code/%s?v=%d"

type server struct{}

func (s *server) Create(ctx context.Context, req *rpc.CreateReq) (*rpc.CreateResp, error) {
	fn := functionFromReq(req.GetFunction())

	if err := ds.CreateFn(&fn); err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	glog.Infof("[%s] Function created", fn.GetName())

	link := fmt.Sprintf(codePath, fn.GetName(), fn.GetVersion())
	return &rpc.CreateResp{Link: link}, nil
}

func (s *server) Update(ctx context.Context, req *rpc.UpdateReq) (*rpc.UpdateResp, error) {
	fn := functionFromReq(req.GetFunction())

	if err := ds.UpdateFn(&fn); err == ErrKeyNotExists {
		return nil, grpc.Errorf(codes.NotFound, err.Error())
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}
	glog.Infof("[%s] Function updated", fn.GetName())

	link := fmt.Sprintf(codePath, fn.GetName(), fn.GetVersion())
	return &rpc.UpdateResp{Link: link}, nil
}

func (s *server) Get(ctx context.Context, req *rpc.GetReq) (*rpc.GetResp, error) {
	fn, err := ds.GetFn(req.GetName())
	if fn == nil {
		return nil, grpc.Errorf(codes.NotFound, "Not Found")
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	resp := rpc.GetResp{
		Version:  fn.GetVersion(),
		CodePath: fmt.Sprintf(codePath, fn.GetName(), fn.GetVersion()),
	}

	if !req.GetLimited() {
		resp.Function = &rpc.Function{
			Name:    fn.GetName(),
			Lang:    fn.GetLang(),
			Code:    fn.GetCode(),
			Package: fn.GetPackage(),
		}
	}
	glog.Infof("[%s] Function fetched", req.GetName())

	return &resp, nil
}

func (s *server) Delete(ctx context.Context, req *rpc.DeleteReq) (*rpc.DeleteResp, error) {
	if err := ds.DeleteFn(req.GetName()); err == ErrKeyNotExists {
		return nil, grpc.Errorf(codes.NotFound, err.Error())
	} else if err != nil {
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
	}
}
