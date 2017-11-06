package main

import (
	"strconv"
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	podIndexer    cache.Indexer
	podController cache.Controller
	routes        Routes
)

func watchPods(clientset *kubernetes.Clientset) cache.Indexer {
	routes = NewRoutes()
	resyncPeriod := 30 * time.Minute

	//Setup an informer to call functions when the watchlist changes
	podIndexer, podController = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc:  listFunc,
			WatchFunc: watchFunc,
		},
		&v1.Pod{},
		resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    podAdded,
			DeleteFunc: podDeleted,
			UpdateFunc: podUpdated,
		},
		cache.Indexers{},
	)

	stop := make(chan struct{})
	go podController.Run(stop)

	return podIndexer
}

func podAdded(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return
	}

	if verifyPodReady(pod) == false {
		return
	}

	glog.Infof("[%s / %s] Pod added\n", pod.Name, pod.Status.PodIP)

	if pod.GetDeletionTimestamp() != nil {
		removePod(pod)
	} else {
		addPod(pod)
	}
}

func podDeleted(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return
	}

	glog.Infof("[%s / %s] Pod removed\n", pod.Name, pod.Status.PodIP)
	removePod(pod)
}

func podUpdated(oldObj, newObj interface{}) {
	newPod, ok := newObj.(*v1.Pod)
	if !ok {
		return
	}

	glog.Infof("[%s / %s] Pod updated\n", newPod.Name, newPod.Status.PodIP)

	if newPod.GetDeletionTimestamp() != nil {
		removePod(newPod)
	}
}

func listFunc(options metav1.ListOptions) (runtime.Object, error) {
	options.LabelSelector = "type=sanfran-func,function"
	return clientset.CoreV1().Pods(v1.NamespaceDefault).List(options)
}

func watchFunc(options metav1.ListOptions) (watch.Interface, error) {
	options.LabelSelector = "type=sanfran-func,function"
	return clientset.CoreV1().Pods(v1.NamespaceDefault).Watch(options)
}

func addPod(pod *v1.Pod) {
	name := pod.Labels["function"]

	version, err := strconv.ParseInt(pod.Annotations["version"], 10, 64)
	if err != nil {
		glog.Fatalln(err.Error())
	}

	routes.AddRoute(name, version, pod.Status.PodIP)
}

func removePod(pod *v1.Pod) {
	name := pod.Labels["function"]

	version, err := strconv.ParseInt(pod.Annotations["version"], 10, 64)
	if err != nil {
		glog.Fatalln(err.Error())
	}

	routes.DeleteRoute(name, version, pod.Status.PodIP)
}

func verifyPodReady(pod *v1.Pod) bool {
	ipAssigned := len(pod.Status.PodIP) != 0

	containersRunning := len(pod.Status.ContainerStatuses) == 2 &&
		pod.Status.ContainerStatuses[0].State.Running != nil &&
		pod.Status.ContainerStatuses[1].State.Running != nil

	return ipAssigned && containersRunning
}
