package builders

import (
	"testing"
	"time"

	"k8s.io/utils/ptr"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	"github.com/yandex-cloud/alb-ingress/pkg/builders/mocks"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
)

func TestBackendGroups(t *testing.T) {
	var (
		port0080 = v12.ServicePort{
			Name:       "port_00_80",
			Port:       10000,
			TargetPort: intstr.IntOrString{IntVal: 8080},
			NodePort:   30080,
		}

		port0081 = v12.ServicePort{
			Name:       "port_00_81",
			Port:       10000,
			TargetPort: intstr.IntOrString{IntVal: 8080},
			NodePort:   30081,
		}
	)

	svc1 := &v12.Service{
		TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "service1"},
		Spec: v12.ServiceSpec{
			Type: v12.ServiceTypeNodePort,
			Ports: []v12.ServicePort{
				port0080, port0081,
			},
		},
	}

	targetGroupsBackend := &apploadbalancer.TargetGroupsBackend{
		TargetGroupIds: []string{
			"target-group-id",
		},
	}

	var (
		basicBG = &apploadbalancer.BackendGroup{
			Name:     "bg-5d6f6ba020fd6ad14f8379b75035170ff8070c7c-30080",
			FolderId: "my-folder",
			Backend: &apploadbalancer.BackendGroup_Http{
				Http: &apploadbalancer.HttpBackendGroup{
					Backends: []*apploadbalancer.HttpBackend{
						{
							Name:          "backend-62cb79b453225af83e214cb4d193552084344e26-10000-30080",
							BackendWeight: &wrapperspb.Int64Value{Value: 1},
							Port:          30080,
							BackendType: &apploadbalancer.HttpBackend_TargetGroups{
								TargetGroups: targetGroupsBackend,
							},
							Healthchecks: defaultHealthChecks,
						},
						{
							Name:          "backend-62cb79b453225af83e214cb4d193552084344e26-10000-30081",
							BackendWeight: &wrapperspb.Int64Value{Value: 1},
							Port:          30081,
							BackendType: &apploadbalancer.HttpBackend_TargetGroups{
								TargetGroups: targetGroupsBackend,
							},
							Healthchecks: defaultHealthChecks,
						},
					},
				},
			},
		}

		secureBGHC = []*apploadbalancer.HealthCheck{{
			Timeout:         &durationpb.Duration{Seconds: 10},
			Interval:        &durationpb.Duration{Seconds: 20},
			HealthcheckPort: 30100,
			Healthcheck: &apploadbalancer.HealthCheck_Http{
				Http: &apploadbalancer.HealthCheck_HttpHealthCheck{
					Path: "/health-1",
				},
			},
			TransportSettings: &apploadbalancer.HealthCheck_Plaintext{
				Plaintext: &apploadbalancer.PlaintextTransportSettings{},
			},
		}}

		secureBG = &apploadbalancer.BackendGroup{
			Name:     "bg-5d6f6ba020fd6ad14f8379b75035170ff8070c7c-30080",
			FolderId: "my-folder",
			Backend: &apploadbalancer.BackendGroup_Http{
				Http: &apploadbalancer.HttpBackendGroup{
					Backends: []*apploadbalancer.HttpBackend{
						{
							Name:          "backend-62cb79b453225af83e214cb4d193552084344e26-10000-30080",
							BackendWeight: &wrapperspb.Int64Value{Value: 1},
							Port:          30080,
							BackendType: &apploadbalancer.HttpBackend_TargetGroups{
								TargetGroups: targetGroupsBackend,
							},
							Healthchecks: secureBGHC,
							Tls:          &apploadbalancer.BackendTls{},
						},
						{
							Name:          "backend-62cb79b453225af83e214cb4d193552084344e26-10000-30081",
							BackendWeight: &wrapperspb.Int64Value{Value: 1},
							Port:          30081,
							BackendType: &apploadbalancer.HttpBackend_TargetGroups{
								TargetGroups: targetGroupsBackend,
							},
							Healthchecks: secureBGHC,
							Tls:          &apploadbalancer.BackendTls{},
						},
					},
				},
			},
		}

		http2BG = &apploadbalancer.BackendGroup{
			Name:     "bg-5d6f6ba020fd6ad14f8379b75035170ff8070c7c-30080",
			FolderId: "my-folder",
			Backend: &apploadbalancer.BackendGroup_Http{
				Http: &apploadbalancer.HttpBackendGroup{
					Backends: []*apploadbalancer.HttpBackend{
						{
							Name:          "backend-62cb79b453225af83e214cb4d193552084344e26-10000-30080",
							BackendWeight: &wrapperspb.Int64Value{Value: 1},
							Port:          30080,
							BackendType: &apploadbalancer.HttpBackend_TargetGroups{
								TargetGroups: targetGroupsBackend,
							},
							Healthchecks: defaultHealthChecks,
							UseHttp2:     true,
						},
						{
							Name:          "backend-62cb79b453225af83e214cb4d193552084344e26-10000-30081",
							BackendWeight: &wrapperspb.Int64Value{Value: 1},
							Port:          30081,
							BackendType: &apploadbalancer.HttpBackend_TargetGroups{
								TargetGroups: targetGroupsBackend,
							},
							Healthchecks: defaultHealthChecks,
							UseHttp2:     true,
						},
					},
				},
			},
		}

		grpcBG = &apploadbalancer.BackendGroup{
			Name:     "bg-5d6f6ba020fd6ad14f8379b75035170ff8070c7c-30080",
			FolderId: "my-folder",
			Backend: &apploadbalancer.BackendGroup_Grpc{
				Grpc: &apploadbalancer.GrpcBackendGroup{
					Backends: []*apploadbalancer.GrpcBackend{
						{
							Name:          "backend-62cb79b453225af83e214cb4d193552084344e26-10000-30080",
							BackendWeight: &wrapperspb.Int64Value{Value: 1},
							Port:          30080,
							BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
								TargetGroups: targetGroupsBackend,
							},
							Healthchecks: defaultHealthChecks,
						},
						{
							Name:          "backend-62cb79b453225af83e214cb4d193552084344e26-10000-30081",
							BackendWeight: &wrapperspb.Int64Value{Value: 1},
							Port:          30081,
							BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
								TargetGroups: targetGroupsBackend,
							},
							Healthchecks: defaultHealthChecks,
						},
					},
				},
			},
		}

		testData = []struct {
			desc      string
			opts      BackendResolveOpts
			nodePorts []v12.ServicePort
			svc       *v12.Service
			exp       *apploadbalancer.BackendGroup
			wantError bool
		}{
			{
				desc: "basic",
				opts: BackendResolveOpts{healthChecks: defaultHealthChecks},
				svc:  svc1,
				exp:  basicBG,
				nodePorts: []v12.ServicePort{
					port0080,
					port0081,
				},
			},
			{
				desc: "grpc",
				svc:  svc1,
				exp:  grpcBG,
				opts: BackendResolveOpts{
					BackendType: GRPC, healthChecks: defaultHealthChecks,
				},
				nodePorts: []v12.ServicePort{
					port0080,
					port0081,
				},
			},
			{
				desc: "http2",
				svc:  svc1,
				exp:  http2BG,
				opts: BackendResolveOpts{
					BackendType: HTTP2, healthChecks: defaultHealthChecks,
				},
				nodePorts: []v12.ServicePort{
					port0080,
					port0081,
				},
			},
			{
				desc: "secure backend",
				svc:  svc1,
				exp:  secureBG,
				opts: BackendResolveOpts{
					BackendType:  HTTP,
					Secure:       true,
					healthChecks: secureBGHC,
				},
				nodePorts: []v12.ServicePort{
					port0080,
					port0081,
				},
			},
		}
	)

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			b := BackendGroupBuilder{
				FolderID: "my-folder",
				Names:    &metadata.Names{ClusterID: "my-cluster"},
			}

			bg, err := b.build(tc.svc, tc.nodePorts, "target-group-id", tc.opts)
			require.True(t, (err != nil) == tc.wantError)
			if tc.wantError {
				return
			}

			assertBackendGroups := func(exp, got *apploadbalancer.BackendGroup) {
				b1, b2 := exp.GetBackend(), got.GetBackend()
				if b1 == nil || b2 == nil {
					require.True(t, (b1 == nil) == (b2 == nil), "exp nil: %v, got nil: %v", b1 == nil, b2 == nil)
				}

				switch t1 := b1.(type) {
				case *apploadbalancer.BackendGroup_Http:
					t2, ok := b2.(*apploadbalancer.BackendGroup_Http)
					require.True(t, ok, "got backend other than %s", "HTTP")
					assert.Condition(t, func() bool { return proto.Equal(t1.Http, t2.Http) }, "backends of group mismatch\nexp %v\ngot %v", t1.Http, t2.Http)
				case *apploadbalancer.BackendGroup_Grpc:
					t2, ok := b2.(*apploadbalancer.BackendGroup_Grpc)
					require.True(t, ok, "got backend other than %s", "GRPC")
					assert.Condition(t, func() bool { return proto.Equal(t1.Grpc, t2.Grpc) }, "backends of group mismatch\nexp %v\ngot %v", t1.Grpc, t2.Grpc)
				}
			}

			assertBackendGroups(bg, tc.exp)
		})
	}
}

func TestBackendGroups_BuildBgFromCR(t *testing.T) {
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

	sessionAffinityBackend := v1alpha1.HttpBackend{
		Name:     "svc_back",
		Weight:   70,
		UseHTTP2: false,
		Service: &v1alpha1.ServiceBackend{
			Name: "service1",
			Port: v1alpha1.ServiceBackendPort{
				Number: 10001,
			},
		},
	}

	sessionAffinityALBBackend := apploadbalancer.HttpBackend{
		Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
		BackendWeight: &wrapperspb.Int64Value{
			Value: 70,
		},
		Port: 30081,
		BackendType: &apploadbalancer.HttpBackend_TargetGroups{
			TargetGroups: targetGroupsBackend,
		},
		Healthchecks: defaultHealthChecks,
		UseHttp2:     false,
	}

	testData := []struct {
		desc         string
		backendGroup *v1alpha1.HttpBackendGroup
		exp          *apploadbalancer.BackendGroup
		wantErr      bool
	}{
		{
			desc: "OK",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					Backends: []*v1alpha1.HttpBackend{
						{
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
							Name:   "s3_back",
							Weight: 30,
							StorageBucket: &v1alpha1.StorageBucketBackend{
								Name: "test-bucket",
							},
						},
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						Backends: []*apploadbalancer.HttpBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.HttpBackend_TargetGroups{
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
								UseHttp2: false,
							},
							{
								Name: "backend-efa6df77815c6618e5029dfb0e026cae569a1616-0-0",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 30,
								},
								BackendType: &apploadbalancer.HttpBackend_StorageBucket{
									StorageBucket: &apploadbalancer.StorageBucketBackend{
										Bucket: "test-bucket",
									},
								},
								UseHttp2: false,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "duplicated service backend",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					Backends: []*v1alpha1.HttpBackend{
						{
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						Backends: []*apploadbalancer.HttpBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.HttpBackend_TargetGroups{
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
								UseHttp2: false,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "different service ports for the same node port",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					Backends: []*v1alpha1.HttpBackend{
						{
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						Backends: []*apploadbalancer.HttpBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10000-30080",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30080,
								BackendType: &apploadbalancer.HttpBackend_TargetGroups{
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
								UseHttp2: false,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "duplicated bucket backend",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					Backends: []*v1alpha1.HttpBackend{
						{
							Name:   "s3_back",
							Weight: 30,
							StorageBucket: &v1alpha1.StorageBucketBackend{
								Name: "test-bucket",
							},
						},
						{
							Name:   "s3_back",
							Weight: 30,
							StorageBucket: &v1alpha1.StorageBucketBackend{
								Name: "test-bucket",
							},
						},
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						Backends: []*apploadbalancer.HttpBackend{
							{
								Name: "backend-efa6df77815c6618e5029dfb0e026cae569a1616-0-0",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 30,
								},
								BackendType: &apploadbalancer.HttpBackend_StorageBucket{
									StorageBucket: &apploadbalancer.StorageBucketBackend{
										Bucket: "test-bucket",
									},
								},
								UseHttp2: false,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "service not found",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					Backends: []*v1alpha1.HttpBackend{
						{
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					Backends: []*v1alpha1.HttpBackend{
						{
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					Backends: []*v1alpha1.HttpBackend{
						{
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						Backends: []*apploadbalancer.HttpBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10000-30080",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30080,
								BackendType: &apploadbalancer.HttpBackend_TargetGroups{
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
								UseHttp2: false,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "HeaderSessionAffinity",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					SessionAffinity: &v1alpha1.SessionAffinity{
						Header: &v1alpha1.SessionAffinityHeader{
							HeaderName: "foo",
						},
					},
					Backends: []*v1alpha1.HttpBackend{
						&sessionAffinityBackend,
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						SessionAffinity: &apploadbalancer.HttpBackendGroup_Header{
							Header: &apploadbalancer.HeaderSessionAffinity{
								HeaderName: "foo",
							},
						},
						Backends: []*apploadbalancer.HttpBackend{
							&sessionAffinityALBBackend,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "CookieSessionAffinity",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					SessionAffinity: &v1alpha1.SessionAffinity{
						Cookie: &v1alpha1.SessionAffinityCookie{
							Name: "foo",
							TTL:  &metav1.Duration{Duration: duration},
						},
					},
					Backends: []*v1alpha1.HttpBackend{
						&sessionAffinityBackend,
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						SessionAffinity: &apploadbalancer.HttpBackendGroup_Cookie{
							Cookie: &apploadbalancer.CookieSessionAffinity{
								Name: "foo",
								Ttl:  durationpb.New(duration),
							},
						},
						Backends: []*apploadbalancer.HttpBackend{
							&sessionAffinityALBBackend,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "ConnectionSessionAffinity",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					SessionAffinity: &v1alpha1.SessionAffinity{
						Connection: &v1alpha1.SessionAffinityConnection{
							SourceIP: true,
						},
					},
					Backends: []*v1alpha1.HttpBackend{
						&sessionAffinityBackend,
					},
				},
			},
			exp: &apploadbalancer.BackendGroup{
				Name:        "bg-cr-60cf398752e89f2e5e7a130f0fdf7e203fe17410",
				Description: "backend group for CR test-ns/test-bg",
				FolderId:    "my-folder",
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						SessionAffinity: &apploadbalancer.HttpBackendGroup_Connection{
							Connection: &apploadbalancer.ConnectionSessionAffinity{
								SourceIp: true,
							},
						},
						Backends: []*apploadbalancer.HttpBackend{
							&sessionAffinityALBBackend,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "BalancingMode",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					SessionAffinity: &v1alpha1.SessionAffinity{
						Connection: &v1alpha1.SessionAffinityConnection{
							SourceIP: true,
						},
					},
					Backends: []*v1alpha1.HttpBackend{
						{
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						SessionAffinity: &apploadbalancer.HttpBackendGroup_Connection{
							Connection: &apploadbalancer.ConnectionSessionAffinity{
								SourceIp: true,
							},
						},
						Backends: []*apploadbalancer.HttpBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.HttpBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Healthchecks: defaultHealthChecks,
								UseHttp2:     false,
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
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					Backends: []*v1alpha1.HttpBackend{
						{
							HealthChecks: []*v1alpha1.HealthCheck{
								{
									HTTP: &v1alpha1.HTTPHealthCheck{
										Path: "/health",
									},
									Port:               ptr.To[int64](30080),
									UnhealthyThreshold: 5,
									HealthyThreshold:   5,
									Interval:           &metav1.Duration{Duration: time.Second * 5},
									Timeout:            &metav1.Duration{Duration: time.Second * 2},
								},
							},
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						Backends: []*apploadbalancer.HttpBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.HttpBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Healthchecks: []*apploadbalancer.HealthCheck{
									{
										Timeout:            &durationpb.Duration{Seconds: 2},
										Interval:           &durationpb.Duration{Seconds: 5},
										HealthcheckPort:    30080,
										HealthyThreshold:   5,
										UnhealthyThreshold: 5,
										Healthcheck: &apploadbalancer.HealthCheck_Http{
											Http: &apploadbalancer.HealthCheck_HttpHealthCheck{
												Path: "/health",
											},
										},
										TransportSettings: &apploadbalancer.HealthCheck_Plaintext{
											Plaintext: &apploadbalancer.PlaintextTransportSettings{},
										},
									},
								},
								UseHttp2: false,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "HealthChecksNoPortSpecified",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					Backends: []*v1alpha1.HttpBackend{
						{
							HealthChecks: []*v1alpha1.HealthCheck{
								{
									HTTP: &v1alpha1.HTTPHealthCheck{
										Path: "/health",
									},
									UnhealthyThreshold: 5,
									HealthyThreshold:   5,
									Interval:           &metav1.Duration{Duration: time.Second * 5},
									Timeout:            &metav1.Duration{Duration: time.Second * 2},
								},
							},
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						Backends: []*apploadbalancer.HttpBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.HttpBackend_TargetGroups{
									TargetGroups: targetGroupsBackend,
								},
								Healthchecks: []*apploadbalancer.HealthCheck{
									{
										Timeout:            &durationpb.Duration{Seconds: 2},
										Interval:           &durationpb.Duration{Seconds: 5},
										HealthcheckPort:    30081,
										HealthyThreshold:   5,
										UnhealthyThreshold: 5,
										Healthcheck: &apploadbalancer.HealthCheck_Http{
											Http: &apploadbalancer.HealthCheck_HttpHealthCheck{
												Path: "/health",
											},
										},
										TransportSettings: &apploadbalancer.HealthCheck_Plaintext{
											Plaintext: &apploadbalancer.PlaintextTransportSettings{},
										},
									},
								},
								UseHttp2: false,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			desc: "HealthChecksTLS",
			backendGroup: &v1alpha1.HttpBackendGroup{
				TypeMeta:   metav1.TypeMeta{Kind: "HttpBackendGroup", APIVersion: "alb.yc.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-bg"},
				Spec: v1alpha1.HttpBackendGroupSpec{
					Backends: []*v1alpha1.HttpBackend{
						{
							HealthChecks: []*v1alpha1.HealthCheck{
								{
									HTTP: &v1alpha1.HTTPHealthCheck{
										Path: "/health",
									},
									Port:               ptr.To[int64](30080),
									UnhealthyThreshold: 5,
									HealthyThreshold:   5,
									Interval:           &metav1.Duration{Duration: time.Second * 5},
									Timeout:            &metav1.Duration{Duration: time.Second * 2},
								},
							},
							Name:     "svc_back",
							Weight:   70,
							UseHTTP2: false,
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
				Backend: &apploadbalancer.BackendGroup_Http{
					Http: &apploadbalancer.HttpBackendGroup{
						Backends: []*apploadbalancer.HttpBackend{
							{
								Name: "backend-2e6ab7c1338beb2b7cfd07166e44b68e20773af1-10001-30081",
								BackendWeight: &wrapperspb.Int64Value{
									Value: 70,
								},
								Port: 30081,
								BackendType: &apploadbalancer.HttpBackend_TargetGroups{
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
										Healthcheck: &apploadbalancer.HealthCheck_Http{
											Http: &apploadbalancer.HealthCheck_HttpHealthCheck{
												Path: "/health",
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
								UseHttp2: false,
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
			f := NewFactory("my-folder", "", &metadata.Names{ClusterID: "my-cluster"}, &metadata.Labels{ClusterID: "my-cluster"}, cli, tgRepo)

			b := f.BackendGroupForCRDBuilder()
			res, err := b.BuildBgFromCR(tc.backendGroup)
			require.True(t, (err != nil) == tc.wantErr)
			if tc.wantErr {
				return
			}
			assert.Condition(t, func() bool { return proto.Equal(tc.exp, res) }, "backend groups mismatch\nexp %v\ngot %v", tc.exp, res)
		})
	}
}
