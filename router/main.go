package main

import (
	"flag"
	"net"

	controller "github.com/dosco/sanfran/controller/rpc"
	"github.com/dosco/sanfran/lib/clb"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset        *kubernetes.Clientset
	controllerClient controller.ControllerClient
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

	clbCfg := clb.Config{
		Namespace:  getNamespace(),
		HostPrefix: getHelmRelease(),
		Services: map[string]clb.Service{
			"controller": clb.Service{Host: "sf-controller", Port: "grpc"},
		},
	}
	lb := clb.NewClb(clientset, clbCfg)

	controllerClient = controller.NewControllerClient(
		lb.ClientConn(clbCfg.Get("controller")))

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
