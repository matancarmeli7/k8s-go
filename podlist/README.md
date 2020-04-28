# podlist 

The podlist project dispalies the pods in a certian namespace.

## Build/Run

```shell
$> cd podlist
$> go build . 
```
```shell
$> ./podlist --help

Usage of ./podlist:
  -kubeconfig string
        The path to your kubeconfig (default "/root/.kube/config")
  -namespace string
        The namespace that you want to see there the pods (default "default")
...
```



