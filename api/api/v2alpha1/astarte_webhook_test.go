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
	"fmt"
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

func TestValidateUpdateAstarteInstanceID(t *testing.T) {
	g := NewWithT(t)

	testCases := []struct {
		description          string
		oldAstarteInstanceID string
		newAstarteInstanceID string
		expectError          bool
	}{
		{
			description:          "should return an error when trying to change the instanceID",
			oldAstarteInstanceID: "old-instance-id",
			newAstarteInstanceID: "new-instance-id",
			expectError:          true,
		},
		{
			description:          "should NOT return an error when the instanceID is unchanged",
			oldAstarteInstanceID: "same-instance-id",
			newAstarteInstanceID: "same-instance-id",
			expectError:          false,
		},
		{
			description:          "should NOT return an error when the instanceID is empty in both old and new spec",
			oldAstarteInstanceID: "",
			newAstarteInstanceID: "",
			expectError:          false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			oldAstarte := &Astarte{
				Spec: AstarteSpec{
					AstarteInstanceID: tc.oldAstarteInstanceID,
				},
			}
			newAstarte := &Astarte{
				Spec: AstarteSpec{
					AstarteInstanceID: tc.newAstarteInstanceID,
				},
			}

			err := newAstarte.validateUpdateAstarteInstanceID(oldAstarte)
			if tc.expectError {
				g.Expect(err).ToNot(BeNil())
			} else {
				g.Expect(err).To(BeNil())
			}
		})
	}

}

func TestValidatePodLabelsForClusteredResources(t *testing.T) {
	testCases := []struct {
		name        string
		labels      map[string]string
		expectError bool
	}{
		{
			name: "with allowed custom labels",
			labels: map[string]string{
				"custom-label":           "lbl",
				"my.custom.domain/label": "value",
			},
			expectError: false,
		},
		{
			name: "with unallowed reserved labels",
			labels: map[string]string{
				"app":          "my-app",
				"component":    "my-component",
				"astarte-role": "my-role",
				"flow-role":    "my-role",
			},
			expectError: true,
		},
		{
			name:        "with no labels",
			labels:      nil,
			expectError: false,
		},
	}

	testComponents := map[string]AstarteSpec{
		"DataUpdaterPlant": {Components: AstarteComponentsSpec{DataUpdaterPlant: AstarteDataUpdaterPlantSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
		"TriggerEngine":    {Components: AstarteComponentsSpec{TriggerEngine: AstarteTriggerEngineSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
		"Flow":             {Components: AstarteComponentsSpec{Flow: AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
		"Housekeeping":     {Components: AstarteComponentsSpec{Housekeeping: AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
		"RealmManagement":  {Components: AstarteComponentsSpec{RealmManagement: AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
		"Pairing":          {Components: AstarteComponentsSpec{Pairing: AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
		"VerneMQ":          {VerneMQ: AstarteVerneMQSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}},
	}

	for componentName, baseSpec := range testComponents {
		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%s %s", componentName, tc.name), func(t *testing.T) {
				g := NewWithT(t)

				r := &Astarte{Spec: baseSpec}

				switch componentName {
				case "DataUpdaterPlant":
					r.Spec.Components.DataUpdaterPlant.PodLabels = tc.labels
				case "TriggerEngine":
					r.Spec.Components.TriggerEngine.PodLabels = tc.labels
				case "Flow":
					r.Spec.Components.Flow.PodLabels = tc.labels
				case "Housekeeping":
					r.Spec.Components.Housekeeping.PodLabels = tc.labels
				case "RealmManagement":
					r.Spec.Components.RealmManagement.PodLabels = tc.labels
				case "Pairing":
					r.Spec.Components.Pairing.PodLabels = tc.labels
				case "VerneMQ":
					r.Spec.VerneMQ.PodLabels = tc.labels
				}

				err := r.validatePodLabelsForClusteredResources()

				if tc.expectError {
					g.Expect(err).ToNot(BeNil())
					g.Expect(err).ToNot(BeEmpty())
				} else {
					g.Expect(err).ToNot(BeNil())
					g.Expect(err).To(BeEmpty())
				}
			})
		}
	}
}

func TestValidatePodLabelsForClusteredResource(t *testing.T) {
	g := NewWithT(t)

	testCases := []struct {
		name        string
		labels      map[string]string
		expectError bool
	}{
		{
			name: "with allowed custom labels",
			labels: map[string]string{
				"custom-label":           "lbl",
				"my.custom.domain/label": "value",
			},
			expectError: false,
		},
		{
			name: "with unallowed reserved labels",
			labels: map[string]string{
				"app":          "my-app",
				"component":    "my-component",
				"astarte-role": "my-role",
				"flow-role":    "my-role",
			},
			expectError: true,
		},
		{
			name:        "with no labels",
			labels:      nil,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			a := &AstarteGenericClusteredResource{
				PodLabels: tc.labels,
			}

			err := validatePodLabelsForClusteredResource(PodLabelsGetter(a))
			if tc.expectError {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err).ToNot(BeEmpty())
			} else {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err).To(BeEmpty())
			}
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

func TestValidateCreateAstarteSystemKeyspace(t *testing.T) {
	g := NewWithT(t)
	testCases := []struct {
		description  string
		astarte      *Astarte
		expectedErrs int
	}{
		{
			description: "It should not return with SimpleStrategy and valid odd replication factor",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy: "SimpleStrategy",
						ReplicationFactor:   3,
					},
				}},
			},
			expectedErrs: 0,
		},
		{
			description: "NetworkTopologyStrategy with single valid DC",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy:   "NetworkTopologyStrategy",
						DataCenterReplication: "dc1:3",
					},
				}},
			},
			expectedErrs: 0,
		},
		{
			description: "NetworkTopologyStrategy with multiple valid DCs",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy:   "NetworkTopologyStrategy",
						DataCenterReplication: "dc1:3,dc2:5,dc3:1",
					},
				}},
			},
			expectedErrs: 0,
		},
		{
			description: "SimpleStrategy with invalid even replication factor",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy: "SimpleStrategy",
						ReplicationFactor:   2,
					},
				}},
			},
			expectedErrs: 1,
		},
		{
			description: "NetworkTopologyStrategy with invalid format (no colon)",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy:   "NetworkTopologyStrategy",
						DataCenterReplication: "dc1",
					},
				}},
			},
			expectedErrs: 1,
		},
		{
			description: "NetworkTopologyStrategy with invalid format (too many colons)",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy:   "NetworkTopologyStrategy",
						DataCenterReplication: "dc1:3:bad",
					},
				}},
			},
			expectedErrs: 1,
		},
		{
			description: "NetworkTopologyStrategy with non-integer replication factor",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy:   "NetworkTopologyStrategy",
						DataCenterReplication: "dc1:three",
					},
				}},
			},
			expectedErrs: 2,
		},
		{
			description: "NetworkTopologyStrategy with even replication factor",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy:   "NetworkTopologyStrategy",
						DataCenterReplication: "dc1:4",
					},
				}},
			},
			expectedErrs: 1,
		},
		{
			description: "NetworkTopologyStrategy with mixed valid and invalid DCs",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy:   "NetworkTopologyStrategy",
						DataCenterReplication: "dc1:3,dc2:4", // dc2 is invalid
					},
				}},
			},
			expectedErrs: 1,
		},
		{
			description: "NetworkTopologyStrategy with multiple invalid entries",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy:   "NetworkTopologyStrategy",
						DataCenterReplication: "dc1:2,dc2:not-a-number,dc3:5",
					},
				}},
			},
			expectedErrs: 3,
		},
		{
			description: "NetworkTopologyStrategy with empty DataCenterReplication string",
			astarte: &Astarte{
				Spec: AstarteSpec{Cassandra: AstarteCassandraSpec{
					AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy:   "NetworkTopologyStrategy",
						DataCenterReplication: "",
					},
				}},
			},
			expectedErrs: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			r := tc.astarte
			err := r.validateCreateAstarteSystemKeyspace()

			g.Expect(err).To(HaveLen(tc.expectedErrs))
		})
	}
}
