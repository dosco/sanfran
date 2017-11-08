package main

import (
	fmt "fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	controller "github.com/dosco/sanfran/controller/rpc"
	fnapi "github.com/dosco/sanfran/fnapi/rpc"
	sidecar "github.com/dosco/sanfran/sidecar/rpc"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func execFunc(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	name := ps.ByName("name")

	fn, err := GetFunction(name, true)
	if grpc.Code(err) == codes.NotFound {
		http.NotFound(w, r)
		return
	} else if err != nil {
		panic(err.Error())
	}

	version := fn.GetVersion()
	ip, ok := routes.GetRoute(name, version)
	if !ok {
		glog.Infof("Route Not Found: %s, %d, %s\n", name, version, ip)

		resp, err := NewFunctionPod(name)
		if err != nil {
			panic(err.Error())
		}
		routes.AddRoute(name, version, resp.GetPodIP())
		ip = resp.GetPodIP()
	}

	glog.Infof("Function Route: %s, %d, %s\n", name, version, ip)

	podHostPort := fmt.Sprintf("%s:8080", ip)
	conn, err := grpc.Dial(podHostPort, grpc.WithInsecure())
	if err != nil {
		panic(err.Error())
	}
	defer conn.Close()

	sc := sidecar.NewSidecarClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req, err := httpToExecuteReq(name, r)
	if err != nil {
		panic(err.Error())
	}

	resp, err := sc.Execute(ctx, req)
	if err != nil {
		routes.DeleteRoute(name, version, ip)
		panic(err.Error())
	}

	if err := executeRespToHttp(resp, w); err != nil {
		panic(err.Error())
	}
}

func httpToExecuteReq(name string, r *http.Request) (*sidecar.ExecuteReq, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	req := sidecar.ExecuteReq{
		Name:   name,
		Method: r.Method,
	}

	req.Header = make(map[string]*sidecar.ListOfString)
	for k, v := range r.Header {
		if key := strings.ToLower(k); key == "upgrade-insecure-requests" ||
			key == "cache-control" ||
			key == "pragma" {
			continue
		}
		req.Header[k] = &sidecar.ListOfString{Value: v}
	}
	req.Header["X-Forwarded-Host"] = &sidecar.ListOfString{Value: []string{r.Host}}

	req.Query = make(map[string]*sidecar.ListOfString)
	for k, v := range r.Form {
		req.Query[k] = &sidecar.ListOfString{Value: v}
	}

	req.Body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	defer r.Body.Close()

	return &req, nil
}

func executeRespToHttp(resp *sidecar.ExecuteResp, w http.ResponseWriter) error {
	header := w.Header()
	for k, v := range resp.Header {
		header[k] = v.Value
	}
	header["X-Powered-By"] = []string{"SanFran/Alpha"}

	_, err := w.Write(resp.Body)
	return err
}

func NewFunctionPod(name string) (*controller.NewFunctionPodResp, error) {
	conn, err := grpc.Dial("sanfran-controller-service:80", grpc.WithInsecure())
	if err != nil {
		glog.Fatalln(err.Error())
	}
	defer conn.Close()
	cc := controller.NewControllerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &controller.NewFunctionPodReq{Name: name}

	return cc.NewFunctionPod(ctx, req)
}

func GetFunction(name string, limited bool) (*fnapi.GetResp, error) {
	conn, err := grpc.Dial("sanfran-fnapi-service:80", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := fnapi.NewFnAPIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := fnapi.GetReq{Name: name, Limited: limited}
	resp, err := c.Get(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
