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

package v1alpha3

import (
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
)

// ConvertTo converts this Astarte to the Hub version (v1alpha2).
func (src *Astarte) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha2.Astarte)

	return Convert_v1alpha3_Astarte_To_v1alpha2_Astarte(src, dst, nil)
}

// ConvertFrom converts from the Hub version (v1alpha2) to this version.
func (dst *Astarte) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha2.Astarte)

	return Convert_v1alpha2_Astarte_To_v1alpha3_Astarte(src, dst, nil)
}

func Convert_v1alpha2_AstartePodPrioritiesSpec_To_v1alpha3_AstartePodPrioritiesSpec(in *v1alpha2.AstartePodPrioritiesSpec, out *AstartePodPrioritiesSpec, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha2_AstartePodPrioritiesSpec_To_v1alpha3_AstartePodPrioritiesSpec(in, out, s); err != nil {
		return err
	}

	out.Enable = in.Enabled
	return nil
}

func Convert_v1alpha3_AstartePodPrioritiesSpec_To_v1alpha2_AstartePodPrioritiesSpec(in *AstartePodPrioritiesSpec, out *v1alpha2.AstartePodPrioritiesSpec, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha3_AstartePodPrioritiesSpec_To_v1alpha2_AstartePodPrioritiesSpec(in, out, s); err != nil {
		return err
	}

	out.Enabled = in.Enable
	return nil
}

func Convert_v1alpha2_AstarteCassandraSSLConfigurationSpec_To_v1alpha3_AstarteCassandraSSLConfigurationSpec(in *v1alpha2.AstarteCassandraSSLConfigurationSpec, out *AstarteCassandraSSLConfigurationSpec, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha2_AstarteCassandraSSLConfigurationSpec_To_v1alpha3_AstarteCassandraSSLConfigurationSpec(in, out, s); err != nil {
		return err
	}

	out.Enable = in.Enabled
	return nil
}

func Convert_v1alpha3_AstarteCassandraSSLConfigurationSpec_To_v1alpha2_AstarteCassandraSSLConfigurationSpec(in *AstarteCassandraSSLConfigurationSpec, out *v1alpha2.AstarteCassandraSSLConfigurationSpec, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha3_AstarteCassandraSSLConfigurationSpec_To_v1alpha2_AstarteCassandraSSLConfigurationSpec(in, out, s); err != nil {
		return err
	}

	out.Enabled = in.Enable
	return nil
}

func Convert_v1alpha2_AstarteRabbitMQSSLConfigurationSpec_To_v1alpha3_AstarteRabbitMQSSLConfigurationSpec(in *v1alpha2.AstarteRabbitMQSSLConfigurationSpec, out *AstarteRabbitMQSSLConfigurationSpec, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha2_AstarteRabbitMQSSLConfigurationSpec_To_v1alpha3_AstarteRabbitMQSSLConfigurationSpec(in, out, s); err != nil {
		return err
	}

	out.Enable = in.Enabled
	return nil
}

func Convert_v1alpha3_AstarteRabbitMQSSLConfigurationSpec_To_v1alpha2_AstarteRabbitMQSSLConfigurationSpec(in *AstarteRabbitMQSSLConfigurationSpec, out *v1alpha2.AstarteRabbitMQSSLConfigurationSpec, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha3_AstarteRabbitMQSSLConfigurationSpec_To_v1alpha2_AstarteRabbitMQSSLConfigurationSpec(in, out, s); err != nil {
		return err
	}

	out.Enabled = in.Enable
	return nil
}
