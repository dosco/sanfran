package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
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
	autoScalePods chan *v1.Pod
	podPoolSize   int
)

const (
	poolSizeConfig = "/etc/sanfran-config/controller.poolsize"
)

func autoScaler(clientset *kubernetes.Clientset) {
	podPoolSize = 3
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
	options := metav1.ListOptions{
		LabelSelector: "type=sanfran-func",
	}
	list, err := clientset.CoreV1().Pods(namespace).List(options)
	if err != nil {
		return nil, err
	}

	for i, pod := range list.Items {
		if pod.GetDeletionTimestamp() != nil {
			continue
		}
		autoScalePods <- &list.Items[i]
	}

	if i, err := strconv.Atoi(readConfig(poolSizeConfig)); err != nil {
		glog.Errorln(err.Error())
	} else {
		podPoolSize = i
	}

	queueSize := getReadyPodQueueSize()
	if queueSize < podPoolSize {
		msg := "Scaling up from %d pods (Pool Size: %d)"
		glog.Infoln(fmt.Sprintf(msg, queueSize, podPoolSize))
	}

	for i := queueSize; i < podPoolSize; i++ {
		if pod, err := newFunctionPod(true); err != nil {
			glog.Error(err.Error())
		} else {
			glog.Infof("[%s] Creating a new pod\n", pod.Name)
		}
	}

	return list, err
}

func autoScaleWorker(pods <-chan *v1.Pod) {
	for pod := range pods {
		cs := pod.Status.ContainerStatuses
		if len(cs) == 2 && (cs[0].State.Terminated != nil || cs[1].State.Terminated != nil) {
			deletePod(pod, "Terminated containers")
			continue
		}

		resp, err := fetchMetrics(pod)
		if err != nil {
			continue
		}

		if _, ok := pod.Labels["function"]; ok {
			err = functionScalingLogic(resp, pod)
		} else {
			err = podScalingLogic(resp, pod)
		}

		if err != nil {
			glog.Error(err.Error())
			continue
		}

		glog.Infof("[%s / %s] %s\n", pod.Name, pod.Status.PodIP, resp)
	}
}

func functionScalingLogic(resp *sidecar.MetricsResp, pod *v1.Pod) error {
	if resp.Terminate || resp.LastReq == 0 || resp.LastReq > 20 {
		glog.Infof("[%s / %s] Removed function label (%f)\n", pod.Name, pod.Status.PodIP, resp.LastReq)

		delete(pod.Labels, "function")
		delete(pod.Annotations, "version")

		_, err := clientset.CoreV1().Pods(namespace).Update(pod)
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

	queueSize := getReadyPodQueueSize()

	if (resp.LastReq == 0 || resp.LastReq > 300) && queueSize > podPoolSize {
		msg := "Scaling down from %d pods (Pool Size: %d)"
		return deletePod(pod, fmt.Sprintf(msg, queueSize, podPoolSize))
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
	err := clientset.CoreV1().Pods(namespace).Delete(pod.Name, options)

	glog.Infof("[%s / %s] Deleting pod (%s)\n", pod.Name, pod.Status.PodIP, reason)

	return err
}

func fetchMetrics(pod *v1.Pod) (*sidecar.MetricsResp, error) {
	podHostPort := fmt.Sprintf("%s:8080", pod.Status.PodIP)
	conn, err := grpc.Dial(podHostPort, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	sidecarClient := sidecar.NewSidecarClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := &sidecar.MetricsReq{}
	return sidecarClient.Metrics(ctx, req)
}

func getReadyPodQueueSize() int {
	mux.Lock()
	len := len(podSet)
	mux.Unlock()
	return len
}

func readConfig(fn string) string {
	if s, err := ioutil.ReadFile(fn); err != nil {
		glog.Errorln(err.Error())
	} else {
		return string(s)
	}
	return ""
}
