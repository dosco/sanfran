package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	podIndexer    cache.Indexer
	podController cache.Controller
	podSet        map[string]struct{}
	mux           sync.Mutex
)

const POD_QUEUE_SIZE int = 10000

func watchPods(clientset *kubernetes.Clientset) cache.Indexer {
	podSet = make(map[string]struct{})

	//Define what we want to look for (Pods)
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
			AddFunc:    process,
			DeleteFunc: process,
			UpdateFunc: func(oldObj, newObj interface{}) {
				process(newObj)
			},
		},
		cache.Indexers{"name": nameIndexFunc},
	)

	stop := make(chan struct{})
	go podController.Run(stop)

	return podIndexer
}

func nameIndexFunc(obj interface{}) ([]string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %v", err)
	}
	return []string{meta.GetName()}, nil
}

func process(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return
	}

	if verifyPodReady(pod) == false {
		return
	}

	if pod.Annotations != nil {
		_, locked := pod.Annotations["locked"]
		aliveFor := time.Now().Sub(pod.GetCreationTimestamp().Time)

		if locked && aliveFor > 1*time.Minute {
			delete(pod.Annotations, "locked")
		}
	}

	mux.Lock()
	if pod.GetDeletionTimestamp() != nil {
		if _, ok := podSet[pod.Name]; ok {
			delete(podSet, pod.Name)
		}
	} else {
		podSet[pod.Name] = struct{}{}
	}
	mux.Unlock()
}

func listFunc(options metav1.ListOptions) (runtime.Object, error) {
	selector := "app=sf-func,controller=%s,!function"
	options.LabelSelector = fmt.Sprintf(selector, getControllerName())

	return clientset.CoreV1().Pods(getNamespace()).List(options)
}

func watchFunc(options metav1.ListOptions) (watch.Interface, error) {
	selector := "app=sf-func,controller=%s,!function"
	options.LabelSelector = fmt.Sprintf(selector, getControllerName())

	return clientset.CoreV1().Pods(getNamespace()).Watch(options)
}

func verifyPodReady(pod *v1.Pod) bool {
	var ipAssigned, containersRunning bool

	ipAssigned = len(pod.Status.PodIP) != 0

	containersRunning = len(pod.Status.ContainerStatuses) == 2 &&
		pod.Status.ContainerStatuses[0].State.Running != nil &&
		pod.Status.ContainerStatuses[1].State.Running != nil

	return ipAssigned && containersRunning
}

func getNextPod() *v1.Pod {
	mux.Lock()
	defer mux.Unlock()

	for podName, _ := range podSet {
		if _, ok := podSet[podName]; !ok {
			continue
		}

		objList, err := podIndexer.ByIndex("name", podName)
		if err != nil {
			glog.Errorln(err.Error())
			continue
		}

		if objList == nil || len(objList) == 0 {
			continue
		}

		if pod, ok := objList[0].(*v1.Pod); ok {
			delete(podSet, podName)
			return pod
		}
	}

	return nil
}
