#!/bin/bash

set -e -o pipefail
set -x

APP=testapp
NETWORK=${APP}-net
CLUSTER=k8s-${APP}
SUPPORTED_KUBE_VERSIONS=( 1.24 1.25 1.26 1.27 )

# check k8s config env var
[ -n "${E2EKUBECONFIG}" ] || { echo E2EKUBECONFIG env var not set; exit 1; }
[ -n "${E2EKEYFILE}" ] || { echo E2EKEYFILE env var not set; exit 1; }
[ -n "${REGISTRY_ID}" ] || { echo REGISTRY_ID env var not set; exit 1; }

FOLDER_ID=${FOLDER_ID:-$(yc config get folder-id)};
[ -n "${FOLDER_ID}" ] || { echo neither FOLDER_ID env var nor folder-id yc config property set; exit 1; }

# create self-signed certificate
if ! yc certificate-manager certificate get --name yc-alb-e2e-cert --folder-id "${FOLDER_ID}";
then
  openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
    -keyout e2e_cert.key -out e2e_cert.crt -subj "/CN=first-server.info/C=RU/L=St.Petersburg/O=Yandex/OU=Yandex.Cloud" \
    -addext "subjectAltName=DNS:first-server.info,DNS:second-server.info,DNS:third-server.info,DNS:fourth-server.info"
  yc certificate-manager certificate create \
    --name yc-alb-e2e-cert \
    --description "self-signed certificate for e2e test data ingresses" \
    --chain e2e_cert.crt \
    --key e2e_cert.key \
    --folder-id "${FOLDER_ID}"
    echo "self-signed certificate yc-alb-e2e-cert created"
fi

[ ! -f "${E2EKUBECONFIG}" ] || { echo using existing k8s config at "${E2EKUBECONFIG}"; exit ; }

# create network
if ! NETWORK_ID=$(yc vpc network get --name ${NETWORK} --folder-id "${FOLDER_ID}" --format json | jq -r '.id') 2>/dev/null ;
then
  NETWORK_ID=$(yc vpc network create --name ${NETWORK} --folder-id "${FOLDER_ID}" --format json | jq -r '.id')
  echo "network ${NETWORK} created"
fi

# create subnets
SUBNETS=$(yc vpc network list-subnets --name "${NETWORK}" --folder-id "${FOLDER_ID}" --format=json | jq -r 'map(.name)')
zones=(a b c)
for i in "${!zones[@]}"; do
  if [ -z "$(echo "${SUBNETS}" | jq -r '.[] | select(.=="'${APP}-subnet-"$i"'")')" ];
  then
    (( k=i+10 )) # avoid CIDR's overlapping with Docker

  SUBNET_CREATE_REQ_TEMPLATE="description: \"\"
dhcp_options: null
egress_nat_enable: false
extra_params: null
folder_id: {{ FOLDER_ID }}
labels: {}
name: {{ NAME }}
network_id: {{ NETWORK_ID }}
route_table_id: \"\"
serverless_functions_pluggable: false
v4_cidr_blocks: [{{ V4_CIDR }}]
v6_cidr_blocks: []
v6_cidr_specs:
  - project_id_cidr_spec:
      prefix_length: 112
      project_id: fcf3
      subnet_bits: cafe
zone_id: {{ ZONE_ID }}"

  SUBNET_CREATE_REQ=$(sed -e "s/{{ FOLDER_ID }}/${FOLDER_ID}/g;
            s/{{ NAME }}/${APP}-subnet-"$i"/g;
            s/{{ NETWORK_ID }}/${NETWORK_ID}/g;
            s/{{ ZONE_ID }}/ru-central1-"${zones[$i]}"/g;
            s/{{ V4_CIDR }}/192.168.${k}.0\/24/g" <<< "${SUBNET_CREATE_REQ_TEMPLATE}")

  ycp vpc subnet create -r <(echo "${SUBNET_CREATE_REQ}")
  echo "subnet ${APP}-subnet-$i created"

  fi
done

# create a static address to be used as a balancer endpoint
if ! yc vpc address get --name "alb-e2e-address" --folder-id "${FOLDER_ID}" 2>/dev/null;
then
  yc vpc address create --external-ipv4 zone=ru-central1-a --name "alb-e2e-address" --description "address for alb ingress e2e tests" --folder-id "${FOLDER_ID}"
fi

# create security groups for default balancer
if ! SG_INGRESS_ID=$(yc vpc security-group get --name sg-e2e-ingress --folder-id "${FOLDER_ID}" --format=json 2>/dev/null | jq -r '.id' );
then
  SG_INGRESS_ID=$(yc vpc security-group create --name=sg-e2e-ingress --network-name ${NETWORK} --folder-id "${FOLDER_ID}" \
  --rule=direction=ingress,port=80,protocol=tcp,v4-cidrs="[0.0.0.0/0]",v6-cidrs="[::/0]" \
  --rule=direction=ingress,port=443,protocol=tcp,v4-cidrs="[0.0.0.0/0]",v6-cidrs="[::/0]" \
  --rule=direction=ingress,port=30080,protocol=tcp,v4-cidrs="[198.18.235.0/24,198.18.248.0/24]" \
  --rule=direction=ingress,from-port=0,to-port=65535,protocol=any,v4-cidrs="[0.0.0.0/0]",v6-cidrs="[::/0]" --format=json | jq -r '.id')
  echo "security group sg-e2e-ingress created"
fi
if ! SG_EGRESS_ID=$(yc vpc security-group get --name sg-e2e-egress --folder-id "${FOLDER_ID}" --format=json 2>/dev/null | jq -r '.id' );
then
  SG_EGRESS_ID=$(yc vpc security-group create --name=sg-e2e-egress --network-name ${NETWORK} --folder-id "${FOLDER_ID}" \
  --rule=direction=egress,from-port=10501,to-port=10502,protocol=tcp,v4-cidrs="[0.0.0.0/0]",v6-cidrs="[::/0]" \
  --rule=direction=egress,from-port=30080,to-port=30085,protocol=tcp,v4-cidrs="[0.0.0.0/0]",v6-cidrs="[::/0]" --format=json | jq -r '.id')
  echo "security group sg-e2e-egress created"
fi

# create security groups for k8s cluster and node group
# https://cloud.yandex.ru/docs/managed-kubernetes/operations/security-groups

# service traffic
SG_K8S_MAIN=sg-k8s-main-e2e
if ! SG_K8S_MAIN_ID=$(yc vpc security-group get --name ${SG_K8S_MAIN} --folder-id "${FOLDER_ID}" --format=json 2>/dev/null | jq -r '.id' );
then
  SG_K8S_MAIN_ID=$(yc vpc security-group create --name=${SG_K8S_MAIN} --network-name ${NETWORK} --folder-id "${FOLDER_ID}" \
  --rule=direction=ingress,from-port=0,to-port=65535,protocol=tcp,v4-cidrs="[198.18.235.0/24,198.18.248.0/24]",v6-cidrs="[2a0d:d6c0:2:ba::/64]" \
  --rule=direction=ingress,from-port=0,to-port=65535,protocol=any,predefined=self_security_group \
  --rule=direction=ingress,from-port=0,to-port=65535,protocol=any,v4-cidrs="[192.168.10.0/24]" \
  --rule=direction=ingress,from-port=0,to-port=65535,protocol=any,v6-cidrs="[::/0]" \
  --rule=direction=ingress,protocol=icmp,v4-cidrs="[172.16.0.0/12]" \
  --rule=direction=egress,from-port=0,to-port=65535,protocol=any,v4-cidrs="[0.0.0.0/0]",v6-cidrs="[::/0]" --format=json | jq -r '.id')
  echo "security group ${SG_K8S_MAIN} created"
fi

# external traffic
SG_K8S_PUBLIC=sg-k8s-public-e2e
if ! SG_K8S_PUBLIC_ID=$(yc vpc security-group get --name ${SG_K8S_PUBLIC} --folder-id "${FOLDER_ID}" --format=json 2>/dev/null | jq -r '.id' );
then
  SG_K8S_PUBLIC_ID=$(yc vpc security-group create --name=${SG_K8S_PUBLIC} --network-name ${NETWORK} --folder-id "${FOLDER_ID}" \
  --rule=direction=ingress,from-port=10501,to-port=10502,protocol=tcp,v4-cidrs="[0.0.0.0/0]",v6-cidrs="[::/0]" \
  --rule=direction=ingress,from-port=30000,to-port=32767,protocol=tcp,v4-cidrs="[0.0.0.0/0]",v6-cidrs="[::/0]" --format=json | jq -r '.id')
  echo "security group ${SG_K8S_PUBLIC} created"
fi

# cluster API
SG_K8S_API=sg-k8s-api-e2e
if ! SG_K8S_API_ID=$(yc vpc security-group get --name ${SG_K8S_API} --folder-id "${FOLDER_ID}" --format=json 2>/dev/null | jq -r '.id' );
then
  SG_K8S_API_ID=$(yc vpc security-group create --name=${SG_K8S_API} --network-name ${NETWORK} --folder-id "${FOLDER_ID}" \
  --rule=direction=ingress,port=443,protocol=tcp,v4-cidrs="[0.0.0.0/0]",v6-cidrs="[::/0]" \
  --rule=direction=ingress,port=6443,protocol=tcp,v4-cidrs="[0.0.0.0/0]",v6-cidrs="[::/0]" --format=json | jq -r '.id')
  echo "security group ${SG_K8S_API} created"
fi

# create cluster SA
if ! SA_ID=$(yc iam service-account get --name "k8s-${APP}-sa-${FOLDER_ID}" --folder-id "${FOLDER_ID}" --format=json 2>/dev/null | jq -r '.id' );
then
  SA_ID=$(yc iam service-account create --name "k8s-${APP}-sa-${FOLDER_ID}" --folder-id "${FOLDER_ID}" --format json | jq -r '.id');
  echo "service account k8s-${APP}-sa-${FOLDER_ID} created"
fi

# add roles
# for simplicity in e2e we'll use the same SA for master, nodes and balancer
yc resource-manager folder add-access-binding --id "${FOLDER_ID}" --role editor --subject serviceAccount:"${SA_ID}"
# at this moment balancer creation doesn't handle cert rights individually, so we set downloader rights on the whole folder
yc resource-manager folder add-access-binding --id "${FOLDER_ID}" --role certificate-manager.certificates.downloader --subject serviceAccount:"${SA_ID}"
yc container registry add-access-binding --id "${REGISTRY_ID}" --role viewer --subject serviceAccount:"${SA_ID}" --folder-id "${FOLDER_ID}"


mkdir -p "$(dirname "${E2EKEYFILE}")"
yc iam key create --service-account-id "${SA_ID}" --output "${E2EKEYFILE}"

# create K8s master
if ! K8S_VERSION=$(yc managed-kubernetes cluster get --name "${CLUSTER}" --folder-id "${FOLDER_ID}" --format=json 2>/dev/null | jq -r '.master.version_info.current_version');
then
  [ -n "${E2EKUBEVERSION}" ] || { echo E2EKUBEVERSION env var not set; exit 1; }
  echo "${SUPPORTED_KUBE_VERSIONS[@]}" | grep -q -w "${E2EKUBEVERSION//./\\.}" || { echo unsupported Kubernetes master version "${K8S_VERSION}"; exit 1; }
  yc managed-kubernetes cluster create \
    --name "${CLUSTER}" \
    --network-name "${NETWORK}" \
    --service-account-id "${SA_ID}" \
    --node-service-account-id "${SA_ID}"\
    --version "${E2EKUBEVERSION}" \
    --security-group-ids="${SG_K8S_MAIN_ID},${SG_K8S_API_ID}" \
    --folder-id "${FOLDER_ID}" \
    --dual-stack \
    --regional \
    --cluster-ipv6-range fc00::/96 \
    --service-ipv6-range fc01::/112 \
    --cluster-ipv4-range 10.97.0.0/16 \
    --service-ipv4-range=10.98.0.0/16 \
    --public-ipv6 "${CLUSTER_IP}"

  echo "cluster ${CLUSTER} created"
else
  echo "${SUPPORTED_KUBE_VERSIONS[@]}" | grep -q -w "${K8S_VERSION//./\\.}" || { echo unsupported Kubernetes master version "${K8S_VERSION}"; exit 1; }
fi

# create K8s node group
if ! yc managed-kubernetes node-group get --name "${CLUSTER}-ng" --folder-id "${FOLDER_ID}" 2>/dev/null;
then
  yc managed-kubernetes node-group create \
    --name ${CLUSTER}-ng \
    --cluster-name ${CLUSTER} \
    --platform-id standard-v2 \
    --cores 2 \
    --memory 4 \
    --core-fraction 50 \
    --disk-type network-ssd \
    --fixed-size 2 \
    --location zone=ru-central1-a \
    --network-interface "security-group-ids=[${SG_K8S_MAIN_ID},${SG_K8S_PUBLIC_ID}],ipv4-address=nat,ipv6-address=auto,subnets=${APP}-subnet-0" \
    --folder-id "${FOLDER_ID}"
  echo "node group ${CLUSTER}-ng created"
fi

# create k8s config
yc managed-kubernetes cluster get-credentials --external-ipv6 --name ${CLUSTER} --kubeconfig "${E2EKUBECONFIG}" --folder-id "${FOLDER_ID}"
echo "configured cluster ${CLUSTER} auth at ${E2EKUBECONFIG}"

# create test S3 bucket with public access
which s3cmd 1>/dev/null || { echo s3cmd not found; exit 1; }
S3_KEY=$(yc iam access-key create --service-account-id "${SA_ID}" --description "key for e2e test S3" --folder-id "${FOLDER_ID}" --format json)
S3_ACCESS_KEY=$(echo "${S3_KEY}" | jq -r '.access_key.key_id')
S3_SECRET_KEY=$(echo "${S3_KEY}" | jq -r '.secret')
E2E_S3_BUCKET="alb-ingress-e2e-${FOLDER_ID}"

S3CMD_PARAMS=$(cat <<-EOT
  access_key = ${S3_ACCESS_KEY}
  secret_key = ${S3_SECRET_KEY}
  bucket_location = ru-central1
  host_base = storage.yandexcloud.net
  host_bucket = %(bucket)s.storage.yandexcloud.net
EOT
)

if ! s3cmd info s3://"${E2E_S3_BUCKET}" -c <(echo "${S3CMD_PARAMS}") 2>/dev/null;
then
  s3cmd mb -P s3://"${E2E_S3_BUCKET}" -c <(echo "${S3CMD_PARAMS}")
  TEMP_FILE=$(mktemp)
  trap 'rm -f ${TEMP_FILE}' EXIT
  s3cmd put "${TEMP_FILE}" s3://"${E2E_S3_BUCKET}/static/index.html" -c <(echo "${S3CMD_PARAMS}")
  echo "<h2>Hello From S3 Backend</>" > "${TEMP_FILE}"
  s3cmd put "${TEMP_FILE}" s3://"${E2E_S3_BUCKET}/static/index.html" -c <(echo "${S3CMD_PARAMS}")
  echo Bucket "${E2E_S3_BUCKET}" with a test file created
fi
