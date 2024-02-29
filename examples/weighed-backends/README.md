## Example 2: weighed backends ##

### Prerequisites ###

This example re-uses the prerequisites of [Example1](../simple-servers/README.md)
Follow the steps of Example 1 until **Apply the ingresses** step, except that the Certificate won't be needed.

By this stage you should have:
- running K8s cluster
- configured tools: yc, kubectl, helm
- deployed K8s resources: services netcat-one-service and netcat-two-service backed up by their corresponding apps
- ingress-controller deployed into K8s cluster
- cloud resources: subnets, security groups and an address for balancer
- variables set: FOLDER_ID NETWORK SUBNET_A SUBNET_B SUBNET_C SVC_PORT NS_NAME TEST_IMG APP_PORT ALB_IP SG_1 SG_2

### What we build ###
We are going to build a simple http-server behind a balancer routing traffic to two apps according to their weights

**Experiment with two ingresses (optional)**
<details>
<summary>insert the corresponding values into the provided ingresses and apply them</summary>

```yaml
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example2-ingress1
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
          - path: /proceed
            pathType: Exact
            backend:
              service:
                name: {{ APP_NAME_1 }}-service
                port:
                  number: {{ SVC_PORT }}
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example2-ingress2
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
          - path: /proceed
            pathType: Exact
            backend:
              service:
                name: {{ APP_NAME_2 }}-service
                port:
                  number: {{ SVC_PORT }}
```
</details>

... or automate the process with the script below
```shell
cat ingress0.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'APP_NAME_1=netcat-one APP_NAME_2=netcat-two && echo "{}"' | \
kubectl apply -f -
```
Test the connectivity
```shell
curl http://first-server.info/proceed
<h1>Hello from netcat-two at netcat-two-deployment-8d98687cc-k7pkb</>
```
All the responses will be from netcat-two. Although this behaviour is likely to change in the future, currently 
the controller doesn't merge rules for the same Host and Path.  

### HttpBackendGroup custom resource ###  

Configuration capabilities provided by the ingress format are quite limited. In order to avoid sophisticated annotations 
we introduce a custom resource allowing flexible configuration of backend groups.
The CRD for HttpBackendGroup is part of the Chart and Helm automatically installs it to the cluster beside the ingress controller.  

**Deploy HttpBackendGroup CR**

<details>
<summary>insert the corresponding values into the provided HttpBackendGroup K8s custom resource and apply it</summary>

```yaml
apiVersion: alb.yc.io/v1alpha1
kind: HttpBackendGroup
metadata:
  namespace: {{ NS_NAME }}-ns
  name: example2-bg
spec:
  backends:
    - name: slow
      weight: 20
      service:
        name: {{ APP_NAME_1 }}-service
        port:
          number: {{ SVC_PORT }}
    - name: fast
      weight: 80
      service:
        name: {{ APP_NAME_2 }}-service
        port:
          number: {{ SVC_PORT }}
```
</details>

... or automate the process with the script below
```shell
cat backendgroup.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'APP_NAME_1=netcat-one APP_NAME_2=netcat-two && echo "{}"' | \
kubectl apply -f -
```
Ingress controller will notice the new resource and create a backend group resource in folder ${FOLDER_ID}. Backend groups
configured by HttpBackendGroup CR have a lifecycle independent of the ALB's one, which means that it can be used by many
routes of different balancers, and it exists even if no route is using it. However, quite naturally it can't be deleted 
if there is such a route and therefore an attempt to delete the HttpBackendGroup CR will be pending until the corresponding
Backend Group is no more used by any route.

HttpBackendGroup backend overrides any "standard" backend defined for the Host and Post, it does _not_ merge with them.

Now let's deploy the ingress using this HttpBackendGroup CR (or if you deployed the ingresses from the experimental example
above modify example2-ingress1 as given below and delete example2-ingress2)

**Deploy Ingress**

<details>
<summary>insert the corresponding values into the provided ingresses and apply them</summary>

```yaml
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example2-ingress1
  namespace: {{ NS_NAME }}-ns
  annotations:
    ingress.alb.yc.io/group-name: default
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
                name: example2-bg
---
```
</details>

... or automate the process with the script below
```shell
cat ingress.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'APP_NAME_1=netcat-one APP_NAME_2=netcat-two && echo "{}"' | \
kubectl apply -f -
```

**Test the connectivity**  
```shell
(for i in {1..50}; do curl -s http://first-server.info/proceed | grep -o "netcat-one \|netcat-two "; done) \
| sort | uniq -c
      7 netcat-one 
     43 netcat-two 
```