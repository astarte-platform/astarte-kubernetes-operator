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

package integration

import (
	"context"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// SetupTestSuite sets up the Ginkgo logger, global Eventually defaults, and a
// cancellable context to be used across all test suites.
func SetupTestSuite() (context.Context, context.CancelFunc) {
	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel := context.WithCancel(context.TODO())
	return ctx, cancel
}

// GetFirstFoundEnvTestBinaryDir finds the first subdirectory under basePath.
// Use it to locate the envtest binaries. Call 'make setup-envtest' first.
func GetFirstFoundEnvTestBinaryDir(basePath string) string {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}

// DefaultEnvTestConfig returns an envtest.Environment for the given paths.
// ErrorIfCRDPathMissing defaults to true.
func DefaultEnvTestConfig(crdBasePath, binaryBasePath string) *envtest.Environment {
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{crdBasePath},
		ErrorIfCRDPathMissing: true,
	}
	if binDir := GetFirstFoundEnvTestBinaryDir(binaryBasePath); binDir != "" {
		env.BinaryAssetsDirectory = binDir
	}
	return env
}

// StartEnvTest starts the envtest and returns the generated rest.Config and Client.
func StartEnvTest(testEnv *envtest.Environment) (*rest.Config, client.Client, error) {
	cfg, err := testEnv.Start()
	if err != nil {
		return nil, nil, err
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, nil, err
	}

	return cfg, k8sClient, nil
}

// StopEnvTest cancels the context (if any) and stops the envtest with a timeout.
func StopEnvTest(testEnv *envtest.Environment, cancel context.CancelFunc) {
	if cancel != nil {
		cancel()
	}
	Eventually(func() error {
		return testEnv.Stop()
	}, time.Minute, time.Second).Should(Succeed())
}
