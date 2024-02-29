
# temporarily support only Yandex Container registry to avoid providing imagePullSecrets
REGISTRY_HOST?=cr.yandex
IMG_NAME?=yc-alb-ingress-controller
TAG?=$(shell git rev-parse --short HEAD)
ifdef REGISTRY_ID
	IMG = $(REGISTRY_HOST)/${REGISTRY_ID}/$(IMG_NAME):${TAG}
	TEST_IMG = $(REGISTRY_HOST)/${REGISTRY_ID}/testapp
	TEST_IMG_GRPC = $(REGISTRY_HOST)/${REGISTRY_ID}/grpc-testapp
endif

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

check_%:
	[ -n "${${*}}" ] || (echo ${*} env var required && false)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./controllers/..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
SHELL='/bin/bash'
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: check_IMG test ## Build docker image with the manager.
	docker build --platform linux/amd64 --build-arg CREATED_AT="$$(date --rfc-3339=seconds)" --build-arg COMMIT=$$(git rev-parse HEAD) -t ${IMG} .

docker-push: check_IMG ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ${KUBECONFIG} or ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ${KUBECONFIG} or ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize check_IMG check_FOLDER_ID check_KEY_FILE patch apply ## Deploy controller to the K8s cluster

undeploy: check_FOLDER_ID check_KEY_FILE unapply unpatch ## Undeploy controller from the K8s cluster

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

apply: kustomize
	$(KUSTOMIZE) build config/default | kubectl apply -f -

unapply: kustomize
	$(KUSTOMIZE) build config/default | kubectl delete -f -

PROD_ENDPOINT=api.cloud.yandex.net:443
patch: check_IMG check_FOLDER_ID check_KEY_FILE
	cp $(KEY_FILE) config/default/ingress-key.json
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	cd config/manager && $(KUSTOMIZE) edit add patch \
		--kind Deployment \
		--name controller-manager \
		--namespace system \
		--patch '[{"op": "add", "path": "/spec/template/spec/containers/0/args/0", "value": "--folder-id='${FOLDER_ID}'"},{"op": "add", "path": "/spec/template/spec/containers/0/args/1", "value": "--endpoint='"$${ENDPOINT:-$(PROD_ENDPOINT)}"'"}]'

unpatch: check_FOLDER_ID
	cd config/manager && $(KUSTOMIZE) edit remove patch \
		--kind Deployment \
		--name controller-manager \
		--namespace system \
		--patch '[{"op": "add", "path": "/spec/template/spec/containers/0/args/0", "value": "--folder-id='${FOLDER_ID}'"},{"op": "add", "path": "/spec/template/spec/containers/0/args/1", "value": "--endpoint='"$${ENDPOINT:-$(PROD_ENDPOINT)}"'"}]' || true
	cd config/manager && $(KUSTOMIZE) edit set image controller=controller
	rm -f config/default/ingress-key.json

##@ e2e

%-e2e: E2ETEMPDIR?=$(shell pwd)/e2e/tmp
%-e2e: export E2EKEYFILE?=${E2ETEMPDIR}/.yc/sa-key.json
%-e2e: export KEY_FILE=$(E2EKEYFILE)
%-e2e: export E2EKUBECONFIG?=${E2ETEMPDIR}/.kube/e2ekubeconfig
%-e2e: export KUBECONFIG=$(E2EKUBECONFIG)
%-e2e: export E2EKUBEVERSION?=1.27
%-e2e: export SUBNET_IDS?=$(shell . e2e/prereq/util.sh && subnet_ids 3)

deploy-e2e: kustomize patch-e2e apply-e2e deploy ## Deploy controller and test app to the K8s cluster

undeploy-e2e: unapply-e2e unpatch-e2e undeploy ## Undeploy controller and test app from the K8s cluster

create-env-e2e: ## Create and configure a K8s cluster for e2e tests
	./e2e/prereq/create.sh
	. ./e2e/prereq/util.sh && inject_cert_id e2e/tests/testdata/*.yaml
	. ./e2e/prereq/util.sh && inject_subnet_ids e2e/tests/testdata/*.yaml
	. ./e2e/prereq/util.sh && inject_address e2e/tests/testdata/*.yaml
	. ./e2e/prereq/util.sh && inject_security_groups e2e/tests/testdata/*.yaml

delete-env-e2e: ## Delete a K8s cluster for e2e tests and related resources (SA, network, kubeconfig etc.)
	. ./e2e/prereq/util.sh && restore_security_group_templates e2e/tests/testdata/*.yaml
	. ./e2e/prereq/util.sh && restore_address_template e2e/tests/testdata/*.yaml
	. ./e2e/prereq/util.sh && restore_subnet_templates e2e/tests/testdata/*.yaml
	./e2e/prereq/cleanup.sh

apply-e2e: patch-e2e
	$(KUSTOMIZE) build e2e/testapp | kubectl apply -f -

unapply-e2e: unpatch-e2e
	$(KUSTOMIZE) build e2e/testapp | kubectl delete -f -

patch-e2e: check_TEST_IMG check_TEST_IMG_GRPC
	cd e2e/testapp && $(KUSTOMIZE) edit set image testapp=${TEST_IMG}
	cd e2e/testapp && $(KUSTOMIZE) edit set image grpc-testapp=${TEST_IMG_GRPC}

unpatch-e2e:
	cd e2e/testapp && $(KUSTOMIZE) edit set image testapp=testapp
	cd e2e/testapp && $(KUSTOMIZE) edit set image grpc-testapp=grpc-testapp

docker-build-e2e-testapp: check_TEST_IMG ## Build docker image with the test app.
	docker build --platform linux/amd64 -t ${TEST_IMG} -f ./e2e/testapp/TestApp.Dockerfile .
	docker build --platform linux/amd64 -t ${TEST_IMG_GRPC} -f ./e2e/testapp/GrpcTestApp.Dockerfile ./e2e/testapp/grpc_hello_server/

docker-push-e2e-testapp: check_TEST_IMG ## Push docker image with the test app.
	docker push ${TEST_IMG}
	docker push ${TEST_IMG_GRPC}

test-e2e: E2ETIMEOUT?=50m
test-e2e: ## Run e2e tests using deployed K8s cluster
	go clean -testcache
	go test -v ./e2e/tests/ -tags e2e -timeout $(E2ETIMEOUT) --keyfile=$(E2EKEYFILE)

update-golden-e2e: E2ETIMEOUT?=50m
update-golden-e2e: ## Run e2e tests using deployed K8s cluster and update golden files
	go clean -testcache
	go test -v ./e2e/tests/ -tags e2e -timeout $(E2ETIMEOUT) --keyfile=$(E2EKEYFILE) -update


%-release: export REGISTRY_ID=crpsjg1coh47p81vh2lc
%-release: export IMG=$(REGISTRY_HOST)/$(REGISTRY_ID)/$(IMG_NAME):${VERSION}

helm-release: check_VERSION check_REGISTRY_ID docker-build docker-push push-helm-release ## build and push helm release

push-helm-release: check_VERSION
	OLD_VERSION=$$(yq -r ".image.tag" ./helm/$(IMG_NAME)/values.yaml) && \
	sed -i "s/$${OLD_VERSION}/$${VERSION}/g" ./helm/$(IMG_NAME)/values.yaml
	sed -i "s/^version: \(.*\)/version: $${VERSION}/" ./helm/$(IMG_NAME)/Chart.yaml
	HELM_EXPERIMENTAL_OCI=1 helm registry login $(REGISTRY_HOST) -u iam -p $$(yc iam create-token)
	HELM_EXPERIMENTAL_OCI=1 helm package ./helm/yc-alb-ingress-controller
	HELM_EXPERIMENTAL_OCI=1 helm push yc-alb-ingress-controller-chart-${VERSION}.tgz  oci://$(REGISTRY_HOST)/$(REGISTRY_ID)

GO_EXCLUDE := /vendor/|/bin/|/genproto/|.pb.go|.gen.go|sensitive.go|validate.go
GO_FILES_CMD := find . -name '*.go' | grep -v -E '$(GO_EXCLUDE)'
Q = $(if $(filter 1,$V),,@)

##@ local

gomod: ## Run go mod vendor
	$(Q) >&2 GOPRIVATE=bb.yandex-team.ru,bb.yandexcloud.net go mod tidy
	$(Q) >&2 GOPRIVATE=bb.yandex-team.ru,bb.yandexcloud.net go mod vendor

GOIMPORTS = $(shell pwd)/bin/goimports
goimports: ## Install goimports if necessary
	$(call go-get-tool,$(GOIMPORTS),golang.org/x/tools/cmd/goimports@latest)

imports: goimports ## Run goimports on all go files
	$(Q) $(GO_FILES_CMD) | xargs -n 50 $(GOIMPORTS) -w -local github.com/yandex-cloud/alb-ingress


GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),github.com/golangci/golangci-lint/cmd/golangci-lint@latest)

lint: golangci-lint
	$(Q) $(GOLANGCI_LINT) run ./... -v

ci: lint
	go test ./...
