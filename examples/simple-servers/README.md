## Example1: simple servers

### Prerequisites  

#### Yandex Cloud Resources  
- k8s cluster from Managed Service For Kubernetes  
- registry from Container Registry to store a test app image  
- TLS certificate for your registered domain uploaded to Certificate Manager. This example for simplicity assumes this domain to be `second-server.info`.
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

### What we build  

Both test applications will be deployed from the same simple image with a primitive netcat hello-server.  
Each application will be exposed via port 8080 on its pods to its dedicated service of type `NodePort`,
which will be exposed via port 9090 to the cluster which in its turn will be exposed to the Application Load Balancer via
port 30090 for one app and 30091 for another one.
Ingress controller will use provided Ingress resources from two groups, marked "default" and, unexpectedly, "non-default",
and configure two Application Load Balancers. As a result we get a http-server (named first-server.info) residing behind 
"default" ALB and backed by both apps, and https-server ("second-server.info") residing behind "non-default" ALB and routing 
secure traffic to one of the apps.

#### Prepare test app
```shell
export TEST_IMG=cr.yandex/${REGISTRY_ID}/example-app1:latest
docker build -t ${TEST_IMG} -f ./app/TestApp.Dockerfile .
docker push ${TEST_IMG}
```

#### Deploy test app into K8s cluster

```shell
export NS_NAME=example1
export APP_PORT=8080
export SVC_PORT=9090
```
<details>
<summary>insert the corresponding values into the provided test application deployment and apply it first time using 
APP_NAME=netcat-one and NODE_PORT=30090 and second time using APP_NAME=netcat-two and NODE_PORT=30091</summary>

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: {{ NS_NAME }}-ns
spec:
  finalizers:
  - kubernetes
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ APP_NAME }}-deployment
  namespace: {{ NS_NAME }}-ns
  labels:
    app: {{ APP_NAME }}
spec:
  replicas: 3
  selector:
    matchLabels:
      app: {{ APP_NAME }}
  template:
    metadata:
      labels:
        app: {{ APP_NAME }}
    spec:
      containers:
      - name: srv
        image: {{ TEST_IMG }}
        ports:
        - containerPort: {{ APP_PORT }}
        env:
        - name: NAME
          value: "{{ APP_NAME }}"
        - name: PORT
          value: "{{ APP_PORT }}"
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "250m"
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{ APP_NAME }}
  name: {{ APP_NAME }}-service
  namespace: {{ NS_NAME }}-ns
spec:
  type: NodePort
  ports:
  - name: http
    port: {{ SVC_PORT }}
    targetPort: {{ APP_PORT }}
    protocol: TCP
    nodePort: {{ NODE_PORT }}
  selector:
    app: {{ APP_NAME }}
```

</details>

... or automate the process with the script below
```shell
# deploy the first app
cat app/testapp_tpl.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'APP_NAME=netcat-one NODE_PORT=30090 && echo "{}"' | \
kubectl apply -f -

# deploy the second app
cat app/testapp_tpl.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'APP_NAME=netcat-two NODE_PORT=30091 && echo "{}"' | \
kubectl apply -f -
```

#### Create security groups

We are going to configure "default" balancer to be created with user-defined _Security Groups_.
Overly simplistic security groups below allow:
- incoming http-traffic on port 80
- incoming http-traffic on port 443
- incoming health checks from "well-known IP ranges"
- outgoing health checks to K8s nodes on port 10256
- outgoing traffic to K8s nodes on NodePorts defined by K8s services exposing deployed apps

rules are split into two groups for demo purposes and could as well belong to the same security group
```shell
SG_1=$(yc vpc security-group create --name=sg-example-ingress --network-name ${NETWORK} --folder-id "${FOLDER_ID}" \
--rule=direction=ingress,port=80,protocol=tcp,v4-cidrs="[0.0.0.0/0]" \
--rule=direction=ingress,port=443,protocol=tcp,v4-cidrs="[0.0.0.0/0]" \
--rule=direction=ingress,port=30080,protocol=tcp,v4-cidrs="[198.18.235.0/24,198.18.248.0/24]" \
--format json | jq -r '.id')

SG_2=$(yc vpc security-group create --name=sg-example-egress --network-name ${NETWORK} --folder-id "${FOLDER_ID}" \
--rule=direction=egress,port=10256,protocol=tcp,v4-cidrs="[0.0.0.0/0]" \
--rule=direction=egress,from-port=30090,to-port=30091,protocol=tcp,v4-cidrs="[0.0.0.0/0]" \
--format json | jq -r '.id')
```

#### Create subnets

we need to define 1 to 3 subnets, one per zone where balancer will be allocated.
let's store their names in env variables SUBNET_A, SUBNET_B, SUBNET_C. Define any non-overlapping CIDRs.
```shell
zones=(a b c)
for i in "${!zones[@]}"; do
  eval SUBNET_"${zones[$i]^^}"="$(yc vpc subnet create \
    --name alb-ingress-example-subnet-${i} \
    --zone ru-central1-${zones[$i]} \
    --range 192.168.2${i}.0/24 \
    --network-name ${NETWORK} \
    --folder-id ${FOLDER_ID} \
    --format json | jq -r '.id')"
done
```

"default" balancer will be created with a user-defined address, which is a preferable way.
"non-default" one will auto-configure its address.
Note that a user-defined address allows to configure a record for a domain before a balancer exists.
```shell
ALB_IP=$(yc vpc address create \
--external-ipv4 zone=ru-central1-a \
--name "alb-ingress-example1-address" \
--description "address for ingress example" \
--folder-id "${FOLDER_ID}" \
--format json | jq -r '.external_ipv4_address.address')
```

#### Apply the ingresses

<details>
<summary>insert the corresponding values into the provided ingresses and apply them</summary>

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
    ingress.alb.yc.io/security-groups: {{ SG_1 }}
    ingress.alb.yc.io/external-ipv4-address: {{ ALB_IP }}
    ingress.alb.yc.io/request-timeout: 15s
    ingress.alb.yc.io/idle-timeout: 6m
    custom: anno1
spec:
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
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example1-ingress2
  namespace: {{ NS_NAME }}-ns
  annotations:
    ingress.alb.yc.io/group-name: default
    ingress.alb.yc.io/security-groups: {{ SG_2 }}
    ingress.alb.yc.io/request-timeout: 15s
    ingress.alb.yc.io/idle-timeout: 6m
spec:
  rules:
    - host: first-server.info
      http:
        paths:
          - path: /vamoose
            pathType: Prefix
            backend:
              service:
                name: {{ APP_NAME_2 }}-service
                port:
                  number: {{ SVC_PORT }}
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example1-ingress3
  namespace: {{ NS_NAME }}-ns
  annotations:
    ingress.alb.yc.io/group-name: non-default
    ingress.alb.yc.io/subnets: {{ SUBNET_A }},{{ SUBNET_B }}
    ingress.alb.yc.io/external-ipv4-address: auto
    ingress.alb.yc.io/request-timeout: 15s
spec:
  tls:
    - hosts:
        - second-server.info
      secretName: yc-certmgr-cert-id-{{ CERTIFICATE_ID }}
  rules:
    - host: third-server.info
      http:
        paths:
          - path: /test
            pathType: Prefix
            backend:
              service:
                name: {{ APP_NAME_1 }}-service
                port:
                  number: {{ SVC_PORT }}
---
```
</details>

... or automate the process with the script below
```shell
export SG_1 SG_2 SUBNET_A SUBNET_B SUBNET_C ALB_IP

cat ingress.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'APP_NAME_1=netcat-one APP_NAME_2=netcat-two && echo "{}"' | \
kubectl apply -f -
```
when balancers are ready the status of ingresses will contain their corresponding IP addresses.
add A-type records with these IPs to the zone of your corresponding domains 
(alternatively simply use `curl -k -H "Host: <host>" <ip>` for testing of these demo example).  
Test the connectivity. Note that http-redirect with code 301 has been automatically created for second-server.

```shell
curl http://first-server.info/go
<h1>Hello from netcat-one at netcat-one-deployment-697499b5fc-h68rn</>

curl http://first-server.info/vamoose
<h1>Hello from netcat-two at netcat-two-deployment-8d98687cc-k7pkb</>

curl https://second-server.info/test
<h1>Hello from netcat-one at netcat-one-deployment-697499b5fc-j7vts</>

curl -L http://second-server.info/test
<h1>Hello from netcat-one at netcat-one-deployment-697499b5fc-xlwzh</>
```

#### Modify the ingresses

in our example there is no real reason for splitting first-server configuration between two ingresses, so let's merge them into
one and delete the unnecessary extra ingress.

<details>
<summary> insert the corresponding values into the merged ingress...</summary>

```yaml
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example1-ingress1
  namespace: {{ NS_NAME }}-ns
  annotations:
    ingress.alb.yc.io/subnets: {{ SUBNET_A }},{{ SUBNET_B }},{{ SUBNET_C }}
    ingress.alb.yc.io/security-groups: {{ SG_1 }},{{ SG_2 }}
    ingress.alb.yc.io/external-ipv4-address: {{ ALB_IP }}
    ingress.alb.yc.io/request-timeout: 15s
    ingress.alb.yc.io/idle-timeout: 6m
    custom: anno1
spec:
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
              service:
                name: {{ APP_NAME_2 }}-service
                port:
                  number: {{ SVC_PORT }}
```
</details>

Note that duplicating data didn't change the configuration of the balancer and thus no update operations were performed.
Now we can delete example1-ingress2, again without provoking any updates.
```shell
kubectl -n ${NS_NAME}-ns delete example1-ingress2
```
Test the connectivity as above

**Migrate server to another balancer**  

Finally, let's transfer second-server to the "default" balancer. This will cause the deletion of the "non-default" one.  

<details>
<summary>insert the corresponding values into the merged ingress...</summary>

```yaml
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example1-ingress1
  namespace: {{ NS_NAME }}-ns
  annotations:
    ingress.alb.yc.io/subnets: {{ SUBNET_A }},{{ SUBNET_B }},{{ SUBNET_C }}
    ingress.alb.yc.io/security-groups: {{ SG_1 }},{{ SG_2 }}
    ingress.alb.yc.io/external-ipv4-address: {{ ALB_IP }}
    ingress.alb.yc.io/request-timeout: 15s
    ingress.alb.yc.io/idle-timeout: 6m
    custom: anno1
spec:
  tls:
    - hosts:
        - second-server.info
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
              service:
                name: {{ APP_NAME_2 }}-service
                port:
                  number: {{ SVC_PORT }}
    - host: second-server.info
      http:
        paths:
          - path: /test
            pathType: Prefix
            backend:
              service:
                name: {{ APP_NAME_1 }}-service
                port:
                  number: {{ SVC_PORT }}
---
```
</details>

Verify that "non-default" balancer is deleted.  
Test the connectivity as above, mind that second-server's IP has changed  