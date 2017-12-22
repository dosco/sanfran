package main

import (
	"flag"
	"fmt"
	"net"

	fnapi "github.com/dosco/sanfran/fnapi/rpc"
	"github.com/dosco/sanfran/lib/clb"
	"github.com/golang/glog"
	grpc "google.golang.org/grpc"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset   *kubernetes.Clientset
	fnapiClient fnapi.FnAPIClient
	fncacheLB   clb.Balancer
)

const port = 8080

func main() {
	var kubeconfig string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig containing embeded authinfo.")
	flag.Parse()
	defer glog.Flush()

	if len(kubeconfig) != 0 {
		glog.Infoln("Using kubeconfig: ", kubeconfig)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalln(err.Error())
	}

	host := "%s-sf-fnapi-0.%s-sf-fnapi"
	host = fmt.Sprintf(host, getHelmRelease(), getHelmRelease())
	fnapiClient = fnapi.NewFnAPIClient(clientConn(host, "8080"))

	clbCfg := clb.Config{
		Namespace:  getNamespace(),
		HostPrefix: getHelmRelease(),
		Services: map[string]clb.Service{
			"fncache": clb.Service{Host: "sf-fncache", Port: "http"},
		},
	}
	lb := clb.NewClb(clientset, clbCfg)

	fncacheLB = clb.HttpRoundRobin(lb)
	if err := fncacheLB.Start(clbCfg.Get("fncache")); err != nil {
		glog.Fatalln(err.Error())
	}

	watchPods(clientset)
	autoScaler(clientset)
	grpcd(port, lb)
}

func clientConn(host, port string) *grpc.ClientConn {
	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial(net.JoinHostPort(host, port), opts...)
	if err != nil {
		panic(err.Error())
	}
	return conn
}
