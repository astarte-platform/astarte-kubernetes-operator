/*
This file is part of Astarte.

Copyright 2024 SECO Mind Srl.

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
	"errors"

	semver "github.com/Masterminds/semver/v3"
)

var (
	// ErrConstraintNotSatisfied means the check happened correctly, but the constraint wasn't satisfied
	ErrConstraintNotSatisfied = errors.New("constraint not satisfied")
)

// CheckConstraintAgainstAstarteVersion validates a given Astarte version against a given constraint. Returns nil if
// the constraint is satisfied, an error otherwise
func CheckConstraintAgainstAstarteVersion(constraint, v string) error {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return err
	}
	semVer, err := GetAstarteSemanticVersionFrom(v)
	if err != nil {
		return err
	}
	*semVer, err = semVer.SetPrerelease("")
	if err != nil {
		return err
	}

	if !c.Check(semVer) {
		return ErrConstraintNotSatisfied
	}

	return nil
}

// CheckConstraintAgainstAstarteComponentVersion checks a constraint against a specialized Astarte component version
func CheckConstraintAgainstAstarteComponentVersion(constraint, componentVersion, astarteVersion string) error {
	versionString := GetVersionForAstarteComponent(astarteVersion, componentVersion)
	return CheckConstraintAgainstAstarteVersion(constraint, versionString)
}

// GetVersionForAstarteComponent returns the version for a given Astarte Component
func GetVersionForAstarteComponent(astarteVersion, componentVersion string) string {
	if componentVersion != "" {
		return componentVersion
	}
	return astarteVersion
}
