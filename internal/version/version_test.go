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

package version

import (
	"testing"
)

func TestCanManageVersion(t *testing.T) {
	testCases := []struct {
		version string
		allowed bool
	}{
		{"1.3.0", true},
		{"1.4.0", true},
		{"1.4.2", true},
		{"1.5.0", true},
		{"1.5.1", true},
		{"1.5.0-rc.0", true},
		{"1.6.0", false},
		{"1.2.9", false},
		{"snapshot", true},
	}

	for _, tc := range testCases {
		t.Run("version_"+tc.version, func(t *testing.T) {
			res := CanManageVersion(tc.version)
			if res != tc.allowed {
				t.Errorf("Expected CanManageVersion(%s) to be %v, got %v", tc.version, tc.allowed, res)
			}
		})
	}
}
