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

package deps

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
)

// This test makes no sense as CFSSL version is hardcoded in the function.
// We keep this here just as a reminder to update the test if we ever decide to
// make changes to GetDefaultVersionForCFSSL.
func TestGetDefaultVersionForCFSSL(t *testing.T) {
	tests := []struct {
		astarteVersion string
		expected       string
	}{
		{
			astarteVersion: "1.3.0",
			expected:       "1.5.0-astarte.3",
		},
	}

	g := NewGomegaWithT(t)
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Astarte %s", tt.astarteVersion), func(t *testing.T) {
			version := GetDefaultVersionForCFSSL(tt.astarteVersion)
			g.Expect(version).To(Equal(tt.expected))
		})
	}
}
