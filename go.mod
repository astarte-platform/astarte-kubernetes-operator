module github.com/astarte-platform/astarte-kubernetes-operator

go 1.15

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/cloudflare/cfssl v1.5.0
	github.com/go-logr/logr v0.1.0
	github.com/golangci/golangci-lint v1.35.2 // indirect
	github.com/imdario/mergo v0.3.11
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openlyinc/pointy v1.1.2
	github.com/sykesm/zap-logfmt v0.0.4
	go.uber.org/zap v1.12.0
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.4
	sigs.k8s.io/controller-tools v0.4.1 // indirect
)
