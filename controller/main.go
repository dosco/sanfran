package main

import (
	"flag"

	fnapi "github.com/dosco/sanfran/fnapi/rpc"
	"github.com/dosco/sanfran/lib/clb"
	"github.com/golang/glog"

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

	lb := clb.NewClb(clientset,
		[]string{"sanfran-fnapi:grpc", "sanfran-fnapi-cache:http"}, namespace)

	fnapiClient = fnapi.NewFnAPIClient(lb.ClientConn("sanfran-fnapi"))
	fnapiCacheLB = clb.HttpRoundRobin(lb)

	if err := fnapiCacheLB.Start("sanfran-fnapi-cache"); err != nil {
		glog.Fatalln(err.Error())
	}

	watchPods(clientset)
	autoScaler(clientset)
	grpcd(port, lb)
}
