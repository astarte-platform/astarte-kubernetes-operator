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

package deps

import semver "github.com/Masterminds/semver/v3"

// GetDefaultVersionForCFSSL returns the default CFSSL version based on the Astarte version requested
func GetDefaultVersionForCFSSL(astarteVersion *semver.Version) string {
	checkVersion, _ := astarteVersion.SetPrerelease("")
	c, _ := semver.NewConstraint("< 0.11.0")
	if c.Check(&checkVersion) {
		return "1.0.0-astarte.0"
	}

	return "1.4.1-astarte.0"
}

// GetDefaultVersionForCassandra returns the default Cassandra version based on the Astarte version requested
func GetDefaultVersionForCassandra(astarteVersion *semver.Version) string {
	// TODO: We should change this to the official images
	return "v13"
}

// GetDefaultVersionForRabbitMQ returns the default RabbitMQ version based on the Astarte version requested
func GetDefaultVersionForRabbitMQ(astarteVersion *semver.Version) string {
	checkVersion, _ := astarteVersion.SetPrerelease("")
	c, _ := semver.NewConstraint("< 0.11.0")
	if c.Check(&checkVersion) {
		return "3.7.15"
	}

	return "3.7.21"
}
