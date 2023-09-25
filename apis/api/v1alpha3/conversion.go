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

package v1alpha3

import (
	"k8s.io/apimachinery/pkg/conversion"

	"github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"
)

func Convert_v1alpha2_AstartePodPrioritiesSpec_To_v1alpha3_AstartePodPrioritiesSpec(in *v1alpha2.AstartePodPrioritiesSpec, out *AstartePodPrioritiesSpec, s conversion.Scope) error {
	if err := autoConvert_v1alpha2_AstartePodPrioritiesSpec_To_v1alpha3_AstartePodPrioritiesSpec(in, out, s); err != nil {
		return err
	}

	out.Enable = in.Enabled
	return nil
}

func Convert_v1alpha3_AstartePodPrioritiesSpec_To_v1alpha2_AstartePodPrioritiesSpec(in *AstartePodPrioritiesSpec, out *v1alpha2.AstartePodPrioritiesSpec, s conversion.Scope) error {
	if err := autoConvert_v1alpha3_AstartePodPrioritiesSpec_To_v1alpha2_AstartePodPrioritiesSpec(in, out, s); err != nil {
		return err
	}

	out.Enabled = in.Enable
	return nil
}

func Convert_v1alpha2_AstarteCassandraSSLConfigurationSpec_To_v1alpha3_AstarteCassandraSSLConfigurationSpec(in *v1alpha2.AstarteCassandraSSLConfigurationSpec, out *AstarteCassandraSSLConfigurationSpec, s conversion.Scope) error {
	if err := autoConvert_v1alpha2_AstarteCassandraSSLConfigurationSpec_To_v1alpha3_AstarteCassandraSSLConfigurationSpec(in, out, s); err != nil {
		return err
	}

	out.Enable = in.Enabled
	return nil
}

func Convert_v1alpha3_AstarteCassandraSSLConfigurationSpec_To_v1alpha2_AstarteCassandraSSLConfigurationSpec(in *AstarteCassandraSSLConfigurationSpec, out *v1alpha2.AstarteCassandraSSLConfigurationSpec, s conversion.Scope) error {
	if err := autoConvert_v1alpha3_AstarteCassandraSSLConfigurationSpec_To_v1alpha2_AstarteCassandraSSLConfigurationSpec(in, out, s); err != nil {
		return err
	}

	out.Enabled = in.Enable
	return nil
}

func Convert_v1alpha2_AstarteRabbitMQSSLConfigurationSpec_To_v1alpha3_AstarteRabbitMQSSLConfigurationSpec(in *v1alpha2.AstarteRabbitMQSSLConfigurationSpec, out *AstarteRabbitMQSSLConfigurationSpec, s conversion.Scope) error {
	if err := autoConvert_v1alpha2_AstarteRabbitMQSSLConfigurationSpec_To_v1alpha3_AstarteRabbitMQSSLConfigurationSpec(in, out, s); err != nil {
		return err
	}

	out.Enable = in.Enabled
	return nil
}

func Convert_v1alpha3_AstarteRabbitMQSSLConfigurationSpec_To_v1alpha2_AstarteRabbitMQSSLConfigurationSpec(in *AstarteRabbitMQSSLConfigurationSpec, out *v1alpha2.AstarteRabbitMQSSLConfigurationSpec, s conversion.Scope) error {
	if err := autoConvert_v1alpha3_AstarteRabbitMQSSLConfigurationSpec_To_v1alpha2_AstarteRabbitMQSSLConfigurationSpec(in, out, s); err != nil {
		return err
	}

	out.Enabled = in.Enable
	return nil
}
