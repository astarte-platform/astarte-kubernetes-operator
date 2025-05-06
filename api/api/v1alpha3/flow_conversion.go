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

package v1alpha3

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
)

// ConvertTo converts this Flow to the Hub version (v1alpha2).
func (src *Flow) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha2.Flow)

	return Convert_v1alpha3_Flow_To_v1alpha2_Flow(src, dst, nil)
}

// ConvertFrom converts from the Hub version (v1alpha2) to this version.
func (dst *Flow) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha2.Flow)

	return Convert_v1alpha2_Flow_To_v1alpha3_Flow(src, dst, nil)
}
