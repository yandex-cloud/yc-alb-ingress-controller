package builders

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/yc-alb-ingress-controller/api/v1alpha1"
	ycerrors "github.com/yandex-cloud/yc-alb-ingress-controller/pkg/errors"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/k8s"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/metadata"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HttpBackendGroupRepository interface { //nolint:revive
	FindTargetGroup(context.Context, string) (*apploadbalancer.TargetGroup, error)
}

type HttpBackendGroupForCrdBuilder struct { //nolint:revive
	FolderID string
	Names    *metadata.Names
	Cli      client.Client
	Repo     HttpBackendGroupRepository
}

func (b *HttpBackendGroupForCrdBuilder) BuildForCrd(
	ctx context.Context, bgCR *v1alpha1.HttpBackendGroup,
) (*apploadbalancer.BackendGroup, error) {
	var backends []*apploadbalancer.HttpBackend

	seenSvc := make(map[exposedNodePort]struct{})
	seenBucket := make(map[string]struct{})

	for _, bcrd := range bgCR.Spec.Backends {
		if bcrd.Service != nil {
			bgs, err := b.buildHttpBackendsForService(ctx, bgCR.Namespace, seenSvc, bcrd)
			if err != nil {
				return nil, fmt.Errorf("failed to build backends for service %s/%s: %w", bgCR.Namespace, bcrd.Service.Name, err)
			}
			backends = append(backends, bgs...)
			continue
		}

		if bcrd.StorageBucket != nil {
			bg, err := b.buildHttpBackendForBucket(ctx, bgCR.Namespace, seenBucket, bcrd)
			if err != nil {
				return nil, fmt.Errorf("failed to build backend for bucket %s/%s: %w", bgCR.Namespace, bcrd.StorageBucket.Name, err)
			}

			if bg != nil {
				backends = append(backends, bg)
			}
			continue
		}
	}

	backend := apploadbalancer.BackendGroup_Http{
		Http: &apploadbalancer.HttpBackendGroup{
			Backends:        backends,
			SessionAffinity: parseHttpBGSessionAffinity(bgCR.Spec.SessionAffinity),
		},
	}

	return &apploadbalancer.BackendGroup{
		Name:        b.Names.BackendGroupForCR(bgCR.Namespace, bgCR.Name),
		Description: fmt.Sprintf("backend group for CR %s/%s", bgCR.Namespace, bgCR.Name),
		FolderId:    b.FolderID,
		Labels:      nil,
		Backend:     &backend,
		CreatedAt:   nil,
	}, nil
}

func (b *HttpBackendGroupForCrdBuilder) buildHttpBackendsForService( //nolint:revive
	ctx context.Context, ns string, seenSvc map[exposedNodePort]struct{}, bgCrd *v1alpha1.HttpBackend,
) ([]*apploadbalancer.HttpBackend, error) {
	var svc core.Service
	err := b.Cli.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      bgCrd.Service.Name,
	}, &svc)
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s/%s: %w", ns, bgCrd.Service.Name, err)
	}
	if svc.Spec.Type != core.ServiceTypeNodePort {
		return nil, fmt.Errorf("type of service %s/%s used by CR HttpBackend %s is not NodePort",
			svc.Namespace, svc.Name, bgCrd.Service.Name)
	}

	tgName := b.Names.TargetGroup(k8s.NamespacedNameOf(&svc))
	tg, err := b.Repo.FindTargetGroup(ctx, tgName)
	if err != nil {
		return nil, fmt.Errorf("failed to find target group %s: %w", tgName, err)
	}
	if tg == nil {
		return nil, ycerrors.YCResourceNotReadyError{ResourceType: "target group", Name: tgName}
	}

	ingressBackendPort := bgCrd.Service.Port
	svcBackendPorts := nodePortsForServicePort(ingressBackendPort.Name, ingressBackendPort.Number, svc.Spec.Ports)
	if len(svcBackendPorts) == 0 {
		return nil, fmt.Errorf("service %s/%s doesn't expose its port %v",
			svc.Namespace, svc.Name, ingressBackendPort)
	}

	balancingConfig, err := parseBalancingConfigFromCRDConfig(bgCrd.LoadBalancingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse load balancing config: %w", err)
	}

	var ret []*apploadbalancer.HttpBackend
	for _, port := range svcBackendPorts {
		nodePort := int64(port.NodePort)
		if _, ok := seenSvc[exposedNodePort{port: nodePort}]; ok {
			// backend for this service and NodePort has already been added to this backend group
			continue
		}

		backend := &apploadbalancer.HttpBackend{
			Name:          b.Names.Backend("", svc.Namespace, svc.Name, port.Port, port.NodePort),
			BackendWeight: &wrappers.Int64Value{Value: bgCrd.Weight},
			Port:          nodePort,
			BackendType: &apploadbalancer.HttpBackend_TargetGroups{
				TargetGroups: &apploadbalancer.TargetGroupsBackend{
					TargetGroupIds: []string{
						tg.Id,
					},
				},
			},
			Healthchecks:        b.buildHttpHealthChecks(bgCrd, svcBackendPorts),
			UseHttp2:            bgCrd.UseHTTP2,
			LoadBalancingConfig: balancingConfig,
		}
		if bgCrd.TLS != nil {
			backend.Tls = &apploadbalancer.BackendTls{
				Sni: bgCrd.TLS.Sni,
			}
			if len(bgCrd.TLS.TrustedCa) > 0 {
				backend.Tls.ValidationContext = &apploadbalancer.ValidationContext{
					TrustedCa: &apploadbalancer.ValidationContext_TrustedCaBytes{
						TrustedCaBytes: bgCrd.TLS.TrustedCa,
					},
				}
			}
		}

		ret = append(ret, backend)
		seenSvc[exposedNodePort{port: nodePort}] = struct{}{}
	}
	return ret, nil
}

func (b *HttpBackendGroupForCrdBuilder) buildHttpBackendForBucket( //nolint:revive
	_ context.Context, ns string, seenBucket map[string]struct{}, bgCrd *v1alpha1.HttpBackend,
) (*apploadbalancer.HttpBackend, error) {
	if _, ok := seenBucket[bgCrd.StorageBucket.Name]; ok {
		return nil, nil
	}
	seenBucket[bgCrd.StorageBucket.Name] = struct{}{}

	balancingConfig, err := parseBalancingConfigFromCRDConfig(bgCrd.LoadBalancingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse load balancing config: %w", err)
	}

	return &apploadbalancer.HttpBackend{
		Name:          b.Names.Backend("", ns, bgCrd.StorageBucket.Name, 0, 0), // TODO: fix naming
		BackendWeight: &wrappers.Int64Value{Value: bgCrd.Weight},
		BackendType: &apploadbalancer.HttpBackend_StorageBucket{
			StorageBucket: &apploadbalancer.StorageBucketBackend{Bucket: bgCrd.StorageBucket.Name},
		},
		LoadBalancingConfig: balancingConfig,
	}, nil
}

func (b *HttpBackendGroupForCrdBuilder) buildHttpHealthChecks(backend *v1alpha1.HttpBackend, svcPorts []core.ServicePort) []*apploadbalancer.HealthCheck { //nolint:revive
	if len(backend.HealthChecks) == 0 {
		return defaultHealthChecks
	}

	res := make([]*apploadbalancer.HealthCheck, 0, len(backend.HealthChecks))
	for _, check := range backend.HealthChecks {
		var transportSettings apploadbalancer.HealthCheck_TransportSettings
		if backend.TLS != nil {
			transportSettings = &apploadbalancer.HealthCheck_Tls{
				Tls: &apploadbalancer.SecureTransportSettings{
					Sni: backend.TLS.Sni,
					ValidationContext: &apploadbalancer.ValidationContext{
						TrustedCa: &apploadbalancer.ValidationContext_TrustedCaBytes{
							TrustedCaBytes: backend.TLS.TrustedCa,
						},
					},
				},
			}
		} else {
			transportSettings = &apploadbalancer.HealthCheck_Plaintext{
				Plaintext: &apploadbalancer.PlaintextTransportSettings{},
			}
		}

		if check.Port != nil {
			res = append(res, &apploadbalancer.HealthCheck{
				Timeout:         convertDuration(check.Timeout),
				Interval:        convertDuration(check.Interval),
				HealthcheckPort: *check.Port,
				Healthcheck: &apploadbalancer.HealthCheck_Http{
					Http: &apploadbalancer.HealthCheck_HttpHealthCheck{
						Path: check.HTTP.Path,
					},
				},
				HealthyThreshold:   check.HealthyThreshold,
				UnhealthyThreshold: check.UnhealthyThreshold,
				TransportSettings:  transportSettings,
			})
		} else {
			for _, port := range svcPorts {
				res = append(res, &apploadbalancer.HealthCheck{
					Timeout:         convertDuration(check.Timeout),
					Interval:        convertDuration(check.Interval),
					HealthcheckPort: int64(port.NodePort),
					Healthcheck: &apploadbalancer.HealthCheck_Http{
						Http: &apploadbalancer.HealthCheck_HttpHealthCheck{
							Path: check.HTTP.Path,
						},
					},
					HealthyThreshold:   check.HealthyThreshold,
					UnhealthyThreshold: check.UnhealthyThreshold,
					TransportSettings:  transportSettings,
				})
			}
		}
	}

	return res
}

func parseHttpBGSessionAffinity(sa *v1alpha1.SessionAffinity) apploadbalancer.HttpBackendGroup_SessionAffinity { //nolint:revive
	if sa == nil {
		return nil
	}

	if sa.Connection != nil {
		return &apploadbalancer.HttpBackendGroup_Connection{
			Connection: &apploadbalancer.ConnectionSessionAffinity{
				SourceIp: sa.Connection.SourceIP,
			},
		}
	}

	if sa.Header != nil {
		return &apploadbalancer.HttpBackendGroup_Header{
			Header: &apploadbalancer.HeaderSessionAffinity{
				HeaderName: sa.Header.HeaderName,
			},
		}
	}

	if sa.Cookie != nil {
		cookie := &apploadbalancer.HttpBackendGroup_Cookie{
			Cookie: &apploadbalancer.CookieSessionAffinity{
				Name: sa.Cookie.Name,
			},
		}

		if sa.Cookie.TTL != nil {
			cookie.Cookie.Ttl = durationpb.New(sa.Cookie.TTL.Duration)
		}

		return cookie
	}

	return nil
}

func parseBalancingConfigFromStruct(lbc LoadBalancingConfig) (*apploadbalancer.LoadBalancingConfig, error) {
	if lbc.Mode == "" {
		return nil, nil
	}
	return parseBalancingConfig(lbc.Mode, lbc.PanicThreshold, lbc.LocalityAwareRouting)
}

func parseBalancingConfigFromCRDConfig(config *v1alpha1.LoadBalancingConfig) (*apploadbalancer.LoadBalancingConfig, error) {
	if config == nil {
		return nil, nil
	}
	return parseBalancingConfig(config.BalancerMode, config.PanicThreshold, config.LocalityAwareRouting)
}

func parseBalancingConfig(mode string, panicThreshold int64, localityAwareRouting int64) (*apploadbalancer.LoadBalancingConfig, error) {
	balancingMode, ok := apploadbalancer.LoadBalancingMode_value[mode]
	if !ok {
		return nil, fmt.Errorf("unknown balancing mode: %s", mode)
	}

	return &apploadbalancer.LoadBalancingConfig{
		Mode:                        apploadbalancer.LoadBalancingMode(balancingMode),
		PanicThreshold:              panicThreshold,
		LocalityAwareRoutingPercent: localityAwareRouting,
	}, nil
}

func convertDuration(s *metav1.Duration) *durationpb.Duration {
	if s == nil {
		return nil
	}
	return durationpb.New(s.Duration)
}
