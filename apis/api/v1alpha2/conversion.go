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

package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
)

// ConvertTo converts this Astarte to the Hub version (v1alpha1).
func (src *Astarte) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.Astarte)
	err := Convert_v1alpha2_Astarte_To_v1alpha1_Astarte(src, dst, nil)
	if err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (dst *Astarte) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.Astarte)
	err := Convert_v1alpha1_Astarte_To_v1alpha2_Astarte(src, dst, nil)
	if err != nil {
		return err
	}
	return nil
}

// ConvertTo converts this AstarteVoyagerIngress to the Hub version (v1alpha1).
func (src *AstarteVoyagerIngress) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.AstarteVoyagerIngress)

	err := Convert_v1alpha2_AstarteVoyagerIngress_To_v1alpha1_AstarteVoyagerIngress(src, dst, nil)
	if err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (dst *AstarteVoyagerIngress) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.AstarteVoyagerIngress)

	err := Convert_v1alpha1_AstarteVoyagerIngress_To_v1alpha2_AstarteVoyagerIngress(src, dst, nil)
	if err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this Flow to the Hub version (v1alpha1).
func (src *Flow) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.Flow)

	err := Convert_v1alpha2_Flow_To_v1alpha1_Flow(src, dst, nil)
	if err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (dst *Flow) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.Flow)

	err := Convert_v1alpha1_Flow_To_v1alpha2_Flow(src, dst, nil)
	if err != nil {
		return err
	}

	return nil
}
