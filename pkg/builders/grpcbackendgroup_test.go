package builders

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	"github.com/yandex-cloud/alb-ingress/pkg/builders/mocks"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGrpcBackendGroup_BuildForCrd(t *testing.T) {
	targetGroupsBackend := &apploadbalancer.TargetGroupsBackend{
		TargetGroupIds: []string{
			"target-group-id",
		},
	}

	svc1 := &v12.Service{
		TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "service1"},
		Spec: v12.ServiceSpec{
			Type: v12.ServiceTypeNodePort,
			Ports: []v12.ServicePort{
				{
					Name:       "port_00_80",
					Port:       10000,
					TargetPort: intstr.IntOrString{IntVal: 8080},
					NodePort:   30080,
				},
				{
					Name:       "port_01_81",
					Port:       10001,
					TargetPort: intstr.IntOrString{IntVal: 8080},
					NodePort:   30081,
				},
				{
					Name:       "port_02_80",
					Port:       10002,
					TargetPort: intstr.IntOrString{IntVal: 8080},
					NodePort:   30080,
				},
			},
		},
	}

	duration := 64 * time.Millisecond

	sessionAffinityBackend := v1alpha1.GrpcBackend{
		Name:   "svc_back",
		Weight: 70,
		Service: &v1alpha1.ServiceBackend{
			Name: "service1",
			Port: v1alpha1.ServiceBackendPort{
				Number: 10001,
			},
		},
	}

	sessionAffinityALBBackend := apploadbalancer.GrpcBackend{
		Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
		BackendWeight: &wrapperspb.Int64Value{
			Value: 70,
		},
		Port: 30081,
		BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
			TargetGroups: targetGroupsBackend,
		},
		Healthchecks: defaultHealthChecks,
	}

	testData := []struct {
		desc         string
		backendGroup *v1alpha1.GrpcBackendGroup
		exp          *apploadbalancer.BackendGroup
		wantErr      bool
	}{
		{
			desc: "OK",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					Backends: []*v1alpha1.GrpcBackend{
						{
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Number: 10001,
								},
							},
							TLS: &v1alpha1.BackendTLS{
								Sni:       "my.fancy.srv",
								TrustedCa: "abcdefxxxx",
							},
						},
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						Backends: []*apploadbalancer.GrpcBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Healthchecks: defaultHealthChecks,
								Tls: &apploadbalancer.BackendTls{
									Sni: "my.fancy.srv",
									ValidationContext: &apploadbalancer.ValidationContext{
										TrustedCa: &apploadbalancer.ValidationContext_TrustedCaBytes{
											TrustedCaBytes: "abcdefxxxx",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "duplicated service backend",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					Backends: []*v1alpha1.GrpcBackend{
						{
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Number: 10001,
								},
							},
							TLS: &v1alpha1.BackendTLS{
								Sni:       "my.fancy.srv",
								TrustedCa: "abcdefxxxx",
							},
						},
						{
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Number: 10001,
								},
							},
							TLS: &v1alpha1.BackendTLS{
								Sni:       "my.fancy.srv",
								TrustedCa: "abcdefxxxx",
							},
						},
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						Backends: []*apploadbalancer.GrpcBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Healthchecks: defaultHealthChecks,
								Tls: &apploadbalancer.BackendTls{
									Sni: "my.fancy.srv",
									ValidationContext: &apploadbalancer.ValidationContext{
										TrustedCa: &apploadbalancer.ValidationContext_TrustedCaBytes{
											TrustedCaBytes: "abcdefxxxx",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "different service ports for the same node port",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					Backends: []*v1alpha1.GrpcBackend{
						{
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Number: 10000,
								},
							},
							TLS: &v1alpha1.BackendTLS{
								Sni:       "my.fancy.srv",
								TrustedCa: "abcdefxxxx",
							},
						},
						{
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Number: 10002,
								},
							},
							TLS: &v1alpha1.BackendTLS{
								Sni:       "my.fancy.srv",
								TrustedCa: "abcdefxxxx",
							},
						},
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						Backends: []*apploadbalancer.GrpcBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10000-30080",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30080,
								BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Healthchecks: defaultHealthChecks,
								Tls: &apploadbalancer.BackendTls{
									Sni: "my.fancy.srv",
									ValidationContext: &apploadbalancer.ValidationContext{
										TrustedCa: &apploadbalancer.ValidationContext_TrustedCaBytes{
											TrustedCaBytes: "abcdefxxxx",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "service not found",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					Backends: []*v1alpha1.GrpcBackend{
						{
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "svc-back70",
								Port: v1alpha1.ServiceBackendPort{
									Number: 10001,
								},
							},
							TLS: &v1alpha1.BackendTLS{
								Sni:       "my.fancy.srv",
								TrustedCa: "abcdefxxxx",
							},
						},
					},
				},
			},
			exp:     nil,
			wantErr: true,
		},
		{
			desc: "wrong service port",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					Backends: []*v1alpha1.GrpcBackend{
						{
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Number: 11111,
								},
							},
							TLS: &v1alpha1.BackendTLS{
								Sni:       "my.fancy.srv",
								TrustedCa: "abcdefxxxx",
							},
						},
					},
				},
			},
			exp:     nil,
			wantErr: true,
		},
		{
			desc: "named service ports",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					Backends: []*v1alpha1.GrpcBackend{
						{
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Name: "port_00_80",
								},
							},
							TLS: &v1alpha1.BackendTLS{
								Sni:       "my.fancy.srv",
								TrustedCa: "abcdefxxxx",
							},
						},
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						Backends: []*apploadbalancer.GrpcBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10000-30080",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30080,
								BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Healthchecks: defaultHealthChecks,
								Tls: &apploadbalancer.BackendTls{
									Sni: "my.fancy.srv",
									ValidationContext: &apploadbalancer.ValidationContext{
										TrustedCa: &apploadbalancer.ValidationContext_TrustedCaBytes{
											TrustedCaBytes: "abcdefxxxx",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "HeaderSessionAffinity",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					SessionAffinity: &v1alpha1.SessionAffinity{
						Header: &v1alpha1.SessionAffinityHeader{
							HeaderName: "foo",
						},
					},
					Backends: []*v1alpha1.GrpcBackend{
						&sessionAffinityBackend,
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						SessionAffinity: &apploadbalancer.GrpcBackendGroup_Header{
							Header: &apploadbalancer.HeaderSessionAffinity{
								HeaderName: "foo",
							},
						},
						Backends: []*apploadbalancer.GrpcBackend{
							&sessionAffinityALBBackend,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "CookieSessionAffinity",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					SessionAffinity: &v1alpha1.SessionAffinity{
						Cookie: &v1alpha1.SessionAffinityCookie{
							Name: "foo",
							TTL:  &metav1.Duration{Duration: duration},
						},
					},
					Backends: []*v1alpha1.GrpcBackend{
						&sessionAffinityBackend,
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						SessionAffinity: &apploadbalancer.GrpcBackendGroup_Cookie{
							Cookie: &apploadbalancer.CookieSessionAffinity{
								Name: "foo",
								Ttl:  durationpb.New(duration),
							},
						},
						Backends: []*apploadbalancer.GrpcBackend{
							&sessionAffinityALBBackend,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "ConnectionSessionAffinity",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					SessionAffinity: &v1alpha1.SessionAffinity{
						Connection: &v1alpha1.SessionAffinityConnection{
							SourceIP: true,
						},
					},
					Backends: []*v1alpha1.GrpcBackend{
						&sessionAffinityBackend,
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						SessionAffinity: &apploadbalancer.GrpcBackendGroup_Connection{
							Connection: &apploadbalancer.ConnectionSessionAffinity{
								SourceIp: true,
							},
						},
						Backends: []*apploadbalancer.GrpcBackend{
							&sessionAffinityALBBackend,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "BalancingMode",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					SessionAffinity: &v1alpha1.SessionAffinity{
						Connection: &v1alpha1.SessionAffinityConnection{
							SourceIP: true,
						},
					},
					Backends: []*v1alpha1.GrpcBackend{
						{
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Number: 10001,
								},
							},
							LoadBalancingConfig: &v1alpha1.LoadBalancingConfig{
								BalancerMode: "MAGLEV_HASH",
							},
						},
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						SessionAffinity: &apploadbalancer.GrpcBackendGroup_Connection{
							Connection: &apploadbalancer.ConnectionSessionAffinity{
								SourceIp: true,
							},
						},
						Backends: []*apploadbalancer.GrpcBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Healthchecks: defaultHealthChecks,
								LoadBalancingConfig: &apploadbalancer.LoadBalancingConfig{
									Mode: apploadbalancer.LoadBalancingMode_MAGLEV_HASH,
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "HealthChecks",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					Backends: []*v1alpha1.GrpcBackend{
						{
							HealthChecks: []*v1alpha1.HealthCheck{
								{
									GRPC: &v1alpha1.GrpcHealthCheck{
										ServiceName: "health",
									},
									Port:               ptr.To[int64](30080),
									UnhealthyThreshold: 5,
									HealthyThreshold:   5,
									Interval:           &metav1.Duration{Duration: time.Second * 5},
									Timeout:            &metav1.Duration{Duration: time.Second * 2},
								},
							},
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Number: 10001,
								},
							},
						},
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						Backends: []*apploadbalancer.GrpcBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Healthchecks: []*apploadbalancer.HealthCheck{
									{
										Timeout:            &durationpb.Duration{Seconds: 2},
										Interval:           &durationpb.Duration{Seconds: 5},
										HealthcheckPort:    30080,
										HealthyThreshold:   5,
										UnhealthyThreshold: 5,
										Healthcheck: &apploadbalancer.HealthCheck_Grpc{
											Grpc: &apploadbalancer.HealthCheck_GrpcHealthCheck{
												ServiceName: "health",
											},
										},
										TransportSettings: &apploadbalancer.HealthCheck_Plaintext{
											Plaintext: &apploadbalancer.PlaintextTransportSettings{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "HealthChecksNoPortSpecified",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					Backends: []*v1alpha1.GrpcBackend{
						{
							HealthChecks: []*v1alpha1.HealthCheck{
								{
									GRPC: &v1alpha1.GrpcHealthCheck{
										ServiceName: "health",
									},
									UnhealthyThreshold: 5,
									HealthyThreshold:   5,
									Interval:           &metav1.Duration{Duration: time.Second * 5},
									Timeout:            &metav1.Duration{Duration: time.Second * 2},
								},
							},
							Name:   "svc_back",
							Weight: 70,
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Number: 10001,
								},
							},
						},
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						Backends: []*apploadbalancer.GrpcBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Healthchecks: []*apploadbalancer.HealthCheck{
									{
										Timeout:            &durationpb.Duration{Seconds: 2},
										Interval:           &durationpb.Duration{Seconds: 5},
										HealthcheckPort:    30081,
										HealthyThreshold:   5,
										UnhealthyThreshold: 5,
										Healthcheck: &apploadbalancer.HealthCheck_Grpc{
											Grpc: &apploadbalancer.HealthCheck_GrpcHealthCheck{
												ServiceName: "health",
											},
										},
										TransportSettings: &apploadbalancer.HealthCheck_Plaintext{
											Plaintext: &apploadbalancer.PlaintextTransportSettings{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "HealthChecksTLS",
			backendGroup: &v1alpha1.GrpcBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "GrpcBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.GrpcBackendGroupSpec{
					Backends: []*v1alpha1.GrpcBackend{
						{
							HealthChecks: []*v1alpha1.HealthCheck{
								{
									GRPC: &v1alpha1.GrpcHealthCheck{
										ServiceName: "health",
									},
									Port:               ptr.To[int64](30080),
									UnhealthyThreshold: 5,
									HealthyThreshold:   5,
									Interval:           &metav1.Duration{Duration: time.Second * 5},
									Timeout:            &metav1.Duration{Duration: time.Second * 2},
								},
							},
							Name:   "svc_back",
							Weight: 70,
							TLS: &v1alpha1.BackendTLS{
								Sni:       "sni",
								TrustedCa: "abcdefxxxxx",
							},
							Service: &v1alpha1.ServiceBackend{
								Name: "service1",
								Port: v1alpha1.ServiceBackendPort{
									Number: 10001,
								},
							},
						},
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Grpc{
					Grpc: &apploadbalancer.GrpcBackendGroup{
						Backends: []*apploadbalancer.GrpcBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Tls: &apploadbalancer.BackendTls{
									Sni: "sni",
									ValidationContext: &apploadbalancer.ValidationContext{
										TrustedCa: &apploadbalancer.ValidationContext_TrustedCaBytes{
											TrustedCaBytes: "abcdefxxxxx",
										},
									},
								},
								Healthchecks: []*apploadbalancer.HealthCheck{
									{
										Timeout:            &durationpb.Duration{Seconds: 2},
										Interval:           &durationpb.Duration{Seconds: 5},
										HealthcheckPort:    30080,
										HealthyThreshold:   5,
										UnhealthyThreshold: 5,
										Healthcheck: &apploadbalancer.HealthCheck_Grpc{
											Grpc: &apploadbalancer.HealthCheck_GrpcHealthCheck{
												ServiceName: "health",
											},
										},
										TransportSettings: &apploadbalancer.HealthCheck_Tls{
											Tls: &apploadbalancer.SecureTransportSettings{
												Sni: "sni",
												ValidationContext: &apploadbalancer.ValidationContext{
													TrustedCa: &apploadbalancer.ValidationContext_TrustedCaBytes{
														TrustedCaBytes: "abcdefxxxxx",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	cli := fake.NewClientBuilder().WithObjects(svc1).Build()
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			tgRepo := mocks.NewMockTargetGroupFinder(ctrl)
			tgRepo.EXPECT().FindTargetGroup(gomock.Any(), gomock.Any()).AnyTimes().Return(&apploadbalancer.TargetGroup{
				Id: "target-group-id",
			}, nil)

			b := GrpcBackendGroupForCrdBuilder{
				FolderID: "my-folder",
				Names:    &metadata.Names{ClusterID: "my-cluster"},
				Cli:      cli,
				Repo:     tgRepo,
			}

			res, err := b.BuildForCrd(context.Background(), tc.backendGroup)
			require.True(t, (err != nil) == tc.wantErr)
			if tc.wantErr {
				return
			}
			assert.Condition(t, func() bool { return proto.Equal(tc.exp, res) }, "backend groups mismatch\nexp %v\ngot %v", tc.exp, res)
		})
	}
}
