# Current Operator version
VERSION ?= 1.1.0-dev
# Default bundle image tag
BUNDLE_IMG ?= astarte/astarte-kubernetes-operator:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Tools versions
CONTROLLER_GEN_VERSION = v0.8.0
CONTROLLER_RUNTIME_VERSION = v0.11.0 # This must be coincident with the version set in go.mod
GOLANGCI_VERSION = v1.35.2
KUSTOMIZE_VERSION = v3.8.7
# Conversion-gen version should match the older k8s version supported by the operator.
# Note: the major lags behind by one (see https://github.com/kubernetes/code-generator#where-does-it-come-from).
CONVERSION_GEN_VERSION = v0.19.16
CRD_REF_DOCS_VERSION=v0.0.8

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23

# Image URL to use all building/pushing image targets
IMG ?= astarte/astarte-kubernetes-operator:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

GOPATHS := ./.;./apis/...;./controllers/...;./lib/...;./test/...;./version/...

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: all
all: manager

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

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen kustomize ## Generate manifests e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(KUSTOMIZE) build config/helm-crd > charts/astarte-operator/templates/crds.yaml
	$(KUSTOMIZE) build config/helm-rbac > charts/astarte-operator/templates/rbac.yaml
	$(KUSTOMIZE) build config/helm-manager > charts/astarte-operator/templates/manager.yaml
	$(KUSTOMIZE) build config/helm-webhook > charts/astarte-operator/templates/webhook.yaml

.PHONY: generate ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
generate: controller-gen conversion-gen
	$(CONVERSION_GEN) --go-header-file "./hack/boilerplate.go.txt" --input-dirs "./apis/api/v1alpha2" \
		-O zz_generated.conversion --output-base "."
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="$(GOPATHS)"

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt $(go list ./... | grep -v /external/)

.PHONY: vet
vet: ## Run go vet against code.
	go vet $(go list ./... | grep -v /external/)

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
.PHONY: test
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/${CONTROLLER_RUNTIME_VERSION}/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test -v ./... -coverprofile cover.out

##@ Build

.PHONY: manager
manager: generate fmt vet  ## Build manager binary.
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: generate fmt vet manifests ## Run a controller from your host against the Kubernetes cluster configured in ~/.kube/config. Call with ENABLE_WEBHOOKS=false to exclude webhooks.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

# Use create due to the annotations limit in apply
.PHONY: install ## Install CRDs into the K8s cluster specified in ~/.kube/config.
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl create -f -

# Replace CRDs into a cluster
.PHONY: replace
replace: manifests kustomize ## Replace CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl replace -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

# Use create due to the annotations limit in apply
.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl create -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION} ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@${KUSTOMIZE_VERSION} ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

.PHONY: conversion-gen
conversion-gen: ## Download conversion-gen locally if necessary.
ifeq (, $(shell which conversion-gen))
	@{ \
	set -e ;\
	CONVERSION_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONVERSION_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get k8s.io/code-generator/cmd/conversion-gen@${CONVERSION_GEN_VERSION} ;\
	rm -rf $$CONVERSION_GEN_TMP_DIR ;\
	}
CONVERSION_GEN=$(GOBIN)/conversion-gen
else
CONVERSION_GEN=$(shell which conversion-gen)
endif

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

##@ Docs

.PHONY: crd-ref-docs
crd-ref-docs: ## Download crd-ref-docs locally if necessary.
ifeq (, $(shell which crd-ref-docs))
	@{ \
	set -e ;\
	CRD_REF_DOCS_TMP_DIR=$$(mktemp -d) ;\
	cd $$CRD_REF_DOCS_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/elastic/crd-ref-docs@${CRD_REF_DOCS_VERSION};\
	rm -rf $$CRD_REF_DOCS_TMP_DIR ;\
	}
CRD_REF_DOCS=$(GOBIN)/crd-ref-docs
else
CRD_REF_DOCS=$(shell crd-ref-docs)
endif

.PHONY: crd-docs
crd-docs: crd-ref-docs ## Generate API reference documentation from code.
	$(CRD_REF_DOCS) --config="docs/autogen/config.yaml" --renderer=markdown --max-depth=10 \
		--source-path="apis" --output-path="docs/content/index.md"

##@ Linter

.PHONY: lint
lint: golangci-lint ## Run linter.
	GOGC=10 $(GOLANGCI_LINT) run -v --timeout 10m

.PHONY: golangci-lint
golangci-lint: ## Download golangci-lint if needed.
ifeq (, $(shell which golangci-lint))
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_VERSION}
GOLANGCI_LINT=$(shell go env GOPATH)/bin/golangci-lint
else
GOLANGCI_LINT=$(shell which golangci-lint)
endif
