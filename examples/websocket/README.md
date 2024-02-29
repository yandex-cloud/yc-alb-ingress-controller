## Example 5: WebSocket

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

Two simple websocket hello-servers behind a balancer terminating TLS, one of the servers will also use TLS on the backend. 

#### Prepare a certificate for backends
In this example we shall use a self-signed certificate for the connection between the balancer and the backend.
Naturally, a certificate from a Trusted Authority can be used as well.
```shell
# create a cert
openssl req -x509 -batch -newkey rsa:4096 -sha256 -days 3650 -nodes -keyout app/key.pem -out app/cert.pem
# pack cert into the pkcs12 archive
openssl pkcs12 -export -passout pass: -out app/output.pkcs12 -inkey app/key.pem -in app/cert.pem
```
```shell
#Create a K8s secret for one of our apps to use
kubectl -n example5-ns create secret generic my-secret --from-file=pkcs12=app/output.pkcs12
```

#### Deploy test app into K8s cluster
Our first app will use WS protocol, thus upgrading from HTTP connection.
<details><summary>app/deployment1.yaml</summary>

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: example5-ns
spec:
  finalizers:
    - kubernetes
---
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: example5-ns
  name: example5
  labels:
    app: ws-server
    version: v1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ws-server
  template:
    metadata:
      labels:
        app: ws-server
    spec:
      terminationGracePeriodSeconds: 5
      containers:
        - name: ws-server
          image: solsson/websocat
          args:
            - -E
            - ws-listen:0.0.0.0:80
            - literalreply:"[ws] Hello from ws"
            - -v
          ports:
            - name: http
              containerPort: 80
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
  namespace: example5-ns
  name: ws-server-service
spec:
  selector:
    app: ws-server
  type: NodePort
  ports:
    - name: http
      port: 8080
      targetPort: 80
      protocol: TCP
      nodePort: 30081
```
</details>

```shell
kubectl apply -f app/deployment1.yaml
```
The second app based on the same image will upgrade to WSS from HTTPS and use the self-signed certificate from K8s
secret we created above.
<details><summary>app/deployment2.yaml</summary>

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: example5-ns
spec:
  finalizers:
    - kubernetes
---
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: example5-ns
  name: example5-wss
  labels:
    app: wss-server
    version: v1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: wss-server
  template:
    metadata:
      labels:
        app: wss-server
    spec:
      terminationGracePeriodSeconds: 5
      volumes:
        - name: example-tls-cert-pkcs12
          secret:
            secretName: example-tls-cert-pkcs12-secret
      containers:
        - name: wss-server
          image: solsson/websocat
          args:
            - wss-listen:0.0.0.0:443
            - literalreply:"[wss] Hello from secure ws"
            - --pkcs12-der
            - /certs/pkcs12
          ports:
            - name: http
              containerPort: 80
          volumeMounts:
            - name: example-tls-cert-pkcs12
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
  namespace: example5-ns
  name: wss-server-service
spec:
  selector:
    app: ws-server
  type: NodePort
  ports:
    - name: http
      port: 8080
      targetPort: 80
      protocol: TCP
      nodePort: 30082
```
</details>

```shell
kubectl apply -f app/deployment2.yaml
```

### HttpBackendGroup custom resource

An HttpBackendGroup CR allows to configure a backend group to use TLS encryption for one of our backends.

#### Deploy HttpBackendGroup CR

This configuration will be noticed by the deployed ingress-controller and a Backend Group with the S3 bucket backend
will be created.

<details>
<summary>insert the corresponding values into the provided HttpBackendGroup K8s custom resource and apply it</summary>

```yaml
apiVersion: alb.yc.io/v1alpha1
kind: HttpBackendGroup
metadata:
  namespace: example5-ns
  name: example5-bg
spec:
  backends:
    - name: wss-server
      weight: 1
      tls:
        sni: first-server.info
        trustedCa: |
          {{ CERT_CA }}
      service:
        name: wss-server-service
        port:
          number: 8080
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
  name: example1-ingress1
  namespace: {{ NS_NAME }}-ns
  annotations:
    ingress.alb.yc.io/group-name: default
    ingress.alb.yc.io/subnets: {{ SUBNET_A }},{{ SUBNET_B }},{{ SUBNET_C }}
    ingress.alb.yc.io/security-groups: {{ SG_1 }},{{ SG_2 }}
    ingress.alb.yc.io/external-ipv4-address: {{ ALB_IP }}
    ingress.alb.yc.io/request-timeout: 15s
    ingress.alb.yc.io/idle-timeout: 6m
    ingress.alb.yc.io/upgrade-types: WebSocket
spec:
  tls:
    - hosts:
        - first-server.info
      secretName: yc-certmgr-cert-id-{{ CERTIFICATE_ID }}
  rules:
    - host: first-server.info
      http:
        paths:
          - path: /go
            pathType: Prefix
            backend:
              service:
                name: {{ APP_NAME_1 }}-service
                port:
                  number: {{ SVC_PORT }}
          - path: /vamoose
            pathType: Prefix
            backend:
              resource:
                apiGroup: alb.yc.io
                kind: HttpBackendGroup
                name: example5-bg
---


```
</details>

... or automate the process with the script below
```shell
export NS_NAME=example5 APP_NAME_1=ws-server SVC_PORT=8080

cat ingress.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'echo "{}"' | \
kubectl apply -f -
```

#### Test the connectivity
Note that we declared TLS encryption in our Ingress, therefore we use WSS protocol
```shell
# establish a connection and send any message. 
# first-server.info should resolve to balancer IP, use --insecure if your cert is not from a trusted authority.

# WS backend
docker run -ti --network=host solsson/websocat  -v wss://first-server.info/go
...
[INFO  websocat::ws_client_peer] Connected to ws
Hello
"[ws] Hello from ws"
# <Ctrl-C>

# WSS backend
docker run -ti --network=host solsson/websocat  -v  --insecure wss://first-server.info/vamoose
...
[INFO  websocat::ws_client_peer] Connected to ws
Hello
"[wss] Hello from secure ws"
# <Ctrl-C>
```