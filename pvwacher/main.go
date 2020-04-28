package main

import (
	"flag"
	"path/filepath"
	"os"
	"log"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

)

var namespace string = ""

func main()  {
	var label, field, maxClaims string
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	flag.StringVar(&namespace, "namespace", "", "namespace, for all namespaces enter nothing")
	flag.StringVar(&label, "l", "", "Label selector")
	flag.StringVar(&field, "f", "", "Field selector")
	flag.StringVar(&maxClaims, "max-claims", "200Gi", "Maximum total claims to watch")
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "kubeconfig file")
	flag.Parse()

	// total resource quantities
	var totalClaimedQuant resource.Quantity
	maxClaimedQuant := resource.MustParse(maxClaims)

	// bootstrap config
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	// create the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// initial PVCs list
	pvcList, err := clientSet.CoreV1().PersistentVolumeClaims(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: label, FieldSelector: field,
	})
	if err != nil {
		log.Fatalln("failed to get pvc:", err)
	}

	printPVCs(pvcList)

	// watch future changes to PVCs
	watch, err := clientSet.CoreV1().PersistentVolumeClaims(namespace).Watch(context.Background(), metav1.ListOptions{
		LabelSelector: label, FieldSelector: field,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("--- PVC Watch (max claims %v) ----\n", maxClaimedQuant.String())

	for event := range watch.ResultChan(){
		pvc, ok := event.Object.(*v1.PersistentVolumeClaim)
		if !ok {
			log.Fatal("unexpected type")
		}
		// Get the size of the new pvc
		quant := pvc.Spec.Resources.Requests[v1.ResourceStorage]

		switch event.Type {
			case "ADDED":
				totalClaimedQuant.Add(quant)
				log.Printf("%s PVC %s added, claim size %s\n", time.Now().Format(time.RFC850), pvc.Name, quant.String())
				if totalClaimedQuant.Cmp(maxClaimedQuant) == 1 {
					log.Printf("Claim overage reached: max %s at %s\n",
						maxClaimedQuant.String(),
						totalClaimedQuant.String(),
					)
					tempTotalClaimedQuant := totalClaimedQuant
					tempTotalClaimedQuant.Sub(maxClaimedQuant)
					log.Printf("There are more %s to remove for bing in a normal size\n", tempTotalClaimedQuant.String())
				}	
			case "MODIFIED":
				//log.Printf("%s PVC %s modified, size %s", time.Now().Format(time.RFC850), pvc.Name, quant.String())
			case "DELETED":
				totalClaimedQuant.Sub(quant)
				log.Printf("%s PVC %s removed, claim size %s", time.Now().Format(time.RFC850), pvc.Name, quant.String())

				if totalClaimedQuant.Cmp(maxClaimedQuant) <= 0 {
					log.Printf("Claim usage normal: max %s at %s",
						maxClaimedQuant.String(),
						totalClaimedQuant.String(),
					)
				}else{
					log.Printf("Claim overage reached: max %s at %s\n",
						maxClaimedQuant.String(),
						totalClaimedQuant.String(),
					)
					tempTotalClaimedQuant := totalClaimedQuant
					tempTotalClaimedQuant.Sub(maxClaimedQuant)
					log.Printf("There are more %s to remove for bing in a normal size\n", tempTotalClaimedQuant.String())
				}
	
			case "ERROR":
				log.Printf("watcher error encountered\n", pvc.Name)
				
		}
		log.Printf("\nAt %3.1f%% claim capcity (%s/%s)\n",
			float64(totalClaimedQuant.Value())/float64(maxClaimedQuant.Value())*100,
			totalClaimedQuant.String(),
			maxClaimedQuant.String(),
		)
	}
}

// Print a list of the PersistentVolumeClaims
func printPVCs(pvcs *v1.PersistentVolumeClaimList)  {
	if len(pvcs.Items) == 0{
		if namespace == ""{
			fmt.Println("No claims found in the cluster")
		}else{
			fmt.Printf("No claims found in namespace: %s\n", namespace)
		}
	}

	fmt.Println("--- PVCs ----")
	template := "%-32s%-8s%-8s\n"
	fmt.Printf(template, "NAME", "STATUS", "CAPACITY")
	totlCap := resource.NewQuantity(0, resource.BinarySI)
	for _, pvc := range pvcs.Items {
		quant := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		totlCap.Add(quant)
		fmt.Printf(template, pvc.Name, pvc.Status.Phase, quant.String())
	}
	fmt.Println("-----------------------------")
	fmt.Printf("Total capacity claimed: %s\n", totlCap.String())
	fmt.Println("-----------------------------")
	fmt.Println()
}