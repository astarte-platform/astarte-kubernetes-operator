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

const (
	// Version is the Operator's version
	Version = "0.11.0-rc.0"

	// AstarteVersionConstraintString represents the range of supported Astarte versions for this Operator.
	// If the Astarte version falls out of this range, reconciliation will be immediately aborted.
	AstarteVersionConstraintString = ">= 0.10.0, < 0.12.0"
)
