# pcwatcher 

The pvwatcher project is a project which watches the total capacity of the persistent volumes claim in a kubernetes cluster or in a specific namespace in the cluster. It checks if it doesnt pass the max capacity that the user chose.

## Build/Run

```shell
$> cd pvwatcher
$> go build . 
```
```shell
$> ./pvcwatch --help

Usage of ./pvcwatch:
  -f string
    	Field selector
  -kubeconfig string
    	kubeconfig file (default "/Users/<username>/.kube/config")
  -l string
    	Label selector
  -max-claims string
    	Maximum total claims to watch (default "200Gi")
  -namespace string
    	namespace
...
```



