# Current Operator version
VERSION ?= 1.0.0
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

GOLANGCI_VERSION = v1.35.2

# Image URL to use all building/pushing image targets
IMG ?= astarte/astarte-kubernetes-operator:latest
CRD_OPTIONS ?= "crd:crdVersions=v1"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GOPATHS := ./.;./apis/...;./controllers/...;./lib/...;./test/...;./version/...

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
# Use create due to the annotations limit in apply
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl create -f -

# Replace CRDs into a cluster
# Use replace due to the annotations limit in apply
replace: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl replace -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
# Use create due to the annotations limit in apply
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl create -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen kustomize
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="$(GOPATHS)" output:crd:artifacts:config=config/crd/bases
	$(KUSTOMIZE) build config/helm-crd > charts/astarte-operator/templates/crds.yaml
	$(KUSTOMIZE) build config/helm-rbac > charts/astarte-operator/templates/rbac.yaml
	$(KUSTOMIZE) build config/helm-manager > charts/astarte-operator/templates/manager.yaml
	$(KUSTOMIZE) build config/helm-webhook > charts/astarte-operator/templates/webhook.yaml

# Run go fmt against code
fmt:
	go fmt $(go list ./... | grep -v /external/)

# Run go vet against code
vet:
	go vet $(go list ./... | grep -v /external/)

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="$(GOPATHS)"

# Build the docker image
docker-build: test
	docker build -t ${IMG} .

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# Download golangci-lint if needed
golangci-lint:
ifeq (, $(shell which golangci-lint))
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_VERSION}
GOLANGCI_LINT=$(shell go env GOPATH)/bin/golangci-lint
else
GOLANGCI_LINT=$(shell which golangci-lint)
endif

# Run linter. GOGC is set to reduce memory footprint
lint: golangci-lint
	GOGC=10 $(GOLANGCI_LINT) run -v --timeout 10m
