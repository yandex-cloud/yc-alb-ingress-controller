## Ingress Controller  
### Environment
Instead of explicitly passing variables to make targets, you can use .env files
```
cp .env.example .env
```

### Deployment  

#### Requirements

- `docker`
- `s3cmd`
- `go` >= 1.17

#### Prerequisites {#1}
 
1. Configure container registry authentication and access. At the moment only [**Yandex Container Registry**](https://cloud.yandex.ru/docs/container-registry/) is supported    
2. Build and push ingress controller image  
```
make docker-build REGISTRY_ID=<registry_id>  
make docker-push REGISTRY_ID=<registry_id> 
```
3. With ycp, create 2 ipv6 yandex-only ip addresses. One will be used for load balancer, another one will be used to communicate with dualstack cluster.
4. Create dns AAAA record for alb ip address. It is needed for grpc tests

Note: image name will be set to `cr.yandex/${REGISTRY_ID}/yc-alb-ingress-controller:$(git rev-parse HEAD)`  
5. Create an authorized service account key. [Instructions](https://cloud.yandex.com/en/docs/cli/operations/authentication/service-account).  
   This account will be used by controller to create/update balancer, it must have following roles:
   `editor` on the load-balancer(s) folder
   `certificate-manager.certificates.downloader` on the TLS certificates folder
6. Set environment variables or pass them with the Makefile commands as in the example below:
   ##### Environment variables
   `FOLDER_ID` - folder for ingress controller cloud resources  
   `KEY_FILE` - path to the authorized service account key (sa-key.json)
   `REGISTRY_ID` - container registry with controller images
7. 8.Install CRD into K8s cluster  
```
make install
```

**Deploy command example:**  
```
make deploy FOLDER_ID=b1gao62h0ixxxxxxxxxx KEY_FILE=${HOME}/sa/key.json REGISTRY_ID=crp3164es1xxxxxxxxxx
```

**Undeploy command example:**  
```
make undeploy FOLDER_ID=b1gao62h0ixxxxxxxxxx KEY_FILE=${HOME}/sa/key.json
```
(It will remove the ingress controller from K8s cluster and delete deployment patches made by `kustomize`)  

### Development

#### About generated filed and kubebuilder

