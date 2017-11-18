package main

import (
	"flag"
	"net"
	"os"

	controller "github.com/dosco/sanfran/controller/rpc"
	fnapi "github.com/dosco/sanfran/fnapi/rpc"
	"github.com/dosco/sanfran/lib/clb"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset *kubernetes.Clientset
	namespace string

	controllerClient controller.ControllerClient
	fnapiClient      fnapi.FnAPIClient
)

const port = 8080

func main() {
	var kubeconfig string

	flag.StringVar(&kubeconfig, "kubeconfig", "",
		"Path to kubeconfig containing embeded authinfo.")

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
		panic(err.Error())
	}

	if ns := os.Getenv("SANFRAN_NAMESPACE"); len(ns) != 0 {
		namespace = ns
	} else {
		namespace = v1.NamespaceDefault
	}

	fnapiClient = fnapi.NewFnAPIClient(
		clientConn("fnapi-0.sanfran-fnapi-service", "8080"))

	lb := clb.NewClb(clientset,
		[]string{"sanfran-controller:grpc"}, namespace)

	controllerClient = controller.NewControllerClient(
		lb.ClientConn("sanfran-controller"))

	watchPods()
	httpd(port)
}

func clientConn(host, port string) *grpc.ClientConn {
	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial(net.JoinHostPort(host, port), opts...)
	if err != nil {
		panic(err.Error())
	}
	return conn
}
