package main

import (
	"flag"
	"fmt"
	"mime"
	"net"
	"net/http"

	swaggerUI "github.com/dosco/sanfran/api-proxy/swagger-ui"
	"github.com/dosco/sanfran/fnapi/rpc"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	port = 8080
)

func main() {
	flag.Parse()
	defer glog.Flush()

	ctx := context.Background()
	gwmux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithInsecure()}
	fnapiHostPort := net.JoinHostPort("fnapi-0.sanfran-fnapi-service", "8080")

	err := rpc.RegisterFnAPIHandlerFromEndpoint(ctx, gwmux, fnapiHostPort, opts)
	if err != nil {
		glog.Fatalf(err.Error())
	}

	mux := http.NewServeMux()
	mux.Handle("/api/v1/fn/", gwmux)

	mux.HandleFunc("/api/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "rpc.swagger.json")
	})
	serveSwaggerUI(mux)

	glog.Infof("SanFran/API-Proxy HTTP Service Listening on :%d\n", port)

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
	if err != nil {
		glog.Fatalf(err.Error())
	}
}

func serveSwaggerUI(mux *http.ServeMux) {
	mime.AddExtensionType(".svg", "image/svg+xml")

	fileServer := http.FileServer(&assetfs.AssetFS{
		Asset:     swaggerUI.Asset,
		AssetDir:  swaggerUI.AssetDir,
		AssetInfo: swaggerUI.AssetInfo,
		Prefix:    "dist",
	})
	prefix := "/api/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
}
