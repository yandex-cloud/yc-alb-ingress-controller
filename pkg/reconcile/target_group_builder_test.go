package reconcile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestServiceTargetGroupBuilder_Build(t *testing.T) {
	const folderID = "abcdef"
	instance1 := &compute.Instance{
		Id: "fhmgp9rcnotn10g8xxxx",
		NetworkInterfaces: []*compute.NetworkInterface{{
			PrimaryV4Address: &compute.PrimaryAddress{
				Address: "192.168.10.30",
			},
			SubnetId: "subnet_1",
		}},
	}
	instance2 := &compute.Instance{
		Id: "fhmgp9rcnotn10g8yyyy",
		NetworkInterfaces: []*compute.NetworkInterface{{
			PrimaryV4Address: &compute.PrimaryAddress{
				Address: "192.168.10.28",
			},
			PrimaryV6Address: &compute.PrimaryAddress{
				Address: "2001:0db8:0001:0000:0000:0ab9:C0A8:0102",
			},
			SubnetId: "subnet_2",
		}},
	}
	instances := map[string]*compute.Instance{instance1.Id: instance1, instance2.Id: instance2}

	fn := func(ctx context.Context, id string) (instance *compute.Instance, err error) {
		if instance = instances[id]; instance == nil {
			err = status.Errorf(codes.NotFound, "instance %s not found", id)
		}
		return
	}

	testData := []struct {
		desc string

		objects []client.Object

		expTargets []*apploadbalancer.Target
		wantErr    bool

		useEndpointSlices bool
	}{
		{
			desc:              "OK, slices",
			useEndpointSlices: true,
			objects: []client.Object{
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc",
						Namespace: "default",
					},
				},
				&discovery.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"kubernetes.io/service-name": "svc",
						},
						Namespace: "default",
						Name:      "svc1",
					},
					Endpoints: []discovery.Endpoint{
						{
							NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-inod"),
						},
					},
				},
				&discovery.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"kubernetes.io/service-name": "svc",
						},
						Namespace: "default",
						Name:      "svc2",
					},
					Endpoints: []discovery.Endpoint{
						{
							NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-ydoj"),
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-inod"},
					Spec: v1.NodeSpec{
						ProviderID: "yandex://fhmgp9rcnotn10g8xxxx",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{Type: v1.NodeExternalIP, Address: "51.250.7.19"},
							{Type: v1.NodeInternalIP, Address: "192.168.10.30"},
							{Type: v1.NodeHostName, Address: "cl1mkq03gu56o26iia82-inod"},
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-ydoj"},
					Spec: v1.NodeSpec{
						ProviderID: "yandex://fhmgp9rcnotn10g8yyyy",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{Type: v1.NodeExternalIP, Address: "51.250.12.174"},
							{Type: v1.NodeInternalIP, Address: "192.168.10.28"},
							{Type: v1.NodeHostName, Address: "cl1mkq03gu56o26iia82-ydoj"},
						},
					},
				},
			},
			expTargets: []*apploadbalancer.Target{{
				AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "192.168.10.30"},
				SubnetId:    "subnet_1",
			}, {
				AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "192.168.10.28"},
				SubnetId:    "subnet_2",
			}},
		},
		{
			desc:              "fail, slices",
			useEndpointSlices: true,
			objects: []client.Object{
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc",
						Namespace: "default",
					},
				},
				&discovery.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"kubernetes.io/service-name": "svc",
						},
						Namespace: "default",
						Name:      "svc2",
					},
					Endpoints: []discovery.Endpoint{
						{
							NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-inod"),
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-inod"},
					Spec:       v1.NodeSpec{},
				},
			},
			wantErr: true,
		},
		{
			desc:              "Instance not found, slices",
			useEndpointSlices: true,
			objects: []client.Object{
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc",
						Namespace: "default",
					},
				},
				&discovery.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"kubernetes.io/service-name": "svc",
						},
						Namespace: "default",
						Name:      "svc1",
					},
					Endpoints: []discovery.Endpoint{
						{
							NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-inod"),
						},
					},
				},
				&discovery.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"kubernetes.io/service-name": "svc",
						},
						Namespace: "default",
						Name:      "svc2",
					},
					Endpoints: []discovery.Endpoint{
						{
							NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-ydoj"),
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-inod"},
					Spec: v1.NodeSpec{
						ProviderID: "yandex://fhmgp9rcnotn10g8zzzz",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{Type: v1.NodeExternalIP, Address: "51.250.7.19"},
							{Type: v1.NodeInternalIP, Address: "192.168.10.30"},
							{Type: v1.NodeHostName, Address: "cl1mkq03gu56o26iia82-inod"},
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-ydoj"},
					Spec: v1.NodeSpec{
						ProviderID: "yandex://fhmgp9rcnotn10g8yyyy",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{Type: v1.NodeExternalIP, Address: "51.250.12.174"},
							{Type: v1.NodeInternalIP, Address: "192.168.10.28"},
							{Type: v1.NodeHostName, Address: "cl1mkq03gu56o26iia82-ydoj"},
						},
					},
				},
			},
			expTargets: []*apploadbalancer.Target{{
				AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "192.168.10.28"},
				SubnetId:    "subnet_2",
			}},
		},
		{
			desc:              "OK, endpoints",
			useEndpointSlices: false,
			objects: []client.Object{
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc",
						Namespace: "default",
					},
				},
				&v1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "svc",
					},
					Subsets: []v1.EndpointSubset{
						{
							Addresses: []v1.EndpointAddress{
								{
									NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-inod"),
								},
								{
									NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-ydoj"),
								},
							},
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-inod"},
					Spec: v1.NodeSpec{
						ProviderID: "yandex://fhmgp9rcnotn10g8xxxx",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{Type: v1.NodeExternalIP, Address: "51.250.7.19"},
							{Type: v1.NodeInternalIP, Address: "192.168.10.30"},
							{Type: v1.NodeHostName, Address: "cl1mkq03gu56o26iia82-inod"},
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-ydoj"},
					Spec: v1.NodeSpec{
						ProviderID: "yandex://fhmgp9rcnotn10g8yyyy",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{Type: v1.NodeExternalIP, Address: "51.250.12.174"},
							{Type: v1.NodeInternalIP, Address: "192.168.10.28"},
							{Type: v1.NodeHostName, Address: "cl1mkq03gu56o26iia82-ydoj"},
						},
					},
				},
			},
			expTargets: []*apploadbalancer.Target{{
				AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "192.168.10.30"},
				SubnetId:    "subnet_1",
			}, {
				AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "192.168.10.28"},
				SubnetId:    "subnet_2",
			}},
		},
		{
			desc:              "fail, endpoints",
			useEndpointSlices: false,
			objects: []client.Object{
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc",
						Namespace: "default",
					},
				},
				&v1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "svc",
					},
					Subsets: []v1.EndpointSubset{
						{
							Addresses: []v1.EndpointAddress{
								{
									NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-inod"),
								},
							},
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-inod"},
					Spec:       v1.NodeSpec{},
				},
			},
			wantErr: true,
		},
		{
			desc:              "Instance not found, endpoints",
			useEndpointSlices: false,
			objects: []client.Object{
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc",
						Namespace: "default",
					},
				},
				&v1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "svc",
					},
					Subsets: []v1.EndpointSubset{
						{
							Addresses: []v1.EndpointAddress{
								{
									NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-inod"),
								},
								{
									NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-ydoj"),
								},
							},
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-inod"},
					Spec: v1.NodeSpec{
						ProviderID: "yandex://fhmgp9rcnotn10g8zzzz",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{Type: v1.NodeExternalIP, Address: "51.250.7.19"},
							{Type: v1.NodeInternalIP, Address: "192.168.10.30"},
							{Type: v1.NodeHostName, Address: "cl1mkq03gu56o26iia82-inod"},
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-ydoj"},
					Spec: v1.NodeSpec{
						ProviderID: "yandex://fhmgp9rcnotn10g8yyyy",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{Type: v1.NodeExternalIP, Address: "51.250.12.174"},
							{Type: v1.NodeInternalIP, Address: "192.168.10.28"},
							{Type: v1.NodeHostName, Address: "cl1mkq03gu56o26iia82-ydoj"},
						},
					},
				},
			},
			expTargets: []*apploadbalancer.Target{{
				AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "192.168.10.28"},
				SubnetId:    "subnet_2",
			}},
		},
		{
			desc:              "prefer IPv6, slices",
			useEndpointSlices: true,
			objects: []client.Object{
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc",
						Namespace: "default",
						Annotations: map[string]string{
							k8s.PreferIPv6Targets: "true",
						},
					},
				},
				&discovery.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"kubernetes.io/service-name": "svc",
						},
						Namespace: "default",
						Name:      "svc234",
					},
					Endpoints: []discovery.Endpoint{
						{
							NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-ydoj"),
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-ydoj"},
					Spec: v1.NodeSpec{
						ProviderID: "yandex://fhmgp9rcnotn10g8yyyy",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{Type: v1.NodeInternalIP, Address: "51.250.12.174"},
							{Type: v1.NodeInternalIP, Address: "2001:0db8:0001:0000:0000:0ab9:C0A8:0102"},
							{Type: v1.NodeHostName, Address: "cl1mkq03gu56o26iia82-ydoj"},
						},
					},
				},
			},
			expTargets: []*apploadbalancer.Target{{
				AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "2001:0db8:0001:0000:0000:0ab9:C0A8:0102"},
				SubnetId:    "subnet_2",
			},
			},
		},
		{
			desc:              "prefer IPv6 without IPv6, slices",
			useEndpointSlices: true,
			objects: []client.Object{
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc",
						Namespace: "default",
						Annotations: map[string]string{
							k8s.PreferIPv6Targets: "true",
						},
					},
				},
				&discovery.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"kubernetes.io/service-name": "svc",
						},
						Namespace: "default",
						Name:      "svc234",
					},
					Endpoints: []discovery.Endpoint{
						{
							NodeName: pointer.StringPtr("cl1mkq03gu56o26iia82-ydoj"),
						},
					},
				},
				&v1.Node{
					TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "cl1mkq03gu56o26iia82-ydoj"},
					Spec: v1.NodeSpec{
						ProviderID: "yandex://fhmgp9rcnotn10g8yyyy",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{Type: v1.NodeInternalIP, Address: "192.168.10.28"},
							{Type: v1.NodeHostName, Address: "cl1mkq03gu56o26iia82-ydoj"},
						},
					},
				},
			},
			expTargets: []*apploadbalancer.Target{{
				AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "192.168.10.28"},
				SubnetId:    "subnet_2",
			},
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			cli := fake.NewClientBuilder().WithObjects(tc.objects...).Build()
			b := NewTargetGroupBuilder(folderID, cli, &metadata.Names{}, &metadata.Labels{}, fn, tc.useEndpointSlices)
			tg, err := b.Build(context.Background(), types.NamespacedName{
				Name:      "svc",
				Namespace: "default",
			})

			assert.Equal(t, tc.wantErr, err != nil, "err should not be nil: %v, got %v", tc.wantErr, err)
			if !tc.wantErr {
				assert.Equal(t, tc.expTargets, tg.GetTargets())
			}
		})
	}
}
