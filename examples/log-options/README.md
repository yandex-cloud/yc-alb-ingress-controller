## Example7: Log options

### Prerequisites

#### Yandex Cloud Resources
- k8s cluster from Managed Service For Kubernetes
- registry from Container Registry to store a test app image

#### Tools
`yc` configured to work with your cloud  
`kubectl` configured for your K8s cluster  
`docker` and `helm` configured and authenticated for Yandex Container Registry

For convenience, throughout the example we'll export all variable configuration data into the environment variables
```shell
export FOLDER_ID=<your folder ID>
export REGISTRY_ID=<your registry ID>
export NETWORK=<your network name>
```

#### Deploy the controller
as described [here](../../helm/yc-alb-ingress-controller/README.md).

This bootstrapping process is described in more details in the [official documentation](https://cloud.yandex.ru/docs/managed-kubernetes/solutions/alb-ingress-controller)

### What we build

We are going to build a simple http-server behind a balancer.
By default Application Load Balancer stores logs in default log group. In this example 
we will change this to our own log group and disable logs at all. 

#### Prepare test app
```shell
export TEST_IMG=cr.yandex/${REGISTRY_ID}/example-app1:latest
docker build -t ${TEST_IMG} -f ./app/Dockerfile .
docker push ${TEST_IMG}
```

#### Deploy test app into K8s cluster

```shell
export NS_NAME=alb-demo
export MAIN_NODEPORT=30080
```
insert the corresponding values into the provided test application deployment and apply it.

... or automate the process with the script below
```shell
# deploy app
cat app/testapp.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'echo "{}"' | \
kubectl apply -f -
```

#### Create security groups

We are going to configure "default" balancer to be created with user-defined _Security Groups_.
Overly simplistic security groups below allow:
- incoming http-traffic on port 80
- incoming http-traffic on port 443
- incoming health checks from "well-known IP ranges"
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

#### Create LogGroup
```shell 
LOG_GROUP_ID=(yc logging group create --name alb-demo12 --folder-id=${FOLDER_ID} --format json| jq -r '.id')
```



#### Apply the ingresses

insert the corresponding values into the provided ingresses and apply them  or automate the process with the script below
```shell
export SG_1 SG_2 SUBNET_A SUBNET_B SUBNET_C

cat ingress.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'echo "{}"' | \
kubectl apply -f -
```

#### Apply the settings
```shell
export LOG_GROUP_ID
cat settings.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'echo "{}"' | \
kubectl apply -f -
```


when balancers are ready the status of ingresses will contain their corresponding IP addresses.
add A-type records with these IPs to the zone of your corresponding domains
(alternatively simply use `curl -k -H "Host: <host>" <ip>` for testing of these demo example).  
Test the connectivity.

You can see in balancer console, that "default" balancer uses default log group, that "disable-logs" balancer does not use any log groups and "non-default" balancer
uses log group, that we created earlier, and has different discard rules from its settings