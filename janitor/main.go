package main

import (
	"flag"

	"github.com/golang/glog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset *kubernetes.Clientset
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

	cleanup(clientset)
}
