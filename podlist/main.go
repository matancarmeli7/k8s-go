package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"golang.org/x/net/context"
)
func main(){
	var namespace string
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	flag.StringVar(&namespace, "namespace", "default", "The namespace that you want to see there the pods")
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "The path to your kubeconfig")
	flag.Parse()

	log.Println("Using kubeconfig file: ", kubeconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	pods, err := clientSet.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalln("failed to get pods:", err)
	}

	for i, pod := range pods.Items {
		fmt.Printf("[%d] %s\n", i, pod.GetName())
	}

}
