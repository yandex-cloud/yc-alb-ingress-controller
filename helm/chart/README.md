## Install


0. create service account in yandex cloud and grant `editor` permission for the folder where ALB load balancer for cluster ingresses will be created:
```
yc iam service-account create --name k8s-alb-ingress --folder-id <FOLDER_ID>
...
id: <SERVICE_ACCOUNT_ID>

yc resource-manager folder add-access-binding --role editor --subject serviceAccount:<SERVICE_ACCOUNT_ID> <FOLDER_ID>
yc resource-manager folder add-access-binding --role certificate-manager.certificates.downloader --subject serviceAccount:<SERVICE_ACCOUNT_ID> <FOLDER_ID>
...

yc iam key create --service-account-id <SERVICE_ACCOUNT_ID> --output sa-key.json
...
```
1. create namespace

```
kubectl create namespace yc-alb-ingress

```

2. install helm chart

```shell
export VERSION=v0.1.3
export HELM_EXPERIMENTAL_OCI=1
helm pull --version ${VERSION} oci://cr.yandex/yc/yc-alb-ingress-controller-chart
helm install -n yc-alb-ingress --set folderId=<FOLDER_ID> --set clusterId=<CLUSTER_ID> yc-alb-ingress-controller --set-file saKeySecretKey=sa-key.json ./yc-alb-ingress-controller-chart-${VERSION}.tgz
```
