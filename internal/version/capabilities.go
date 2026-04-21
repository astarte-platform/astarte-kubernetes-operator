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
	"github.com/Masterminds/semver/v3"
)

// Feature acts as an enum for togglable Astarte capabilities
type Feature string

const (
	// Vault represents the HashiCorp Vault support
	Vault Feature = "Vault"
)

// matrix stores the pre-compiled semver constraints for each feature
var matrix map[Feature]*semver.Constraints

// Checker evaluates if a specific Astarte version supports certain features.
type Checker struct {
	version *semver.Version
}

// NewChecker safely parses the Astarte version and returns a Checker.
// It uses the same version handling as the rest of the operator, including
// support for the "snapshot" version string and prerelease stripping.
func NewChecker(versionStr string) (*Checker, error) {
	v, err := GetAstarteSemanticVersionFrom(versionStr)
	if err != nil {
		return nil, err
	}
	return &Checker{version: v}, nil
}

func init() {
	matrix = map[Feature]*semver.Constraints{
		// Vault is supported strictly in 1.4.0 and above
		Vault: mustParseConstraint(">= 1.4.0"),
	}
}

// mustParseConstraint is a local helper that panics if a hardcoded constraint is invalid.
func mustParseConstraint(c string) *semver.Constraints {
	constraint, err := semver.NewConstraint(c)
	if err != nil {
		panic("invalid semver constraint in capabilities matrix: " + err.Error())
	}
	return constraint
}

// Supports evaluates if the configured version satisfies the feature's constraint.
func (c *Checker) Supports(f Feature) bool {
	constraint, exists := matrix[f]
	if !exists {
		// Default to false if a feature isn't in the matrix
		return false
	}
	return constraint.Check(c.version)
}
