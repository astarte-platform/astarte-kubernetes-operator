/*
This file is part of Astarte.

Copyright 2020-25 SECO Mind Srl.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

const Timeout = "30s"
const Interval = "1s"

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s",
			fmt.Sprintf("1.30.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = apiv2alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

// Helper function to create a standard Astarte resource for testing
// nolint:unparam
func createTestAstarteResource(name string, namespace string) *apiv2alpha1.Astarte {
	return &apiv2alpha1.Astarte{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apiv2alpha1.AstarteSpec{
			Version: "1.3.0",
			API: apiv2alpha1.AstarteAPISpec{
				Host: "api.example.com",
			},
			RabbitMQ: apiv2alpha1.AstarteRabbitMQSpec{
				Connection: &apiv2alpha1.AstarteRabbitMQConnectionSpec{
					HostAndPort: apiv2alpha1.HostAndPort{
						Host: "rabbitmq.example.com",
						Port: pointy.Int32(5672),
					},
					GenericConnectionSpec: apiv2alpha1.GenericConnectionSpec{
						CredentialsSecret: &apiv2alpha1.LoginCredentialsSecret{
							Name:        "rabbitmq-credentials",
							UsernameKey: "username",
							PasswordKey: "password",
						},
					},
				},
			},
			VerneMQ: apiv2alpha1.AstarteVerneMQSpec{
				HostAndPort: apiv2alpha1.HostAndPort{
					Host: "vernemq.example.com",
					Port: pointy.Int32(1883),
				},
				AstarteGenericClusteredResource: apiv2alpha1.AstarteGenericClusteredResource{
					Image: "docker.io/astarte/vernemq:1.3-snapshot",
				},
			},
			Cassandra: apiv2alpha1.AstarteCassandraSpec{
				Connection: &apiv2alpha1.AstarteCassandraConnectionSpec{
					Nodes: []apiv2alpha1.HostAndPort{
						{
							Host: "cassandra1.example.com",
							Port: pointy.Int32(9042),
						},
					},
					GenericConnectionSpec: apiv2alpha1.GenericConnectionSpec{
						CredentialsSecret: &apiv2alpha1.LoginCredentialsSecret{
							Name:        "cassandra-credentials",
							UsernameKey: "username",
							PasswordKey: "password",
						},
					},
				},
			},
		},
	}
}

// Helper function to clean up test resources
func cleanupAstarteResource(ctx context.Context, client client.Client, namespacedName types.NamespacedName) {
	resource := &apiv2alpha1.Astarte{}
	err := client.Get(ctx, namespacedName, resource)
	if err == nil {
		Expect(client.Delete(ctx, resource)).To(Succeed())
		// Remove finalizers if present to unblock deletion in envtest
		Eventually(func() error {
			current := &apiv2alpha1.Astarte{}
			if getErr := client.Get(ctx, namespacedName, current); getErr != nil {
				if errors.IsNotFound(getErr) {
					return nil
				}
				return getErr
			}
			if len(current.Finalizers) == 0 {
				return nil
			}
			current.Finalizers = nil
			if updErr := client.Update(ctx, current); updErr != nil {
				if errors.IsNotFound(updErr) {
					return nil
				}
				return updErr
			}
			return nil
		}, Timeout, Interval).Should(Succeed())
	} else if !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	// Ensure the CR is gone
	Eventually(func() error {
		return client.Get(ctx, namespacedName, &apiv2alpha1.Astarte{})
	}, Timeout, Interval).ShouldNot(Succeed())
}
