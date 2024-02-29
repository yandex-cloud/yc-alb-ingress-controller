#!/bin/bash

set -e -o pipefail

subnet_ids() {
  local slice
  local SUBNETS
  [ -z "$1" ] || slice="| .[0:$1]"
  SUBNETS=$(yc vpc network list-subnets --name "${NETWORK:-testapp-net}" --folder-id "${FOLDER_ID}" --format=json | jq -r '[.[] | select(.name | test("testapp."))] | sort_by(.zone_id) '"${slice}"'| map(.id) | join(",")')
  echo "${SUBNETS}"
}

inject_cert_id() {
  local CERT_NAME=yc-certmgr-cert-id
  local CERT_ID
  CERT_ID=$(yc certificate-manager certificate get --name yc-alb-e2e-cert --folder-id "${FOLDER_ID}" --format json | jq -r '.id')
  sed -i "s/secretName: \(${CERT_NAME}\)-.*/secretName: \1-${CERT_ID}/g" "$@"
}

inject_subnet_ids() {
  local IDS
  local zones=(A B C)
  read -ra IDS < <(subnet_ids 3 | tr ',' ' ')
  [ "${#IDS[@]}" -eq 3 ] || { echo "error : ${#IDS[@]} subnets found, expected 3"; exit 1; }
  for i in "${!zones[@]}";
  do
    local replacement="${replacement} -e \"s/{{ SUBNET_${zones[i]} }}/${IDS[i]}/g\""
  done
  eval "sed -i $replacement" "$@"
}

restore_subnet_templates() {
  local IDS
  local zones=(A B C)
  read -ra IDS < <(subnet_ids 3 | tr ',' ' ')
  for i in "${!IDS[@]}";
  do
    local replacement="${replacement} -e \"s/${IDS[i]}/{{ SUBNET_${zones[i]} }}/g\""
  done
  [ -z "${replacement}" ] || eval "sed -i $replacement" "$@"
}

inject_address() {
# TODO(CLOUD-162064): use created address when it will be created in the test
#  local ALB_IP
#  ALB_IP=$(yc vpc address get --name "alb-e2e-address" --folder-id "${FOLDER_ID}" --format json | jq -r '.external_ipv4_address.address')
  sed -i "s/{{ ALB_IP }}/${ALB_IP}/g" "$@"
}

restore_address_template() {
# TODO(CLOUD-162064): use created address when it will be created in the test
#  local ALB_IP
#  ALB_IP=$(yc vpc address get --name "alb-e2e-address" --folder-id "${FOLDER_ID}" --format json | jq -r '.external_ipv4_address.address') || true
  [ -z "${ALB_IP}" ] || sed -i "s/${ALB_IP}/{{ ALB_IP }}/g" "$@"
}

inject_security_groups() {
  local SG_1
  local SG_2
  SG_1=$(yc vpc security-groups get --name "sg-e2e-ingress" --folder-id "${FOLDER_ID}" --format json | jq -r '.id' ) || true
  SG_2=$(yc vpc security-groups get --name "sg-e2e-egress" --folder-id "${FOLDER_ID}" --format json | jq -r '.id') || true
  sed -i -e "s/{{ SG_1 }}/${SG_1}/g" -e "s/{{ SG_2 }}/${SG_2}/g" "$@"
}

restore_security_group_templates() {
  local SG_1
  local SG_2
  SG_1=$(yc vpc security-groups get --name "sg-e2e-ingress" --folder-id "${FOLDER_ID}" --format json | jq -r '.id' ) || true
  [ -z "${SG_1}" ] || sed -i "s/${SG_1}/{{ SG_1 }}/g" "$@"
  SG_2=$(yc vpc security-groups get --name "sg-e2e-egress" --folder-id "${FOLDER_ID}" --format json | jq -r '.id') || true
  [ -z "${SG_1}" ] || sed -i "s/${SG_2}/{{ SG_2 }}/g" "$@"
}