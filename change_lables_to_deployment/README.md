# change_lables_to_deployment
This project add a label selector to every Deployment in a cluster. Deployment label selectors are [immutable](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#label-selector-updates). So my approach is to:
* Create a copy of each Deployment with the only difference being the name is suffixed with "-temp". This is to minimize downtime of existing Deployments.
* Delete the original Deployments.
* Recreate the original Deployments with the only difference being an additional label selector.
* Delete the temporary Deployments.

## Build/Run
```shell
$> cd change_lables_to_deployment
$> go build . 
```

```shell
$> ./change_lables_to_deployment --help --help

Usage of ./change_lables_to_deployment:
   -kubeconfig string
        The path to your kubeconfig (default "/root/.kube/config")
  -labelKeyToReplace string
        The label key of the deployments that you want to add to them the new lable (default "component")
  -labelValueToReplace string
        The label value of the deployments that you want to add to them the new lable
  -namespace string
        The namespace that you want to see there the pods (default "default")
  -newLabelKey string
        The new label key to add to the deployments (default "key")
  -newLabelValue string
        The new label value to add to the deployments (default "value")
...
```
