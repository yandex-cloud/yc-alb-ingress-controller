## Ingress Controller  

### Deployment  

#### Requirements

- `docker`
- `s3cmd`
- `go` >= 1.17

#### Prerequisites {#1}  
 
1. Configure container registry authentication and access. At the moment only [**Yandex Container Registry**](https://cloud.yandex.ru/docs/container-registry/) is supported    
2. Build and push ingress controller image  
```
REGISTRY_ID=<registry_id> make docker-build  
REGISTRY_ID=<registry_id> make docker-push  
```
Note: image name will be set to `cr.yandex/${REGISTRY_ID}/ingress-ctrl:$(git rev-parse HEAD)`  
3. Create an authorized service account key. [Instructions](https://cloud.yandex.com/en/docs/cli/operations/authentication/service-account).  
   This account will be used by controller to create/update balancer, it must have following roles:
   `editor` on the load-balancer(s) folder
   `certificate-manager.certificates.downloader` on the TLS certificates folder
4. Set environment variables or pass them with the Makefile commands as in the example below:  
   `FOLDER_ID` - folder for ingress controller cloud resources  
   `KEY_FILE` - path to the authorized service account key (see above)  
   `REGISTRY_ID` - container registry with controller images (see above)  
5. Install CRD into K8s cluster  
```
make install
```

**Deploy command example:**  
```
FOLDER_ID=b1gao62h0ixxxxxxxxxx KEY_FILE=${HOME}/sa/key.json REGISTRY_ID=crp3164es1xxxxxxxxxx make deploy
```

**Undeploy command example:**  
```
FOLDER_ID=b1gao62h0ixxxxxxxxxx KEY_FILE=${HOME}/sa/key.json make undeploy
```
(It will remove the ingress controller from K8s cluster and delete deployment patches made by `kustomize`)  

## FAQ

- **Error `tar: Error opening archive: Unrecognized archive format` on Macbook with m1**.

  In the generated `testbin/setup-envtest.sh` file, in the `fetch_envtest_tools` function, replace `goarch="$(go env GOARCH)"` with `goarch=amd64`.
