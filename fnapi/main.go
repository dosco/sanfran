package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/soheilhy/cmux"
	"gitlab.com/dosco/sanfran/fnapi/rpc"
	"google.golang.org/grpc"
)

const port = 8080

var ds *datastore

func main() {
	var err error

	flag.Parse()
	defer glog.Flush()

	if ds, err = NewDatastore(); err != nil {
		glog.Fatalln(err.Error())
	}
	defer ds.Close()

	grpcS := grpc.NewServer()
	rpc.RegisterFnAPIServer(grpcS, new(server))

	httpR := httprouter.New()
	httpR.GET("/code/:name", fetchCode)
	httpS := &http.Server{Handler: httpR}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) // RPC port
	if err != nil {
		glog.Fatalf(err.Error())
	}

	m := cmux.New(lis)
	grpcL := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(cmux.HTTP1Fast())

	go grpcS.Serve(grpcL)
	go httpS.Serve(httpL)

	glog.Infof("SanFran/FnAPI HTTP Service Listening on :%d\n", port)
	m.Serve()
}
