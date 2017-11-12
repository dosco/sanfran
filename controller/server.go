package main

import (
	"fmt"
	"net"
	"strconv"
	"time"

	fnapi "github.com/dosco/sanfran/fnapi/rpc"
	"github.com/golang/glog"

	controller "github.com/dosco/sanfran/controller/rpc"
	sidecar "github.com/dosco/sanfran/sidecar/rpc"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
)

type server struct{}

func initServer(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) // RPC port
	if err != nil {
		glog.Fatalln(err.Error())
	}
	g := grpc.NewServer()
	controller.RegisterControllerServer(g, new(server))

	glog.Infof("SanFran/Controller Service, Port: %d, Namespace: %s\n", port, namespace)
	glog.Infof("Name: %s, UID: %s\n", getControllerName(), getControllerUID())

	g.Serve(lis)
}

func (s *server) NewFunctionPod(ctx context.Context, req *controller.NewFunctionPodReq) (*controller.NewFunctionPodResp, error) {
	var err error

	name := req.GetName()
	if len(name) == 0 {
		return nil, fmt.Errorf("No 'name' specified")
	}

	fn, err := getFunction(name, true)
	if err != nil {
		return nil, err
	}

	version := strconv.FormatInt(fn.GetVersion(), 10)
	codeLink := fn.GetCodeLink()
	pod := getNextPod()

	if pod == nil {
		if pod, err = createFunctionPod(false); err != nil {
			return nil, err
		}
		glog.Infof("[%s / %s / %s:%s] Creating a new pod\n", pod.Name,
			pod.Status.PodIP, name, version)
	} else {
		glog.Infof("[%s / %s / %s:%s] Found a pod to use\n", pod.Name,
			pod.Status.PodIP, name, version)
	}

	pod, err = activateFunctionPod(name, version, codeLink, pod)
	if err != nil {
		return nil, err
	}

	glog.Infof("[%s / %s / %s:%s] Activated pod\n", pod.Name,
		pod.Status.PodIP, name, version)

	glog.Flush()

	return &controller.NewFunctionPodResp{
		PodName: pod.Name,
		PodIP:   pod.Status.PodIP,
	}, nil
}

func activateFunctionPod(name, version, codeLink string, pod *v1.Pod) (*v1.Pod, error) {
	podHostPort := fmt.Sprintf("%s:8080", pod.Status.PodIP)
	conn, err := grpc.Dial(podHostPort, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	sidecarClient := sidecar.NewSidecarClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req := sidecar.ActivateReq{Link: codeLink}
	_, activateErr := sidecarClient.Activate(ctx, &req)

	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	if _, ok := pod.Annotations["locked"]; ok {
		delete(pod.Annotations, "locked")
	}

	pod.Annotations["version"] = version
	pod.Labels["function"] = name

	updatedPod, err := clientset.CoreV1().Pods(namespace).Update(pod)
	if err != nil {
		return nil, err
	}

	if activateErr != nil {
		return nil, activateErr
	}

	return updatedPod, nil
}

func getFunction(name string, limited bool) (*fnapi.GetResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	req := fnapi.GetReq{Name: name, Limited: limited}
	return fnapiClient.Get(ctx, &req)
}
