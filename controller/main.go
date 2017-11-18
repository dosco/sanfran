package main

import (
	"flag"
	"net"

	fnapi "github.com/dosco/sanfran/fnapi/rpc"
	"github.com/dosco/sanfran/lib/clb"
	"github.com/golang/glog"
	grpc "google.golang.org/grpc"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset *kubernetes.Clientset
	namespace string

	fnapiClient  fnapi.FnAPIClient
	fnapiCacheLB clb.Balancer
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
	namespace = getNamespace()

	fnapiClient = fnapi.NewFnAPIClient(
		clientConn("fnapi-0.sanfran-fnapi-service", "8080"))

	lb := clb.NewClb(clientset,
		[]string{"sanfran-fnapi-cache:http"}, namespace)

	fnapiCacheLB = clb.HttpRoundRobin(lb)
	if err := fnapiCacheLB.Start("sanfran-fnapi-cache"); err != nil {
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
