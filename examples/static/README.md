## Example 3: backend group from S3 bucket

### Prerequisites ###

This example re-uses the prerequisites of [Example1](../simple-servers/README.md)
Follow the steps of Example 1 until **Apply the ingresses** step, _except that_  
- applications and services are not needed
- SVC_PORT, TEST_IMG, APP_PORT vars need not be set

By this stage you should have:
- running K8s cluster
- configured tools: yc, kubectl, helm
- ingress-controller deployed into K8s cluster
- cloud resources: subnets, security groups, an address for balancer and a certificate 
for a domain which this example for simplicity assumes to be `first-server.info`.
- variables set: FOLDER_ID NETWORK SUBNET_A SUBNET_B SUBNET_C NS_NAME ALB_IP SG_1 SG_2  

### What we build ###

A static web server backed up by an S3 bucket behind an ALB will be created

### HttpBackendGroup custom resource ###  

An HttpBackendGroup CR allows to configure a backend based on S3-bucket.

#### Prepare S3 backend  

Let's create an S3 bucket and upload our server files to it. Note that the bucket should have _public_ access.  
This example uses [s3cmd](https://s3tools.org/s3cmd) to work with _Yandex Object Storage_. You can read about other tools in the [Official documentation](https://cloud.yandex.ru/docs/storage/tools/)  

```shell
# create a service account for s3cmd configuration
$ SA_ID=$(yc iam service-account create --name s3cmd-sa --folder-id "${FOLDER_ID}" --format json | jq -r '.id')
yc resource-manager folder add-access-binding --id "${FOLDER_ID}" --role storage.editor --subject serviceAccount:"${SA_ID}"

# prepare s3cmd config params.
S3_KEY=$(yc iam access-key create --service-account-id "${SA_ID}" --folder-id "${FOLDER_ID}" --format json)
S3_ACCESS_KEY=$(echo "${S3_KEY}" | jq -r '.access_key.key_id')
S3_SECRET_KEY=$(echo "${S3_KEY}" | jq -r '.secret')
export BUCKET="example3-alb-ingress"

# configure s3cmd. s3cmd --configure is an interactive alternative.
S3CMD_PARAMS=$(cat <<-EOT
  access_key = ${S3_ACCESS_KEY}
  secret_key = ${S3_SECRET_KEY}
  bucket_location = ru-central1
  host_base = storage.yandexcloud.net
  host_bucket = %(bucket)s.storage.yandexcloud.net
EOT
)

# create an S3 bucket with public access
s3cmd mb -P s3://"${BUCKET}" -c <(echo "${S3CMD_PARAMS}")

# add server files
TEMP_FILE=$(mktemp)
echo "<h2>Hello From S3 Backend</>" > "${TEMP_FILE}"
s3cmd put "${TEMP_FILE}" s3://"${BUCKET}/static/index.html" -c <(echo "${S3CMD_PARAMS}")
echo "<h2>Hello From The Page</>" > "${TEMP_FILE}"
s3cmd put "${TEMP_FILE}" s3://"${BUCKET}/static/pages/page.html" -c <(echo "${S3CMD_PARAMS}")
rm -f ${TEMP_FILE}
```

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
  name: bg-with-bucket-e2e
spec:
  backends:
    - name: bucket-backend
      weight: 1
      storageBucket:
        name: {{ BUCKET }}
---
```
</details>

... or automate the process with the script below
```shell
cat backendgroup.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'echo "{}"' | \
kubectl apply -f -
```

#### Deploy Ingress

<details>
<summary>insert the corresponding values into the provided ingress and apply it</summary>

```yaml
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example3-ingress1
  namespace: {{ NS_NAME }}-ns
  annotations:
    ingress.alb.yc.io/group-name: static
    ingress.alb.yc.io/subnets: {{ SUBNET_B }},{{ SUBNET_C }}
    ingress.alb.yc.io/external-ipv4-address: {{ ALB_IP }}
spec:
  rules:
    - host: first-server.info
      http:
        paths:
          - path: /static
            pathType: Prefix
            backend:
              resource:
                apiGroup: alb.yc.io
                kind: HttpBackendGroup
                name: bg-with-bucket-e2e
---
```

</details>

... or automate the process with the script below  
```shell
cat ingress.yaml | \
sed -e "s/{{ \([A-Za-z0-9_]\+\) }}/$\\1/g" -e "s/\"/\\\\\"/g" | \
xargs -d '\n' -I {} sh -c 'echo "{}"' | \
kubectl apply -f -
```

#### Test the connectivity.

```shell
curl http://first-server.info/static/index.html
<h2>Hello From S3 Backend</>
curl http://first-server.info/static/pages/page.html
<h2>Hello From The Page</>
```

**Provide certificate and update server to https**
We could create the balancer already with TLS from the very beginning using the ingress with the corresponding
field, but we can as well add it now and update the balancer.

```shell
kubectl -n ${NS_NAME}-ns patch ingress example3-ingress1 -p \
'{"spec": {"tls": [{"hosts": ["first-server.info"], "secretName": "yc-certmgr-cert-id-'${CERTIFICATE_ID}'"}]}}'
```

naturally, inserting the `tls` field with the proper cert value into the existing Ingress spec and applying it is a valid alternative.
```yaml
spec:
  tls:
  - hosts:
    - first-server.info
    secretName: yc-certmgr-cert-id-{{ CERTIFICATE_ID }}
```

Test the secure connectivity.

```shell
curl https://first-server.info/static/index.html
<h2>Hello From S3 Backend</>
curl https://first-server.info/static/pages/page.html
<h2>Hello From The Page</>

# redirect
curl -L http://first-server.info/static/index.html
<h2>Hello From S3 Backend</>
curl -L http://first-server.info/static/pages/page.html
<h2>Hello From The Page</>
```
