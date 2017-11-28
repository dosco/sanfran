package main

import (
	"fmt"
	"net"
	"time"

	sidecar "github.com/dosco/sanfran/sidecar/rpc"
	"github.com/golang/glog"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var (
	autoScalePods      chan *v1.Pod
	podPoolSize        int
	totalReadyPodCount int
)

const (
	defaultPoolSize = 3
)

func autoScaler(clientset *kubernetes.Clientset) {
	podPoolSize = getPoolSize(defaultPoolSize)
	autoScalePods = make(chan *v1.Pod, 300)

	for w := 1; w <= 10; w++ {
		go autoScaleWorker(autoScalePods)
	}

	go func() {
		err := wait.PollInfinite(15*time.Second, func() (bool, error) {
			_, err := scalePods()
			return false, err
		})

		if err != nil {
			panic(err.Error())
		}
	}()
}

func scalePods() (*v1.PodList, error) {
	selector := "app=sf-func,controller=%s"
	options := metav1.ListOptions{
		LabelSelector: fmt.Sprintf(selector, getControllerName())}

	list, err := clientset.CoreV1().Pods(getNamespace()).List(options)
	if err != nil {
		return nil, err
	}

	totalReadyPodCount = 0
	for i := range list.Items {
		pod := &list.Items[i]

		if pod.GetDeletionTimestamp() != nil {
			continue
		}

		if !hasExitedContainers(pod) && !isActivatedPod(pod) {
			totalReadyPodCount++
		}
		autoScalePods <- &list.Items[i]
	}

	podPoolSize = getPoolSize(defaultPoolSize)

	if totalReadyPodCount < podPoolSize {
		msg := "Scaling up from %d pods (Pool Size: %d)"
		glog.Infoln(fmt.Sprintf(msg, totalReadyPodCount, podPoolSize))
	}

	for i := totalReadyPodCount; i < podPoolSize; i++ {
		if pod, err := createFunctionPod(true); err != nil {
			glog.Error(err.Error())
		} else {
			glog.Infof("[%s] Creating a new pod\n", pod.Name)
		}
	}

	return list, err
}

func autoScaleWorker(pods <-chan *v1.Pod) {
	for pod := range pods {
		if hasExitedContainers(pod) {
			deletePod(pod, "Terminated containers")
			continue
		}

		resp, err := fetchMetrics(pod)
		if err != nil {
			glog.Warning(err.Error())
			continue
		}

		if isActivatedPod(pod) {
			err = functionScalingLogic(resp, pod)
		} else {
			err = podScalingLogic(resp, pod)
		}

		if err != nil {
			glog.Error(err.Error())
		}

		glog.Infof("[%s / %s] %s\n", pod.Name, pod.Status.PodIP, resp)
	}
}

func functionScalingLogic(resp *sidecar.MetricsResp, pod *v1.Pod) error {
	if resp.Terminate || resp.LastReq == 0 || resp.LastReq > 20 {
		glog.Infof("[%s / %s] Removed function label (%f)\n", pod.Name, pod.Status.PodIP, resp.LastReq)

		delete(pod.Labels, "function")
		delete(pod.Annotations, "version")

		_, err := clientset.CoreV1().Pods(getNamespace()).Update(pod)
		if err != nil {
			return err
		}
	}

	return nil
}

func podScalingLogic(resp *sidecar.MetricsResp, pod *v1.Pod) error {
	if resp.Terminate {
		return deletePod(pod, "Marked for termination")
	}

	if (resp.LastReq == 0 || resp.LastReq > 300) && totalReadyPodCount > podPoolSize {
		msg := "Scaling down from %d pods (Pool Size: %d)"
		return deletePod(pod, fmt.Sprintf(msg, totalReadyPodCount, podPoolSize))
	}

	return nil
}

func deletePod(pod *v1.Pod, reason string) error {
	mux.Lock()
	if _, ok := podSet[pod.Name]; ok {
		delete(podSet, pod.Name)
	}
	mux.Unlock()

	options := &metav1.DeleteOptions{}
	err := clientset.CoreV1().Pods(getNamespace()).Delete(pod.Name, options)

	glog.Infof("[%s / %s] Deleting pod (%s)\n", pod.Name, pod.Status.PodIP, reason)

	return err
}

func fetchMetrics(pod *v1.Pod) (*sidecar.MetricsResp, error) {
	hostPort := net.JoinHostPort(pod.Status.PodIP, "8080")
	conn, err := grpc.Dial(hostPort, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	sidecarClient := sidecar.NewSidecarClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req := &sidecar.MetricsReq{FromController: true}
	return sidecarClient.Metrics(ctx, req)
}

func hasExitedContainers(pod *v1.Pod) bool {
	cs := pod.Status.ContainerStatuses

	return len(cs) != 2 ||
		cs[0].State.Terminated != nil ||
		cs[1].State.Terminated != nil
}

func isActivatedPod(pod *v1.Pod) bool {
	_, ok := pod.Labels["function"]
	return ok
}
