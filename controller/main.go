package main

import (
	"flag"
	"os"
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset *kubernetes.Clientset
	namespace string
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

	if ns := os.Getenv("SANFRAN_NAMESPACE"); len(ns) != 0 {
		namespace = ns
	} else {
		namespace = v1.NamespaceDefault
	}

	watchPods(clientset)
	autoScaler(clientset)
	initServer(port)
}

func newFunctionPod(async bool) (*v1.Pod, error) {
	sharedVolume := v1.Volume{
		Name:         "shared-data",
		VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
	}

	runAsNonRoot := true
	readOnlyRootFilesystem := true
	allowPrivilegeEscalation := false
	runAsUser := int64(65534)

	podSecurityContext := v1.PodSecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		RunAsUser:    &runAsUser,
	}

	containerSecurityContext := v1.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
	}

	megaBytes := int64(1024 * 1024)

	containerExecuteResources := v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceMemory: *resource.NewQuantity(500*megaBytes, resource.BinarySI),
			v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
		},
		Requests: v1.ResourceList{
			v1.ResourceMemory: *resource.NewQuantity(50*megaBytes, resource.BinarySI),
			v1.ResourceCPU:    *resource.NewMilliQuantity(20, resource.DecimalSI),
		},
	}

	containerExecute := v1.Container{
		Name:  "function-container",
		Image: "sanfran-base-nodejs:v1",
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{Name: "shared-data", MountPath: "/shared", ReadOnly: true},
		},
		SecurityContext: &containerSecurityContext,
		Resources:       containerExecuteResources,
	}

	containerSidecarResources := v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceMemory: *resource.NewQuantity(500*megaBytes, resource.BinarySI),
			v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
		},
		Requests: v1.ResourceList{
			v1.ResourceMemory: *resource.NewQuantity(50*megaBytes, resource.BinarySI),
			v1.ResourceCPU:    *resource.NewMilliQuantity(20, resource.DecimalSI),
		},
	}

	containerSidecar := v1.Container{
		Name:  "sidecar-container",
		Image: "sanfran-sidecar:v1",
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{Name: "shared-data", MountPath: "/shared", ReadOnly: false},
		},
		SecurityContext: &containerSecurityContext,
		Resources:       containerSidecarResources,
	}

	functionPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "sanfran-pod-",
			Labels:       map[string]string{"type": "sanfran-func"},
		},
		Spec: v1.PodSpec{
			RestartPolicy:   v1.RestartPolicyNever,
			Volumes:         []v1.Volume{sharedVolume},
			Containers:      []v1.Container{containerExecute, containerSidecar},
			SecurityContext: &podSecurityContext,
		},
	}

	// If pod creation is not async then mark it as locked
	// so as to ensure its not added to the empty pods list
	// by the controllers watcher
	if !async {
		functionPod.Annotations = map[string]string{"locked": "true"}
	}

	pod, err := clientset.CoreV1().Pods(namespace).Create(functionPod)
	if err != nil {
		return nil, err
	}

	if async {
		return pod, nil
	}

	var createdPod *v1.Pod

	err = wait.Poll(50*time.Millisecond, 15*time.Second, func() (bool, error) {
		createdPod, err = clientset.CoreV1().Pods(namespace).
			Get(pod.Name, metav1.GetOptions{})

		return err == nil && verifyPodReady(createdPod), err
	})

	if err != nil {
		return nil, err
	}

	return createdPod, nil
}
