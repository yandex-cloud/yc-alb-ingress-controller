## Example 4: TLS backend

### Prerequisites ###  

#### Yandex Cloud Resources
- k8s cluster from Managed Service For Kubernetes
- TLS certificate for your registered domain uploaded to Certificate Manager. This example for simplicity assumes this domain to be `first-server.info`.
  (naturally, for the example you can also use a self-signed certificate and any other imaginary domain)

#### Tools  
`yc` configured to work with your cloud  
`kubectl` configured for your K8s cluster  
`docker` and `helm` configured and authenticated for Yandex Container Registry

For convenience, throughout the example we'll export all variable configuration data into the environment variables
```shell
export FOLDER_ID=<your folder ID>
export REGISTRY_ID=<your registry ID>
export NETWORK=<your network name>
export CERTIFICATE_ID=<your certificate ID>
```

#### Deploy the controller
as described [here](../../helm/yc-alb-ingress-controller/README.md).

This bootstrapping process is described in more details in the [official documentation](https://cloud.yandex.ru/docs/managed-kubernetes/solutions/alb-ingress-controller)

By this stage you should have:
- running K8s cluster
- configured tools: yc, kubectl, helm
- ingress-controller deployed into K8s cluster
- cloud resources: subnets, security groups, an address for balancer and a certificate
  for a domain which this example for simplicity assumes to be `first-server.info`.
- variables set: FOLDER_ID NETWORK SUBNET_A SUBNET_B SUBNET_C NS_NAME ALB_IP SG_1 SG_2

### What we build ###

A http2 web server behind an ALB will be created, where the traffic between the balancer and the backends is TLS-encrypted.

#### Prepare a certificate for backends
In our example, for a change, we shall use a self-signed certificate for the connection between the balancer and the backend.
Naturally, a certificate from a Trusted Authority can be used as well. 
```shell
# create a self-signed cert
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
-keyout server.key -out server.crt -subj "/CN=first-server.info/C=RU/L=St.Petersburg/O=Yandex/OU=Yandex.Cloud" \
-addext "subjectAltName=DNS:first-server.info"
```

```shell
kubectl create namespace example4-ns
# load cert into K8s secret
kubectl -n example4-ns create secret tls example-tls-cert-secret --cert=server.crt --key=server.key
```

Apply the K8s resources configuration below to create a new namespace and deploy a http2 server application and s service. 
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: example4-ns
spec:
  finalizers:
    - kubernetes
---
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: example4-ns
  name: go-http2-server
data:
  server.go: |+
    package main

    import (
      "log"
      "net/http"
    )

    func main() {
      srv := &http.Server{Addr: ":443", Handler: http.HandlerFunc(handle)}
      log.Printf("Serving on https://0.0.0.0:443")
      log.Fatal(srv.ListenAndServeTLS("/certs/tls.crt", "/certs/tls.key"))
    }

    func handle(w http.ResponseWriter, r *http.Request) {
      _, _ = w.Write([]byte("Hello " + r.Proto + " client"))
    }

---
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: example4-ns
  name: example4
  labels:
    app: alb-demo-1
    version: v1
spec:
  replicas: 2
  selector:
    matchLabels:
      app: http2-server
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: http2-server
        version: v1
    spec:
      terminationGracePeriodSeconds: 5
      volumes:
        - name: srv
          configMap:
            name: go-http2-server
        - name: example-tls-cert
          secret:
            secretName: example-tls-cert-secret
      containers:
        - name: http2-server
          image: golang:1.16
          command:
            - go
          args:
            - run
            - server.go
          workingDir: /app
          ports:
            - name: http
              containerPort: 443
          volumeMounts:
            - name: srv
              mountPath: /app
            - name: example-tls-cert
              mountPath: /certs
              readOnly: true
          resources:
            limits:
              cpu: 250m
              memory: 128Mi
            requests:
              cpu: 100m
              memory: 64Mi
---
apiVersion: v1
kind: Service
metadata:
  namespace: example4-ns
  name: http2-server-service
spec:
  selector:
    app: http2-server
  type: NodePort
  ports:
    - name: http
      port: 8443
      targetPort: 443
      protocol: TCP
      nodePort: 30090
```

### HttpBackendGroup custom resource

An HttpBackendGroup CR allows to configure a backend group to use `http2` and TLS encryption.

#### Deploy HttpBackendGroup CR

This configuration will be noticed by the deployed ingress-controller and a Backend Group with the S3 bucket backend
will be created.

<details>
<summary>insert the corresponding values into the provided HttpBackendGroup K8s custom resource and apply it</summary>

```yaml
---
apiVersion: alb.yc.io/v1alpha1
kind: HttpBackendGroup
metadata:
  namespace: {{ NS_NAME }}-ns
  name: example4-bg
spec:
  backends:
    - name: http2-server
      weight: 1
      useHttp2: true
      tls:
        sni: first-server.info
        trustedCa: |
          {{ CERT_CA }}
      service:
        name: http2-server-service
        port:
          number: 8443
---
```
</details>

... or automate the process with the script below
```shell
export CERT_CA=$(sed '2,$s/^/          /g' server.crt)
cat backendgroup.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'echo "{}"' | \
kubectl apply -f -
```

#### Prepare resources to be used by balancer
Create [subnets](../simple-servers/README.md#create-subnets) and [security groups](../simple-servers/README.md#create-security-groups)
as described in the corresponding sections of [Example 1](../simple-servers/README.md)

#### Apply the ingress
<details>
<summary>insert the corresponding values into the provided Ingress resource and apply it</summary>

```yaml
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example4-ingress1
  namespace: {{ NS_NAME }}-ns
  annotations:
    ingress.alb.yc.io/group-name: http2-server
    ingress.alb.yc.io/subnets: {{ SUBNET_A }},{{ SUBNET_B }},{{ SUBNET_C }}
    ingress.alb.yc.io/security-groups: {{ SG_1 }},{{ SG_2 }}
    ingress.alb.yc.io/external-ipv4-address: {{ ALB_IP }}
    ingress.alb.yc.io/request-timeout: 15s
    ingress.alb.yc.io/idle-timeout: 6m
spec:
  rules:
    - host: first-server.info
      http:
        paths:
          - path: /proceed
            pathType: Exact
            backend:
              resource:
                apiGroup: alb.yc.io
                kind: HttpBackendGroup
                name: example4-bg
---

```
</details>

... or automate the process with the script below
```shell
export NS_NAME=example4

cat ingress.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'echo "{}"' | \
kubectl apply -f -
```

#### Test the connectivity
```shell
curl --http2 http://first-server.info/proceed
Hello HTTP/2.0 client
```

#### Add TLS on the balancer
We started with a plain http listener on the balancer only to avoid confusion. Of course, normally the external traffic
should be encrypted with a trusted certificate. Assuming, you have it in the Certificate Manager with id stored in
CERTIFICATE_ID, adding it is as simple as adding the familiar `tls` section to the ingress spec:

by a patch command
```shell
kubectl -n ${NS_NAME}-ns patch ingress example4-ingress1 -p \
'{"spec": {"tls": [{"hosts": ["first-server.info"], "secretName": "yc-certmgr-cert-id-'${CERTIFICATE_ID}'"}]}}'
```

or manual edit
```yaml
spec:
  tls:
  - hosts:
    - first-server.info
    secretName: yc-certmgr-cert-id-{{ CERTIFICATE_ID }}
```

Test the secure connectivity
```shell
curl --http2 https://first-server.info/proceed
Hello HTTP/2.0 client

# redirect
curl --http2 http://first-server.info/proceed -w '%{http_code} --> %{redirect_url}'
301 --> https://first-server.info:443/proceed
```