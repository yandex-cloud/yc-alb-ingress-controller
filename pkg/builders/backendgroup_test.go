package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBackendGroupForSvcBuilder_BuildForSvc(t *testing.T) {
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
			b := BackendGroupForSvcBuilder{
				FolderID: "my-folder",
				Names:    &metadata.Names{ClusterID: "my-cluster"},
			}

			bg, err := b.buildForSvc(tc.svc, tc.nodePorts, "target-group-id", tc.opts)
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
