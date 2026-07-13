/*
This file is part of Astarte.

Copyright 2020-26 SECO Mind Srl.

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
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/yaml"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	// +kubebuilder:scaffold:imports
)

const Timeout = "30s"
const Interval = "1s"

var cfg *rest.Config
var k8sClient client.Client
var ctx context.Context
var cancel context.CancelFunc
var testEnv *envtest.Environment
var baseCr *apiv2alpha1.Astarte

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = integrationutils.SetupTestSuite()

	By("bootstrapping test environment")
	testEnv = integrationutils.DefaultEnvTestConfig(
		filepath.Join("..", "..", "..", "config", "crd", "bases"),
		filepath.Join("..", "..", "..", "bin", "k8s"),
	)

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = apiv2alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Tests call Reconcile() directly and inspect return values, so we
	// don't register the controller with a manager here.

	By("loading the base Astarte manifest")
	manifestPath := filepath.Join("..", "..", "..", "test", "manifests", "api_v2alpha1_astarte_1.4.yaml")
	manifestBytes, err := os.ReadFile(manifestPath)
	Expect(err).ToNot(HaveOccurred())

	baseCr = &apiv2alpha1.Astarte{}
	err = yaml.Unmarshal(manifestBytes, baseCr)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	integrationutils.StopEnvTest(testEnv, cancel)
})
