// Just throwing some ChatGPT stuff here for now.

package main

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Kubernetes struct {
	clientset kubernetes.Interface
}

func NewKubernetes () *Kubernetes {
	// Load Kubernetes config
	kubeconfig := "./kubernetes/.kubeconfig"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig) // or use rest.InClusterConfig()
	if err != nil {
			panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
			panic(err.Error())
	}
	return &Kubernetes{
		clientset: clientset,
	}
}

func createPod(k *Kubernetes) error {
	pod := &metav1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "starpod",
			Namespace: "starfleet",
		},
		Spec: metav1.PodSpec{
			Containers: []metav1.Container{
				{
					Name:  "starpod",
					Image: "starpod:latest",
					Resources: metav1.ResourceRequirements{
						Requests: metav1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("0.5"), // 0.5 CPU core
							v1.ResourceMemory: resource.MustParse("1Gi"),
						},
						Limits: metav1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("1"), // 1 CPU core
							v1.ResourceMemory: resource.MustParse("2Gi"),
							// Not sure if this works, couldn't find good documentation on it.
							"nvidia.com/gpu": resource.MustParse("1"),
						},
					},
				},
			},
		},
	}

	_, err := k.clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	fmt.Println("Pod created successfully")
	return nil
}


