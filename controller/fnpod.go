package main

import (
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func newFunctionPod() *v1.Pod {
	megaBytes := int64(1024 * 1024)

	sharedVolume := v1.Volume{
		Name: "shared-data",
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{
				Medium:    v1.StorageMediumMemory,
				SizeLimit: resource.NewQuantity(10*megaBytes, resource.BinarySI),
			},
		},
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

	containerFunctionResources := v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceMemory: *resource.NewQuantity(500*megaBytes, resource.BinarySI),
			v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
		},
		Requests: v1.ResourceList{
			v1.ResourceMemory: *resource.NewQuantity(50*megaBytes, resource.BinarySI),
			v1.ResourceCPU:    *resource.NewMilliQuantity(20, resource.DecimalSI),
		},
	}

	containerFunction := v1.Container{
		Name:  "function",
		Image: getFnLangImage(),
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{Name: "shared-data", MountPath: "/shared", ReadOnly: true},
		},
		SecurityContext: &containerSecurityContext,
		Resources:       containerFunctionResources,
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
		Name:  "sidecar",
		Image: getSidecarImage(),
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{Name: "shared-data", MountPath: "/shared", ReadOnly: false},
		},
		SecurityContext: &containerSecurityContext,
		Resources:       containerSidecarResources,
	}

	labels := map[string]string{
		"app":        "sf-func",
		"release":    getHelmRelease(),
		"controller": getControllerName(),
	}

	gp := int64(0)

	functionPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "sf-pod-",
			Labels:       labels,
		},
		Spec: v1.PodSpec{
			TerminationGracePeriodSeconds: &gp,
			RestartPolicy:                 v1.RestartPolicyNever,
			Volumes:                       []v1.Volume{sharedVolume},
			Containers:                    []v1.Container{containerSidecar, containerFunction},
			SecurityContext:               &podSecurityContext,
		},
	}

	return functionPod
}

func createFunctionPod(async bool) (*v1.Pod, error) {
	functionPod := newFunctionPod()

	// If pod creation is not async then mark it as locked
	// so as to ensure its not added to the empty pods list
	// by the controllers watcher
	if !async {
		functionPod.Annotations = map[string]string{"locked": "true"}
	}

	pod, err := clientset.CoreV1().Pods(getNamespace()).Create(functionPod)
	if err != nil {
		return nil, err
	}

	if async {
		return pod, nil
	}

	var createdPod *v1.Pod

	err = wait.Poll(50*time.Millisecond, 10*time.Second, func() (bool, error) {
		createdPod, err = clientset.CoreV1().Pods(getNamespace()).
			Get(pod.Name, metav1.GetOptions{})

		return err == nil && verifyPodReady(createdPod), err
	})

	if err != nil {
		return nil, err
	}

	return createdPod, nil
}
