package main

import (
	"flag"
	"os"
	"syscall"

	"github.com/TheCodeTeam/goodbye"
	"github.com/golang/glog"
	context "golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	port        = 8080
	cacheExpiry = 60                // Seconds
	cacheSize   = 100 * 1024 * 1024 // Bytes
)

var (
	ds        *datastore
	clientset *kubernetes.Clientset
)

func main() {
	var err error
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
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalln(err.Error())
	}

	ctx := context.Background()
	goodbye.Register(func(ctx context.Context, sig os.Signal) {
		if !goodbye.IsNormalExit(sig) {
			return
		}

		if ds != nil && sig == syscall.SIGTERM {
			ds.Close()
			os.Exit(0)
		}
	})

	if ds, err = NewDatastore(cacheSize, cacheExpiry); err != nil {
		glog.Fatalln(err.Error())
	}
	defer goodbye.Exit(ctx, -1)

	initServer(clientset, port)
}
