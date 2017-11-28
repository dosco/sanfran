package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	sidecar "github.com/dosco/sanfran/sidecar/rpc"
	"github.com/golang/glog"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	orphanPods chan *v1.Pod
)

const orphanAfterPingGap = 900

func cleanup(clientset *kubernetes.Clientset) {
	var wg sync.WaitGroup
	orphanPods = make(chan *v1.Pod, 300)

	for w := 1; w <= 10; w++ {
		go cleanupWorker(&wg, orphanPods)
	}

	if err := findOphanPods(&wg); err != nil {
		glog.Fatal(err.Error())
	}

	wg.Wait()
	glog.Info("I'm done thanks, goodbye!")
}

func findOphanPods(wg *sync.WaitGroup) error {
	options := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s-sf-controller", getHelmRelease())}

	list, err := clientset.CoreV1().Pods(getNamespace()).List(options)
	if err != nil {
		return err
	}

	var ac []string
	for i := range list.Items {
		pod := list.Items[i]

		if pod.GetDeletionTimestamp() != nil {
			continue
		}
		ac = append(ac, pod.Name)
	}

	podSel := fmt.Sprintf("app=sf-func,release=%s,controller notin (%s)",
		getHelmRelease(), strings.Join(ac, ","))

	glog.Infof("Orphan Pod Selector: %s\n", podSel)

	options = metav1.ListOptions{LabelSelector: podSel}
	list, err = clientset.CoreV1().Pods(getNamespace()).List(options)
	if err != nil {
		return err
	}

	glog.Infof("Found %d orphans\n", len(list.Items))
	wg.Add(len(list.Items))

	for i := range list.Items {
		orphanPods <- &list.Items[i]
	}

	return nil
}

func cleanupWorker(wg *sync.WaitGroup, pods <-chan *v1.Pod) {
	for pod := range pods {
		delete := hasExitedContainers(pod)

		if !delete {
			resp, err := fetchMetrics(pod)
			if err != nil {
				glog.Errorln(err.Error())
			}
			delete = resp == nil ||
				(resp.LastPing > orphanAfterPingGap || resp.Terminate)

			glog.Infof("[%s] Pod Metrics: %s\n", pod.Name, resp)
		}

		if delete {
			if err := deletePod(pod); err != nil {
				glog.Errorln(err.Error())
			}
		}

		wg.Done()
	}
}

func hasExitedContainers(pod *v1.Pod) bool {
	cs := pod.Status.ContainerStatuses

	return len(cs) != 2 ||
		cs[0].State.Terminated != nil ||
		cs[1].State.Terminated != nil
}

func deletePod(pod *v1.Pod) error {
	options := &metav1.DeleteOptions{}
	err := clientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, options)
	if err != nil {
		return err
	}

	glog.Infof("[%s / %s] Deleting pod\n", pod.Name, pod.Status.PodIP)
	return nil
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

	req := &sidecar.MetricsReq{FromController: false}
	return sidecarClient.Metrics(ctx, req)
}

func getNamespace() string {
	if v := os.Getenv("SANFRAN_NAMESPACE"); len(v) != 0 {
		return v
	}
	return v1.NamespaceDefault
}

func getEnv(name string, required bool) string {
	if v := os.Getenv(name); len(v) != 0 {
		return v
	}
	if required {
		glog.Fatalln(fmt.Errorf("%s not defined", name))
	}
	return ""
}

func getHelmRelease() string {
	return getEnv("SANFRAN_HELM_RELEASE", true)
}
