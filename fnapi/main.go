package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/dosco/sanfran/fnapi/rpc"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"google.golang.org/grpc"
)

const (
	grpcPort    = 8080
	httpPort    = 8081
	cacheExpiry = 60                // Seconds
	cacheSize   = 100 * 1024 * 1024 // Bytes
)

var (
	ds *datastore
)

func main() {
	var err error

	flag.Parse()
	defer glog.Flush()

	if ds, err = NewDatastore(cacheSize, cacheExpiry); err != nil {
		glog.Fatalln(err.Error())
	}
	defer ds.Close()

	httpS := httprouter.New()
	httpS.GET("/:name", fetchCode)

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", httpPort), httpS)
		if err != nil {
			glog.Fatalf(err.Error())
		}
	}()

	grpcS := grpc.NewServer()
	rpc.RegisterFnAPIServer(grpcS, new(server))

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort)) // RPC port
	if err != nil {
		glog.Fatalf(err.Error())
	}

	glog.Infof("SanFran/FnAPI GRPC Service Listening on :%d\n", grpcPort)
	glog.Infof("SanFran/FnAPI HTTP Service Listening on :%d\n", httpPort)

	grpcS.Serve(l)
}
