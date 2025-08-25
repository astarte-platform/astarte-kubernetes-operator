/*
  This file is part of Astarte.

  Copyright 2020-23 SECO Mind Srl

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
	"strings"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"

	semver "github.com/Masterminds/semver/v3"
)

const (
	// Version is the Operator's version
	Version = "24.5.1"

	// AstarteVersionConstraintString represents the range of supported Astarte versions for this Operator.
	// If the Astarte version falls out of this range, reconciliation will be immediately aborted.
	// TODO: this logic should be moved to validation webhooks
	AstarteVersionConstraintString = ">= 1.0.0, < 1.3.0"

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
		// TODO: as of now, snapshots versions follow the major.minor-snapshot pattern.
		// Hence, this will never match the constraint version="snapshot".
		return CanManageSnapshot()
	}

	targetVersion, err := semver.NewVersion(v)
	if err != nil {
		return false
	}

	// Strip the prerelease
	*targetVersion, _ = targetVersion.SetPrerelease("")

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

// compareAstarteVersions compares two Astarte version strings (including snapshot versions).
// version as always greater than its corresponding base version (e.g., "1.2-snapshot" > "1.2").
// This behavior is a custom rule and is not aligned with the official semver spec, hence the custom implementation.
func CompareAstarteVersions(v1, v2 string) (int, error) {
	normalize := func(v string) (string, bool) {
		isSnapshot := strings.HasSuffix(v, "-snapshot")
		base := strings.TrimSuffix(v, "-snapshot")
		if isSnapshot && strings.Count(base, ".") == 1 {
			// If missing patch version, normalize (e.g., "1.2-snapshot" → "1.2.9999")
			base += ".9999"
		}
		return base, isSnapshot
	}

	base1, snap1 := normalize(v1)
	base2, snap2 := normalize(v2)

	ver1, err := semver.NewVersion(base1)
	if err != nil {
		return 0, err
	}
	ver2, err := semver.NewVersion(base2)
	if err != nil {
		return 0, err
	}

	if c := ver1.Compare(ver2); c != 0 {
		return c, nil
	}
	switch {
	case snap1 && !snap2:
		return 1, nil
	case !snap1 && snap2:
		return -1, nil
	default:
		return 0, nil
	}
}

// Erlang clustering is introduced in Astarte 1.2.1. For every
// Astarte version >= 1.2.1 (1.2-snapshots included), we need to set
// the environment variables needed for kubernetes-based clustering
func AstarteVersionImplementsErlangClustering(r *apiv1alpha2.Astarte) (err error, implements bool) {
	if r.Spec.Version == "" {
		// If no version is specified, we assume the latest and thus
		// we assume it implements clustering.
		return nil, true
	}

	comparison, err := CompareAstarteVersions(r.Spec.Version, "1.2.1")
	if err != nil {
		return err, false
	}

	// If comparison is >= 0, r.Spec.Version is >= 1.2.1
	return nil, comparison >= 0
}
