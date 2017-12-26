package main

import (
	fmt "fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	controller "github.com/dosco/sanfran/controller/rpc"
	sidecar "github.com/dosco/sanfran/sidecar/rpc"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func httpd(port int) {
	router := httprouter.New()
	router.GET("/fn/*path", execFunc)
	router.POST("/fn/*path", execFunc)
	router.PUT("/fn/*path", execFunc)
	router.HEAD("/fn/*path", execFunc)
	router.DELETE("/fn/*path", execFunc)
	router.PATCH("/fn/*path", execFunc)
	router.OPTIONS("/fn/*path", execFunc)

	glog.Infof("SanFran/Router Service, Port: %d, Namespace: %s\n",
		port, getNamespace())
	glog.Fatalln(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}

func execFunc(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fullPath := ps.ByName("path")
	if len(fullPath) == 0 {
		http.NotFound(w, r)
		return
	}
	p := strings.SplitN(fullPath[1:], "/", 2)
	name := p[0]

	var path string
	if len(p) == 2 {
		path = p[1]
	}

	conn, ok := routes.GetConn(name)
	if !ok {
		glog.Infof("Route Not Found: %s\n", name)

		resp, err := newFunctionPod(name)
		if grpc.Code(err) == codes.NotFound {
			http.NotFound(w, r)
			return
		} else if err != nil {
			panic(err.Error())
		}
		routes.AddRoute(name, resp.GetVersion(), resp.GetPodIP())
		conn, ok = routes.GetConn(name)
	}

	sc := sidecar.NewSidecarClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req, err := httpToExecuteReq(name, path, r)
	if err != nil {
		panic(err.Error())
	}

	resp, err := sc.Execute(ctx, req)
	if err != nil {
		panic(err.Error())
	}

	if err := executeRespToHttp(resp, w); err != nil {
		panic(err.Error())
	}
}

func httpToExecuteReq(name, path string, r *http.Request) (*sidecar.ExecuteReq, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	req := sidecar.ExecuteReq{
		Name:   name,
		Method: r.Method,
		Path:   path,
	}

	req.Header = make(map[string]*sidecar.ListOfString)
	for k, v := range r.Header {
		if key := strings.ToLower(k); key == "upgrade-insecure-requests" {
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
	var err error

	if resp.GetStatusCode() != 200 {
		http.Error(w, resp.GetStatus(), int(resp.GetStatusCode()))
	}

	header := w.Header()
	for k, v := range resp.Header {
		header[k] = v.Value
	}
	header["X-Powered-By"] = []string{"SanFran/Alpha"}

	if len(resp.Body) != 0 {
		_, err = w.Write(resp.Body)
	}

	return err
}

func newFunctionPod(name string) (*controller.NewFunctionPodResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := &controller.NewFunctionPodReq{Name: name}
	return controllerClient.NewFunctionPod(ctx, req)
}
