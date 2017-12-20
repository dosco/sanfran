package clb

import (
	"net"
	"strconv"
	"time"

	"google.golang.org/grpc/naming"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func (clb *Clb) watchPods() {
	resyncPeriod := 1 * time.Minute

	//Setup an informer to call functions when the watchlist changes
	clb.indexer, clb.controller = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc:  clb.listFunc,
			WatchFunc: clb.watchFunc,
		},
		&v1.Pod{},
		resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    clb.podAdded,
			DeleteFunc: clb.podDeleted,
			UpdateFunc: clb.podUpdated,
		},
		cache.Indexers{},
	)

	stop := make(chan struct{})
	go clb.controller.Run(stop)
}

func (clb *Clb) podAdded(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return
	}

	if verifyPodReady(pod) == false {
		return
	}

	target := pod.Labels["app"]
	addr := hostPort(pod, clb.ports[target])
	op := naming.Add

	if pod.GetDeletionTimestamp() != nil {
		op = naming.Delete
	}

	glog.Infof("[clb] [%s / %s] Pod added / deleted\n", pod.Name, pod.Status.PodIP)

	clb.updates[target] <- []*naming.Update{{Op: op, Addr: addr}}
}

func (clb *Clb) podDeleted(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return
	}

	target := pod.Labels["app"]
	addr := hostPort(pod, clb.ports[target])

	glog.Infof("[clb] [%s / %s] Pod removed\n", pod.Name, pod.Status.PodIP)

	clb.updates[target] <- []*naming.Update{{Op: naming.Delete, Addr: addr}}
}

func (clb *Clb) podUpdated(oldObj, newObj interface{}) {
	clb.podAdded(newObj)
}

func (clb *Clb) listFunc(options metav1.ListOptions) (runtime.Object, error) {
	options.LabelSelector = clb.selector
	return clb.clientset.CoreV1().Pods(clb.namespace).List(options)
}

func (clb *Clb) watchFunc(options metav1.ListOptions) (watch.Interface, error) {
	options.LabelSelector = clb.selector
	return clb.clientset.CoreV1().Pods(clb.namespace).Watch(options)
}

func verifyPodReady(pod *v1.Pod) bool {
	if len(pod.Status.PodIP) == 0 {
		return false
	}

	if len(pod.Status.ContainerStatuses) == 0 {
		return false
	}

	for i := 0; i < len(pod.Status.ContainerStatuses); i++ {
		if pod.Status.ContainerStatuses[i].State.Running == nil {
			return false
		}
	}

	return true
}

func hostPort(pod *v1.Pod, portName string) string {
	port := findPort(pod, portName)
	if len(port) == 0 {
		glog.Errorf("[clb] Unable to find a '%s' port on %s", portName, pod.Name)
		port = "8080"
	}
	return net.JoinHostPort(pod.Status.PodIP, port)
}

func findPort(pod *v1.Pod, portName string) string {
	for i := range pod.Spec.Containers {
		for _, port := range pod.Spec.Containers[i].Ports {
			if port.Name == portName {
				return strconv.Itoa(int(port.ContainerPort))
			}
		}
	}
	return ""
}
