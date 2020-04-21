/*
  This file is part of Astarte.

  Copyright 2020 Ispirata Srl

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
	"fmt"

	semver "github.com/Masterminds/semver/v3"
)

const (
	// Version is the Operator's version
	Version = "1.0.0-dev"

	// AstarteVersionConstraintString represents the range of supported Astarte versions for this Operator.
	// If the Astarte version falls out of this range, reconciliation will be immediately aborted.
	AstarteVersionConstraintString = ">= 0.10.0, < 1.1.0"

	// SnapshotVersion represents the name of the master/snapshot version, which can or cannot be installed
	// by this cluster
	SnapshotVersion = "snapshot"
)

// CanManageSnapshot returns whether the Operator can handle snapshot or not. We assume it can in case
// the Operator itself is a prerelease
func CanManageSnapshot() bool {
	operatorVersion, err := semver.NewVersion(Version)
	if err != nil {
		return false
	}

	return operatorVersion.Prerelease() != ""
}

// CanManageVersion returns whether the Operator can manage the given version.
func CanManageVersion(v string) bool {
	if v == SnapshotVersion {
		return CanManageSnapshot()
	}

	targetVersion, err := semver.NewVersion(v)
	if err != nil {
		return false
	}

	c, _ := semver.NewConstraint(AstarteVersionConstraintString)

	return c.Check(targetVersion)
}

// GetAstarteSemanticVersionFrom returns a semver object out of an Astarte version string, returning an
// error also if the version does not adhere to constraints or isn't supported by the Operator
func GetAstarteSemanticVersionFrom(v string) (*semver.Version, error) {
	// Build a SemVer out of the requested Astarte Version in the Spec.
	if !CanManageVersion(v) {
		return nil, fmt.Errorf("version %s cannot be managed by this Operator", v)
	}
	if v == SnapshotVersion {
		// Return the special snapshot semantic version
		return semver.NewVersion("999.99.9")
	}

	return semver.NewVersion(v)
}
