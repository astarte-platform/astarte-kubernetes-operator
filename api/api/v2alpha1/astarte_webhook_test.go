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

package v2alpha1

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
)

var _ = Describe("Astarte Webhook", func() {

	Context("When creating Astarte under Defaulting Webhook", func() {
		It("Should fill in the default value if a required field is empty", func() {

			// TODO(user): Add your logic here

		})
	})

	Context("When creating Astarte under Validating Webhook", func() {
		It("Should deny if a required field is empty", func() {

			// TODO(user): Add your logic here

		})

		It("Should admit if all required fields are provided", func() {

			// TODO(user): Add your logic here

		})
	})

})

func TestValidateSSLListener(t *testing.T) {
	testCases := []struct {
		description    string
		verneSpec      AstarteVerneMQSpec
		expectedErrors int
	}{
		{
			description: "SSL Listener disabled",
			verneSpec: AstarteVerneMQSpec{
				SSLListener: pointy.Bool(false),
			},
			expectedErrors: 0,
		},
		{
			description: "SSL Listener enabled and empty SSLListenerCertSecretName",
			verneSpec: AstarteVerneMQSpec{
				SSLListener:               pointy.Bool(true),
				SSLListenerCertSecretName: "",
			},
			expectedErrors: 1,
		},
		{
			description: "SSL Listener enabled and no SSLListenerCertSecretName",
			verneSpec: AstarteVerneMQSpec{
				SSLListener: pointy.Bool(true),
			},
			expectedErrors: 1,
		},
	}

	// TODO: Test against k8s api to check if the secret exists

	g := NewWithT(t)
	r := &Astarte{}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			r.Spec.VerneMQ = tc.verneSpec
			errs := r.validateSSLListener()
			g.Expect(errs).ToNot(BeNil())
			g.Expect(errs).To(HaveLen(tc.expectedErrors))
		})
	}
}

func TestValidateAutoscalerForClusteredResources(t *testing.T) {
	// TODO: implement this test
}

func TestValidateAutoscalerForClusteredResourcesExcluding(t *testing.T) {
	// TODO: implement this test
}

func TestValidateAstartePriorityClasses(t *testing.T) {
	// Use Gomega with standard Go testing
	g := NewWithT(t)

	testCases := []struct {
		description       string
		enable            bool
		highPriorityValue int
		midPriorityValue  int
		lowPriorityValue  int
		expectError       bool
	}{
		{
			description:       "should not return an error when pod priorities are disabled and values are in correct order",
			enable:            false,
			lowPriorityValue:  100,
			midPriorityValue:  500,
			highPriorityValue: 1000,
			expectError:       false,
		},
		{
			description:       "should not return an error when pod priorities are disabled and values are not in correct order",
			enable:            false,
			lowPriorityValue:  1000,
			midPriorityValue:  500,
			highPriorityValue: 0,
			expectError:       false,
		},
		{
			description:       "should return an error when pod priorities are enabled and values are not in correct order",
			enable:            true,
			lowPriorityValue:  1000,
			midPriorityValue:  1000,
			highPriorityValue: 500,
			expectError:       true,
		},
		{
			description:       "should not return an error when pod priorities are enabled and values are in correct order",
			enable:            true,
			lowPriorityValue:  100,
			midPriorityValue:  500,
			highPriorityValue: 1000,
			expectError:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              tc.enable,
							AstarteHighPriority: &tc.highPriorityValue,
							AstarteMidPriority:  &tc.midPriorityValue,
							AstarteLowPriority:  &tc.lowPriorityValue,
						},
					},
				},
			}

			err := astarte.validateAstartePriorityClasses()

			if tc.expectError {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err.Field).To(Equal("spec.features.astarte{Low|Medium|High}Priority"))
			} else {
				g.Expect(err).To(BeNil())
			}
		})
	}
}

func TestValidatePriorityClassesValues(t *testing.T) {

	// Use Gomega with standard Go testing
	g := NewWithT(t)

	testCases := []struct {
		description       string
		highPriorityValue int
		midPriorityValue  int
		lowPriorityValue  int
		expectError       bool
	}{
		{
			description:       "should not return an error when priorities are in correct order",
			highPriorityValue: 1000,
			midPriorityValue:  500,
			lowPriorityValue:  100,
			expectError:       false,
		},
		{
			description:       "should return an error when high priority is less than mid priority",
			highPriorityValue: 400,
			midPriorityValue:  500,
			lowPriorityValue:  100,
			expectError:       true,
		},
		{
			description:       "should return an error when mid priority is less than low priority",
			highPriorityValue: 1000,
			midPriorityValue:  50,
			lowPriorityValue:  100,
			expectError:       true,
		},
		{
			description:       "should not return an error when priorities are equal",
			highPriorityValue: 500,
			midPriorityValue:  500,
			lowPriorityValue:  100,
			expectError:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              true,
							AstarteHighPriority: &tc.highPriorityValue,
							AstarteMidPriority:  &tc.midPriorityValue,
							AstarteLowPriority:  &tc.lowPriorityValue,
						},
					},
				},
			}

			err := astarte.validatePriorityClassesValues()

			if tc.expectError {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err.Field).To(Equal("spec.features.astarte{Low|Medium|High}Priority"))
			} else {
				g.Expect(err).To(BeNil())
			}
		})
	}

}

func TestValidateUpdateAstarteSystemKeyspace(t *testing.T) {
	g := NewWithT(t)

	testCases := []struct {
		description string
		oldKeyspace AstarteSystemKeyspaceSpec
		newKeyspace AstarteSystemKeyspaceSpec
		expectError bool
	}{
		{
			description: "should return an error when trying to change the keyspace",
			oldKeyspace: AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "SimpleStrategy",
				ReplicationFactor:     1,
				DataCenterReplication: "dc1:3,dc2:2",
			},
			newKeyspace: AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "NetworkTopologyStrategy",
				ReplicationFactor:     2,
				DataCenterReplication: "dc1:2,dc2:3",
			},
			expectError: true,
		},
		{
			description: "should NOT return an error when the keyspace is unchanged",
			oldKeyspace: AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "SimpleStrategy",
				ReplicationFactor:     1,
				DataCenterReplication: "dc1:3,dc2:2",
			},
			newKeyspace: AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "SimpleStrategy",
				ReplicationFactor:     1,
				DataCenterReplication: "dc1:3,dc2:2",
			},
			expectError: false,
		},
		{
			description: "should NOT return an error when the keyspace is empty in both old and new spec",
			oldKeyspace: AstarteSystemKeyspaceSpec{},
			newKeyspace: AstarteSystemKeyspaceSpec{},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			oldAstarte := &Astarte{
				Spec: AstarteSpec{
					Cassandra: AstarteCassandraSpec{
						AstarteSystemKeyspace: tc.oldKeyspace,
					},
				},
			}
			newAstarte := &Astarte{
				Spec: AstarteSpec{
					Cassandra: AstarteCassandraSpec{
						AstarteSystemKeyspace: tc.newKeyspace,
					},
				},
			}

			err := newAstarte.validateUpdateAstarteSystemKeyspace(oldAstarte)
			if tc.expectError {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err.Field).To(Equal("spec.cassandra.astarteSystemKeyspace"))
			} else {
				g.Expect(err).To(BeNil())
			}
		})
	}
}

func TestValidateCFSSLDefinition(t *testing.T) {
	g := NewWithT(t)

	testCases := []struct {
		description string
		cfsslSpec   AstarteCFSSLSpec
		expectError bool
	}{
		{
			description: "should return an error when Deploy is false and URL is empty",
			cfsslSpec: AstarteCFSSLSpec{
				Deploy: pointy.Bool(false),
				URL:    "",
			},
			expectError: true,
		},
		{
			description: "should NOT return an error when Deploy is false and URL is provided",
			cfsslSpec: AstarteCFSSLSpec{
				Deploy: pointy.Bool(false),
				URL:    "http://my-cfssl.com",
			},
			expectError: false,
		},
		{
			description: "should NOT return an error when Deploy is true and URL is empty",
			cfsslSpec: AstarteCFSSLSpec{
				Deploy: pointy.Bool(true),
				URL:    "",
			},
			expectError: false,
		},
		{
			description: "should NOT return an error when Deploy is true and URL is provided",
			cfsslSpec: AstarteCFSSLSpec{
				Deploy: pointy.Bool(true),
				URL:    "http://my-cfssl.com",
			},
			expectError: false,
		},
		{
			description: "should NOT return an error when Deploy is nil (defaults to true) and URL is empty",
			cfsslSpec: AstarteCFSSLSpec{
				Deploy: nil,
				URL:    "",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			astarte := &Astarte{
				Spec: AstarteSpec{
					CFSSL: tc.cfsslSpec,
				},
			}

			err := astarte.validateCFSSLDefinition()

			if tc.expectError {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err.Field).To(Equal("spec.cfssl.url"))
			} else {
				g.Expect(err).To(BeNil())
			}
		})
	}
}
