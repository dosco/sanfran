package main

import (
	"flag"
	fmt "fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
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
		panic(err.Error())
	}

	watchPods(clientset)

	router := httprouter.New()
	router.GET("/fn/:name", execFunc)
	router.POST("/fn/:name", execFunc)
	router.PUT("/fn/:name", execFunc)
	router.HEAD("/fn/:name", execFunc)
	router.DELETE("/fn/:name", execFunc)
	router.PATCH("/fn/:name", execFunc)
	router.OPTIONS("/fn/:name", execFunc)

	glog.Infof("SanFran/Router Service Listening on :%d\n", port)
	glog.Fatalln(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}
