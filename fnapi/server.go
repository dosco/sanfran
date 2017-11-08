package main

import (
	"fmt"
	"net/http"

	"github.com/dosco/sanfran/fnapi/data"
	"github.com/dosco/sanfran/fnapi/rpc"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const CODE_LINK = "http://sanfran-fnapi-service/code/%s?v=%d"

func functionFromReq(reqFn *rpc.Function) data.Function {
	return data.Function{
		Name:    reqFn.GetName(),
		Lang:    reqFn.GetLang(),
		Code:    reqFn.GetCode(),
		Package: reqFn.GetPackage(),
	}
}

type server struct{}

func (s *server) Create(ctx context.Context, req *rpc.CreateReq) (*rpc.CreateResp, error) {
	fn := functionFromReq(req.GetFunction())

	if err := ds.CreateFn(&fn); err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	glog.Infof("[%s] Function created", fn.GetName())

	link := fmt.Sprintf(CODE_LINK, fn.GetName(), fn.GetVersion())
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

	link := fmt.Sprintf(CODE_LINK, fn.GetName(), fn.GetVersion())
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
		CodeLink: fmt.Sprintf(CODE_LINK, fn.GetName(), fn.GetVersion()),
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

func fetchCode(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	//TODO: Need version here
	name := ps.ByName("name")

	fn, err := ds.GetFn(name)
	if err != nil {
		httpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Transfer-Encoding", "binary")

	if _, err := w.Write(fn.GetCode()); err != nil {
		httpError(w, err)
		return
	}
}

func httpError(w http.ResponseWriter, err error) {
	glog.Errorln(err.Error())
	http.Error(w, err.Error(), 500)
}
