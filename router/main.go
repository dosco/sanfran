package main

import (
	"flag"
	"os"

	controller "github.com/dosco/sanfran/controller/rpc"
	fnapi "github.com/dosco/sanfran/fnapi/rpc"
	"github.com/dosco/sanfran/lib/clb"
	"github.com/golang/glog"
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

	lb := clb.NewClb(clientset,
		[]string{"sanfran-controller", "sanfran-fnapi"}, namespace)

	controllerClient = controller.NewControllerClient(
		lb.RoundRobinClientConn("sanfran-controller"))

	fnapiClient = fnapi.NewFnAPIClient(
		lb.RoundRobinClientConn("sanfran-fnapi"))

	watchPods()
	httpd(port)
}
