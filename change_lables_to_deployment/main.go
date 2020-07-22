package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const tempLabelKey = "temporary"
const tempSuffix = "-temp"

func main() {
	var namespace, labelKeyToReplace, labelValueToReplace, labelSelector, newLabelKey, newLabelValue string
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	flag.StringVar(&namespace, "namespace", "default", "The namespace that you want to see there the pods")
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "The path to your kubeconfig")
	flag.StringVar(&labelKeyToReplace, "labelKeyToReplace", "component", "The label key of the deployments that you want to add to them the new lable")
	flag.StringVar(&labelValueToReplace, "labelValueToReplace", "", "The label value of the deployments that you want to add to them the new lable")
	flag.StringVar(&newLabelKey, "newLabelKey", "key", "The new label key to add to the deployments")
	flag.StringVar(&newLabelValue, "newLabelValue", "value", "The new label value to add to the deployments")
	flag.Parse()

	log.Println("Using kubeconfig file: ", kubeconfig)

	// bootstrap config
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	exitOnErr(err)

	// create the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	exitOnErr(err)

	if labelValueToReplace != "" {
		labelSelector = fmt.Sprintf("%s=%s", labelKeyToReplace, labelValueToReplace)
	} else {
		labelSelector = fmt.Sprintf("%s", labelKeyToReplace)
	}

	deployments, err := clientSet.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	exitOnErr(err)

	if len(deployments.Items) == 0 {
		log.Fatal("There is no available deployments")
	}

	tempDeployments := createNewDeploymentsWithNewLables(deployments, tempSuffix, tempLabelKey, "true")

	factory := informers.NewSharedInformerFactory(clientSet, 0)
	informer := factory.Apps().V1().Deployments().Informer()
	stopper := make(chan struct{})
	defer close(stopper)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*v1.Deployment)
			labels := deployment.GetLabels()

			if _, ok := labels[tempLabelKey]; ok {
				deploymentToDelete := strings.Replace(deployment.GetName(), tempSuffix, "", -1)
				deleteDeployment(namespace, deploymentToDelete, clientSet)
			} else if labelVal, ok := labels[newLabelKey]; ok && labelVal == newLabelValue {
				deploymentToDelete := deployment.GetName() + tempSuffix
				deleteDeployment(namespace, deploymentToDelete, clientSet)
			}

		},
		DeleteFunc: func(obj interface{}) {
			deployment := obj.(*v1.Deployment)
			labels := deployment.GetLabels()

			if _, ok := labels[tempLabelKey]; !ok {
				if _, ok := labels[newLabelKey]; !ok {
					deploymentToCreate := processDeployments(*deployment)
					deploymentToCreate = createDeploymentWithNewLabel(newLabelKey, newLabelValue, deploymentToCreate)
					createDeploymentOnAPI(deploymentToCreate, clientSet, namespace)
				}
			}
		},
	})

	createDeploymentsOnAPI(tempDeployments, clientSet, namespace)

	informer.Run(stopper)
}

func exitOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func createNewDeploymentsWithNewLables(deployments *v1.DeploymentList, tempSuffix, labelKey, labelValue string) *v1.DeploymentList {
	newDeployments := &v1.DeploymentList{}
	for _, deployment := range deployments.Items {
		deployment = processDeployments(deployment)
		deployment = appendToDeploymentName(deployment, tempSuffix)
		deployment = createDeploymentWithNewLabel(labelKey, labelValue, deployment)
		newDeployments.Items = append(newDeployments.Items, deployment)
	}
	return newDeployments
}

func processDeployments(deployment v1.Deployment) v1.Deployment {
	deployment.Status = v1.DeploymentStatus{}
	deployment.SetUID(types.UID(""))
	deployment.SetSelfLink("")
	deployment.SetGeneration(0)
	deployment.SetCreationTimestamp(metav1.Now())
	deployment.SetResourceVersion("")
	return deployment
}

func appendToDeploymentName(deployment v1.Deployment, tempSuffix string) v1.Deployment {
	deployment.SetName(fmt.Sprintf("%s%s", deployment.GetName(), tempSuffix))
	return deployment
}

func createDeploymentWithNewLabel(labelKey string, labelValue string, deployment v1.Deployment) v1.Deployment {
	newDeployment := deployment.DeepCopy()
	labels := newDeployment.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
		newDeployment.SetLabels(labels)
	}
	labels[labelKey] = labelValue

	podTemplateSpecLabels := newDeployment.Spec.Template.GetLabels()
	if podTemplateSpecLabels == nil {
		podTemplateSpecLabels = make(map[string]string)
		newDeployment.Spec.Template.SetLabels(podTemplateSpecLabels)
	}
	podTemplateSpecLabels[labelKey] = labelValue

	labelSelectors := newDeployment.Spec.Selector.MatchLabels
	if labelSelectors == nil {
		labelSelectors = make(map[string]string)
		newDeployment.Spec.Selector.MatchLabels = labelSelectors
	}
	labelSelectors[labelKey] = labelValue
	return *newDeployment
}

func createDeploymentOnAPI(deployment v1.Deployment, clientSet *kubernetes.Clientset, namespace string) {
	_, err := clientSet.AppsV1().Deployments(namespace).Update(context.Background(), &deployment, metav1.UpdateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			_, createErr := clientSet.AppsV1().Deployments(namespace).Create(context.Background(), &deployment, metav1.CreateOptions{})
			exitOnErr(createErr)
		} else {
			exitOnErr(err)
		}
	}
}

func createDeploymentsOnAPI(deployments *v1.DeploymentList, clientSet *kubernetes.Clientset, namespace string) {
	for _, deployment := range deployments.Items {
		createDeploymentOnAPI(deployment, clientSet, namespace)
	}
}

func deleteDeployment(namespace string, name string, client *kubernetes.Clientset) {
	err := client.AppsV1().Deployments(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	exitOnErr(err)
}
