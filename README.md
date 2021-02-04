# KUBEVIRT-CSI-DRIVER-OPERATOR
Operator for deploying kubevirt/csi-driver in OCP cluster
## Introduction
The kubevirt/csi-driver is intended for clusters installed over Kubevirt/CNV.
Such a cluster uses Kubevirt VMs as its nodes.
## Deployment steps
These are the steps performed by the operator for deploying the driver.
* Automatically set parameters for the driver. E.g. namespace name of infra cluster
* Create ConfigMap for driver
* Create a CredentialsRequest. This step relies on cloud-credentials-operator. The result is a secret in the driver's namespace that allows it to operate in the infra cluster (cluster where Kubevirt is deployed).
* Deploy all driver YAMLs
* Attempt to create a StorageClass
  * Name is 'kubevirt-csi-driver'
  * Provisioner is 'csi.kubevirt.io'
  * The StorageClass requires a parameter with infra cluster's storage class name. These are the steps taken by the operator for determining the infra's storage class name:
    * Try kube-system/ConfigMap/cluster-config-v1
    * Try openshift-machine-api/MachineSets. Use first in the list.
    * If not found then log warning and continue without creating the StorageClass.
## Installation
* Deploy the files in folder `manifests`
* Perform post deployment steps as described in the [driver's README](https://github.com/kubevirt/csi-driver):
  * Create StorageClass (if not created by operator)
  * Create PVCs
  * Set feature gates
## Development
* Build the operator
```
make build
```
* Create image. Use environment variable IMAGE_REF for setting the image tag.
```
make image
```
* Push image
```
make push
```
