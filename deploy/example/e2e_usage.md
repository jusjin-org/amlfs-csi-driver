## CSI driver example (TO BE UPDATED FOR AMLFS)
> refer to [driver parameters](../../docs/driver-parameters.md) for more detailed usage

### Dynamic Provisioning
#### Option#1: create storage account by CSI driver
 - Create storage class
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/amlfs-csi-driver/master/deploy/example/storageclass-amlfs.yaml
```

#### Option#2: bring your own storage account
 > only available from `v0.9.0`
 > This option does not depend on cloud provider config file, supports cross subscription and on-premise cluster scenario.
 - Use `kubectl create secret` to create `azure-secret` with existing storage account name and key
```console
kubectl create secret generic azure-secret --from-literal azurestorageaccountname=NAME --from-literal azurestorageaccountkey="KEY" --type=Opaque
```

 - create storage class referencing `azure-secret`
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/amlfs-csi-driver/master/deploy/example/storageclass-amlfs-secret.yaml
```

#### Create application
 - Create a statefulset with volume mount
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/amlfs-csi-driver/master/deploy/example/statefulset.yaml
```

 - Execute `df -h` command in the container
```console
kubectl exec -it statefulset-amlfs-0 -- df -h
```
<pre>
Filesystem      Size  Used Avail Use% Mounted on
...
amlfs         14G   41M   13G   1% /mnt/amlfs
...
</pre>

### Static Provisioning(use an existing storage account)
#### Option#1: Use storage class
> make sure cluster identity could access storage account
 - Download [amlfs storage CSI storage class](https://raw.githubusercontent.com/kubernetes-sigs/amlfs-csi-driver/master/deploy/example/storageclass-amlfs-existing-container.yaml), edit `resourceGroup`, `storageAccount`, `containerName` in storage class
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: amlfs
provisioner: amlfs.csi.azure.com
parameters:
  resourceGroup: EXISTING_RESOURCE_GROUP_NAME
  storageAccount: EXISTING_STORAGE_ACCOUNT_NAME  # cross subscription is not supported
  containerName: EXISTING_CONTAINER_NAME
reclaimPolicy: Retain  # If set as "Delete" container would be removed after pvc deletion
volumeBindingMode: Immediate
```

 - Create storage class and PVC
```console
kubectl create -f storageclass-amlfs-existing-container.yaml
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/amlfs-csi-driver/master/deploy/example/pvc-amlfs-csi.yaml
```

#### Option#2: Use secret
 - Use `kubectl create secret` to create `azure-secret` with existing storage account name and key(or sastoken)
```console
kubectl create secret generic azure-secret --from-literal azurestorageaccountname=NAME --from-literal azurestorageaccountkey="KEY" --type=Opaque
```

or create `azure-secret` with existing storage account name and sastoken:

```console
kubectl create secret generic azure-secret --from-literal azurestorageaccountname=NAME --from-literal azurestorageaccountsastoken
="sastoken" --type=Opaque
```

> storage account key(or sastoken) could also be stored in Azure Key Vault, check example here: [read-from-keyvault](../../docs/read-from-keyvault.md)

 - Create PV: download [`pv-amlfs-csi.yaml` file](https://raw.githubusercontent.com/kubernetes-sigs/amlfs-csi-driver/master/deploy/example/pv-amlfs-csi.yaml) and edit `containerName` in `volumeAttributes`
```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-amlfs
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain  # "Delete" is not supported in static provisioning
  csi:
    driver: amlfs.csi.azure.com
    readOnly: false
    volumeHandle: unique-volumeid  # make sure this volumeid is unique in the cluster
    volumeAttributes:
      containerName: EXISTING_CONTAINER_NAME
    nodeStageSecretRef:
      name: azure-secret
      namespace: default
```

 - Create PV and PVC
```console
kubectl create -f pv-amlfs-csi.yaml
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/amlfs-csi-driver/master/deploy/example/pvc-amlfs-csi-static.yaml
```

 - make sure pvc is created and in `Bound` status after a while
```console
kubectl describe pvc pvc-amlfs
```

#### create a pod with PVC mount
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/amlfs-csi-driver/master/deploy/example/nginx-pod-amlfs.yaml
```

 - Execute `df -h` command in the container
```console
kubectl exec -it nginx-amlfs -- df -h
```
<pre>
Filesystem      Size  Used Avail Use% Mounted on
...
amlfs         14G   41M   13G   1% /mnt/amlfs
...
</pre>

In the above example, there is a `/mnt/amlfs` directory mounted as `amlfs` filesystem.

#### Option#3: Inline volume
 > only available from `v1.2.0` for amlfs protocol (NFS protocol is not supported)
 - Create `azure-secret` with existing storage account name and key in the same namespace as pod
 > in below example, both secret and pod are in `default` namespace
```console
kubectl create secret generic azure-secret --from-literal azurestorageaccountname=NAME --from-literal azurestorageaccountkey="KEY" --type=Opaque
```

 - download `nginx-pod-azurefile-inline-volume.yaml` file and edit `containerName`, `secretName`
```console
wget https://raw.githubusercontent.com/kubernetes-sigs/amlfs-csi-driver/master/deploy/example/nginx-amlfs-inline-volume.yaml
#edit nginx-amlfs-inline-volume.yaml
kubectl create -f nginx-amlfs-inline-volume.yaml
```
