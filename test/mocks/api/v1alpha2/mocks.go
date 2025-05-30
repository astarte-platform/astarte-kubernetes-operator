package v1alpha2mocks

import (
	. "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetResourceReqirementsMock() *v1.ResourceRequirements {
	return &v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1000m"),
			v1.ResourceMemory: resource.MustParse("1024M"),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("2000m"),
			v1.ResourceMemory: resource.MustParse("2048"),
		}}
}

func GetAstarteMock(name string, namespace string) *Astarte {
	return &Astarte{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: AstarteSpec{
			Version: "1.1.1",
			API: AstarteAPISpec{
				Host: "api.astarte.yourdomain.com",
			},
			RabbitMQ: AstarteRabbitMQSpec{
				AstarteGenericClusteredResource: AstarteGenericClusteredResource{
					Resources: GetResourceReqirementsMock(),
				},
			},
			Cassandra: AstarteCassandraSpec{
				MaxHeapSize: "1024M",
				HeapNewSize: "256M",
				Storage: &AstartePersistentStorageSpec{
					Size: resource.NewQuantity(30*1024*1024*1024, resource.BinarySI), // 30Gi
				},
				AstarteGenericClusteredResource: AstarteGenericClusteredResource{
					Resources: GetResourceReqirementsMock(),
				},
			},
			VerneMQ: AstarteVerneMQSpec{
				Host:        "broker.astarte.yourdomain.com",
				SSLListener: pointy.Bool(false),
				AstarteGenericClusteredResource: AstarteGenericClusteredResource{
					Resources: GetResourceReqirementsMock(),
				},
			},
			CFSSL: AstarteCFSSLSpec{
				Resources: GetResourceReqirementsMock(),
				Storage: &AstartePersistentStorageSpec{
					Size: resource.NewQuantity(2*1024*1024*1024, resource.BinarySI), // 2Gi
				},
			},
			Components: AstarteComponentsSpec{
				Resources: GetResourceReqirementsMock(),
			},
		},
	}
}
