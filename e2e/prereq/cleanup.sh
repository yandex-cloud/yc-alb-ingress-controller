#!/bin/bash

set -e -o pipefail
set -x

APP=testapp
NETWORK=${APP}-net
CLUSTER=k8s-${APP}

FOLDER_ID=${FOLDER_ID:-$(yc config get folder-id)};
[ -n "${FOLDER_ID}" ] || { echo neither FOLDER_ID env var nor folder-id yc config property set; exit 1; }

# delete K8s cluster and its config
if yc managed-kubernetes cluster get --name "${CLUSTER}" --folder-id "${FOLDER_ID}" 2>/dev/null;
then
  yc managed-kubernetes cluster delete --name "${CLUSTER}" --folder-id "${FOLDER_ID}"
  echo "cluster ${CLUSTER} deleted"
  rm -f "${E2EKUBECONFIG}"
fi

if SA_ID=$(yc iam service-account get --name "k8s-${APP}-sa-${FOLDER_ID}" --folder-id "${FOLDER_ID}" --format=json 2>/dev/null | jq -r '.id' );
then
# delete test S3 bucket
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
fi

if s3cmd info s3://"${E2E_S3_BUCKET}" -c <(echo "${S3CMD_PARAMS}") 2>/dev/null;
then
  s3cmd rm s3://"${E2E_S3_BUCKET}"/ --force --recursive -c <(echo "${S3CMD_PARAMS}")
  s3cmd rb s3://"${E2E_S3_BUCKET}" -c <(echo "${S3CMD_PARAMS}")
fi

# delete cluster SA
if [ -n "${SA_ID}" ];
then
  yc iam service-account delete "${SA_ID}"
  echo "service account k8s-${APP}-sa-${FOLDER_ID} deleted"
  rm -f "${E2EKEYFILE}"
fi

# delete subnets and network
if yc vpc network get --name "${NETWORK}" --folder-id "${FOLDER_ID}" 2>/dev/null;
then
  SUBNETS=$(yc vpc network list-subnets --name "${NETWORK}" --folder-id "${FOLDER_ID}" --format=json | jq -r 'map(.name)')
  zones=(a b c)
  for i in "${!zones[@]}"; do
    if [ -n "$(echo "${SUBNETS}" | jq -r '.[] | select(.=="'${APP}-subnet-"$i"'")')" ];
    then
      yc vpc subnet delete --name ${APP}-subnet-"$i" --folder-id "${FOLDER_ID}"
      echo "subnet ${APP}-subnet-$i deleted"
    fi
  done
  SG_K8S_MAIN=sg-k8s-main-e2e
  SG_K8S_PUBLIC=sg-k8s-public-e2e
  SG_K8S_API=sg-k8s-api-e2e
  if yc vpc security-group get --name sg-e2e-ingress --folder-id "${FOLDER_ID}";
  then
    yc vpc security-group delete --name=sg-e2e-ingress --folder-id "${FOLDER_ID}"
  fi
  if yc vpc security-group get --name sg-e2e-egress --folder-id "${FOLDER_ID}";
  then
    yc vpc security-group delete --name=sg-e2e-egress --folder-id "${FOLDER_ID}"
  fi
  if yc vpc security-group get --name ${SG_K8S_MAIN} --folder-id "${FOLDER_ID}";
  then
    yc vpc security-group delete --name=${SG_K8S_MAIN} --folder-id "${FOLDER_ID}"
  fi
  if yc vpc security-group get --name ${SG_K8S_PUBLIC} --folder-id "${FOLDER_ID}";
  then
    yc vpc security-group delete --name=${SG_K8S_PUBLIC} --folder-id "${FOLDER_ID}"
  fi
  if yc vpc security-group get --name ${SG_K8S_API} --folder-id "${FOLDER_ID}";
  then
    yc vpc security-group delete --name=${SG_K8S_API} --folder-id "${FOLDER_ID}"
  fi

  yc vpc network delete --name ${NETWORK} --folder-id "${FOLDER_ID}"
  echo "network ${NETWORK} deleted"
fi

# TODO(CLOUD-162064): uncomment it, when address will be created in test
## delete static test address
#if yc vpc address get --name "alb-e2e-address" --folder-id "${FOLDER_ID}" 2>/dev/null;
#then
#  yc vpc address delete --name "alb-e2e-address" --folder-id "${FOLDER_ID}"
#fi
