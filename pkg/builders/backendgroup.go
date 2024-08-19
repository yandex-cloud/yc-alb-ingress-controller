package builders

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/types/known/durationpb"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	ycerrors "github.com/yandex-cloud/alb-ingress/pkg/errors"
	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
)

const (
	hcPort = int64(10501)
)

type BackendType int

const (
	HTTP BackendType = iota
	HTTP2
	GRPC
)

const PathTypeRegex = "Regex"

func healthCheckTemplate() *apploadbalancer.HealthCheck {
	return &apploadbalancer.HealthCheck{
		Timeout:            &durationpb.Duration{Seconds: 2},
		Interval:           &durationpb.Duration{Seconds: 5},
		HealthyThreshold:   1,
		UnhealthyThreshold: 1,
		HealthcheckPort:    hcPort,
		Healthcheck: &apploadbalancer.HealthCheck_Http{
			Http: &apploadbalancer.HealthCheck_HttpHealthCheck{
				Path: "/healthz",
			},
		},
		TransportSettings: &apploadbalancer.HealthCheck_Plaintext{
			Plaintext: &apploadbalancer.PlaintextTransportSettings{},
		},
	}
}

var defaultHealthChecks = []*apploadbalancer.HealthCheck{healthCheckTemplate()}

type HostAndPath struct {
	Host, Path, PathType string
}

func HTTPIngressPathToHostAndPath(host string, path networking.HTTPIngressPath, useRegex bool) (HostAndPath, error) {
	if useRegex {
		if path.PathType != nil && *path.PathType == networking.PathTypePrefix {
			return HostAndPath{}, fmt.Errorf("path type prefix is not supported with regex annotation, use path type exact")
		}

		return HostAndPath{Host: host, Path: path.Path, PathType: PathTypeRegex}, nil
	}

	if path.PathType == nil {
		return HostAndPath{Host: host, Path: path.Path, PathType: string(networking.PathTypePrefix)}, nil
	}
	return HostAndPath{Host: host, Path: path.Path, PathType: string(*path.PathType)}, nil
}

// exposedNodePort - exposed NodePort and route which will be served by a service which opened it
type exposedNodePort struct {
	port int64
}

type BackendResolveOpts struct {
	BackendType BackendType
	Secure      bool
	UseRegex    bool

	BalancingMode string

	healthChecks []*apploadbalancer.HealthCheck
	affinityOpts SessionAffinityOpts
}

type TargetGroupFinder interface {
	FindTargetGroup(context.Context, string) (*apploadbalancer.TargetGroup, error)
}

type BackendGroups struct {
	BackendGroups          []*apploadbalancer.BackendGroup
	BackendGroupByHostPath map[HostAndPath]*apploadbalancer.BackendGroup
	BackendGroupByName     map[string]*apploadbalancer.BackendGroup
	CRBGNameOrIDMap        map[HostAndPath]string
}

type BackendGroupBuilder struct {
	FolderID string
	Names    *metadata.Names
}

func (b *BackendGroupBuilder) Build(svc *core.Service, ings []networking.Ingress, tgID string) (*apploadbalancer.BackendGroup, error) {
	if svc.Spec.Type != core.ServiceTypeNodePort {
		return nil, fmt.Errorf("type of service %s/%s used by path is not NodePort", svc.Name, svc.Namespace)
	}

	nodePorts, err := collectPortsForService(svc, ings)
	if err != nil {
		return nil, err
	}

	opts, err := b.backendOpts(svc, ings)
	if err != nil {
		return nil, err
	}

	return b.build(svc, nodePorts, tgID, opts)
}

func (b *BackendGroupBuilder) build(svc *core.Service, nodePorts []core.ServicePort, tgID string, opts BackendResolveOpts) (*apploadbalancer.BackendGroup, error) {
	balancingConfig, err := parseBalancingConfigFromString(opts.BalancingMode)
	if err != nil {
		return nil, err
	}

	var tls *apploadbalancer.BackendTls
	if opts.Secure {
		tls = &apploadbalancer.BackendTls{}
	}

	var backend apploadbalancer.BackendGroup_Backend
	if opts.BackendType == GRPC {
		backends, err := b.buildGrpcBackends(svc, tgID, nodePorts, balancingConfig, tls)
		if err != nil {
			return nil, err
		}

		sessionAffinity, err := parseGRPCSessionAffinityFromOpts(opts)
		if err != nil {
			return nil, err
		}

		backend = &apploadbalancer.BackendGroup_Grpc{Grpc: &apploadbalancer.GrpcBackendGroup{Backends: backends, SessionAffinity: sessionAffinity}}
	} else {
		backends, err := b.buildHTTPBackends(
			svc, tgID, nodePorts, balancingConfig,
			tls, opts.BackendType == HTTP2, opts.healthChecks,
		)
		if err != nil {
			return nil, err
		}

		sessionAffinity, err := parseHTTPSessionAffinityFromOpts(opts)
		if err != nil {
			return nil, err
		}

		backend = &apploadbalancer.BackendGroup_Http{Http: &apploadbalancer.HttpBackendGroup{Backends: backends, SessionAffinity: sessionAffinity}}
	}

	return &apploadbalancer.BackendGroup{
		Name:        b.Names.NewBackendGroup(k8s.NamespacedNameOf(svc)),
		FolderId:    b.FolderID,
		Description: fmt.Sprintf("backend group for k8s service %s/%s", svc.Namespace, svc.Name),
		Backend:     backend,
	}, nil
}

func (b *BackendGroupBuilder) backendOpts(svc *core.Service, ings []networking.Ingress) (BackendResolveOpts, error) {
	annotations := svc.GetAnnotations()
	r := BackendOptsResolver{}

	var err error
	protocol, ok := annotations[k8s.Protocol]
	if !ok {
		protocol, err = parseSvcAnnotationFromIngs(ings, k8s.Protocol)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	}

	balancingMode, ok := annotations[k8s.BalancingMode]
	if !ok {
		balancingMode, err = parseSvcAnnotationFromIngs(ings, k8s.BalancingMode)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	}

	transportSecurity, ok := annotations[k8s.TransportSecurity]
	if !ok {
		transportSecurity, err = parseSvcAnnotationFromIngs(ings, k8s.TransportSecurity)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	}

	saHeader, ok := annotations[k8s.SessionAffinityHeader]
	if !ok {
		saHeader, err = parseSvcAnnotationFromIngs(ings, k8s.SessionAffinityHeader)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	}

	saCookie, ok := annotations[k8s.SessionAffinityCookie]
	if !ok {
		saCookie, err = parseSvcAnnotationFromIngs(ings, k8s.SessionAffinityCookie)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	}

	saConnection, ok := annotations[k8s.SessionAffinityConnection]
	if !ok {
		saConnection, err = parseSvcAnnotationFromIngs(ings, k8s.SessionAffinityConnection)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	}

	healthChecks, ok := annotations[k8s.HealthChecks]
	if !ok {
		healthChecks, err = parseSvcAnnotationFromIngs(ings, k8s.HealthChecks)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	}

	return r.Resolve(
		protocol, balancingMode, transportSecurity,
		saHeader, saCookie, saConnection, healthChecks,
	)
}

func parseSvcAnnotationFromIngs(ings []networking.Ingress, annotation string) (string, error) {
	var result string
	for _, ing := range ings {
		val, ok := ing.GetAnnotations()[annotation]
		if !ok {
			continue
		}

		if result != "" && result != val {
			return "", fmt.Errorf("different values passed for one annotation. Annotation: %s, values: %s, %s", annotation, val, result)
		}

		result = val
	}
	return result, nil
}

func collectPortsForService(svc *core.Service, ings []networking.Ingress) ([]core.ServicePort, error) {
	nodePorts := make(map[core.ServicePort]struct{})
	collectPortsForBackend := func(backend networking.IngressBackend) error {
		if backend.Service == nil || backend.Service.Name != svc.Name {
			return nil
		}

		ingressBackendPort := backend.Service.Port
		svcBackendPorts := nodePortsForServicePort(ingressBackendPort.Name, ingressBackendPort.Number, svc.Spec.Ports)
		if len(svcBackendPorts) == 0 {
			return fmt.Errorf("service %s/%s doesn't expose its port %v",
				svc.Namespace, svc.Name, ingressBackendPort)
		}

		for _, p := range svcBackendPorts {
			nodePorts[p] = struct{}{}
		}

		return nil
	}

	for _, ing := range ings {
		if ing.Spec.DefaultBackend != nil {
			err := collectPortsForBackend(*ing.Spec.DefaultBackend)
			if err != nil {
				return nil, fmt.Errorf("error %w on ingresses %s/%s default backend", err, ing.Namespace, ing.Name)
			}
		}

		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				err := collectPortsForBackend(path.Backend)
				if err != nil {
					return nil, fmt.Errorf("error %w on ingress: %s/%s, path: %s", err, path.Path, ing.Namespace, ing.Name)
				}
			}
		}
	}

	result := make([]core.ServicePort, 0, len(nodePorts))
	for p := range nodePorts {
		result = append(result, p)
	}

	return result, nil
}

func (b *BackendGroupBuilder) buildHTTPBackends(
	svc *core.Service, tgID string,
	ports []core.ServicePort,
	balancingConfig *apploadbalancer.LoadBalancingConfig,
	tls *apploadbalancer.BackendTls, useHTTP2 bool,
	healthChecks []*apploadbalancer.HealthCheck,
) ([]*apploadbalancer.HttpBackend, error) {
	backends := make([]*apploadbalancer.HttpBackend, 0, len(ports))
	for _, p := range ports {
		backends = append(backends, &apploadbalancer.HttpBackend{
			Name: b.Names.Backend("", svc.Namespace, svc.Name, p.Port, p.NodePort), // TODO(khodasevich): make better name
			Port: int64(p.NodePort),
			BackendType: &apploadbalancer.HttpBackend_TargetGroups{
				TargetGroups: &apploadbalancer.TargetGroupsBackend{TargetGroupIds: []string{tgID}},
			},
			Healthchecks:        healthChecks,
			BackendWeight:       &wrappers.Int64Value{Value: 1},
			LoadBalancingConfig: balancingConfig,
			Tls:                 tls,
			UseHttp2:            useHTTP2,
		})
	}

	return backends, nil
}

func (b *BackendGroupBuilder) buildGrpcBackends(
	svc *core.Service, id string,
	ports []core.ServicePort,
	balancingConfig *apploadbalancer.LoadBalancingConfig,
	tls *apploadbalancer.BackendTls,
) ([]*apploadbalancer.GrpcBackend, error) {
	backends := make([]*apploadbalancer.GrpcBackend, 0, len(ports))

	for _, p := range ports {
		backends = append(backends, &apploadbalancer.GrpcBackend{
			Name: b.Names.Backend("", svc.Namespace, svc.Name, p.Port, p.NodePort), // TODO(khodasevich): make better name
			Port: int64(p.NodePort),
			BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
				TargetGroups: &apploadbalancer.TargetGroupsBackend{TargetGroupIds: []string{id}},
			},
			Healthchecks:        defaultHealthChecks,
			BackendWeight:       &wrappers.Int64Value{Value: 1},
			LoadBalancingConfig: balancingConfig,
			Tls:                 tls,
		})
	}

	return backends, nil
}

type BackendGroupForCRDBuilder struct {
	cli        client.Client
	names      *metadata.Names
	labels     *metadata.Labels
	folderID   string
	seenSvc    map[exposedNodePort]struct{}
	seenBucket map[string]struct{}
	tag        string // TODO: delete

	targetGroupFinder TargetGroupFinder
}

func (b *BackendGroupForCRDBuilder) BuildBgFromCR(bgCR *v1alpha1.HttpBackendGroup) (*apploadbalancer.BackendGroup, error) {
	var backends []*apploadbalancer.HttpBackend
	for _, bcrd := range bgCR.Spec.Backends {
		if bcrd.Service != nil {
			bgs, err := b.buildBackendForService(bgCR.Namespace, bcrd)
			if err != nil {
				return nil, err
			}
			backends = append(backends, bgs...)
			continue
		}

		if bcrd.StorageBucket != nil {
			bg, err := b.buildBackendForBucket(bgCR.Namespace, bcrd)
			if err != nil {
				return nil, err
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
			SessionAffinity: parseSessionAffinity(bgCR.Spec.SessionAffinity),
		},
	}

	return &apploadbalancer.BackendGroup{
		Name:        b.names.BackendGroupForCR(bgCR.Namespace, bgCR.Name),
		Description: fmt.Sprintf("backend group for CR %s/%s", bgCR.Namespace, bgCR.Name),
		FolderId:    b.folderID,
		Labels:      nil,
		Backend:     &backend,
		CreatedAt:   nil,
	}, nil
}

func parseSessionAffinity(sa *v1alpha1.SessionAffinity) apploadbalancer.HttpBackendGroup_SessionAffinity {
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

// TODO: duplicated code with BackendGroupBuilder.AddBackend
func (b *BackendGroupForCRDBuilder) buildBackendForService(ns string, crdBackend *v1alpha1.HttpBackend) ([]*apploadbalancer.HttpBackend, error) {
	var svc core.Service
	ctx := context.Background() // TODO: obtain as param
	err := b.cli.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      crdBackend.Service.Name,
	}, &svc)
	if err != nil {
		return nil, err
	}
	if svc.Spec.Type != core.ServiceTypeNodePort {
		return nil, fmt.Errorf("type of service %s/%s used by CR HttpBackend %s is not NodePort",
			svc.Namespace, svc.Name, crdBackend.Service.Name)
	}

	tgName := b.names.TargetGroup(k8s.NamespacedNameOf(&svc))
	tg, err := b.targetGroupFinder.FindTargetGroup(ctx, tgName)
	if err != nil {
		return nil, err
	}
	if tg == nil {
		return nil, ycerrors.YCResourceNotReadyError{ResourceType: "target group", Name: tgName}
	}

	ingressBackendPort := crdBackend.Service.Port
	svcBackendPorts := nodePortsForServicePort(ingressBackendPort.Name, ingressBackendPort.Number, svc.Spec.Ports)
	if len(svcBackendPorts) == 0 {
		return nil, fmt.Errorf("service %s/%s doesn't expose its port %v",
			svc.Namespace, svc.Name, ingressBackendPort)
	}

	balancingConfig, err := parseBalancingConfigFromCRDConfig(crdBackend.LoadBalancingConfig)
	if err != nil {
		return nil, err
	}

	var ret []*apploadbalancer.HttpBackend
	for _, port := range svcBackendPorts {
		nodePort := int64(port.NodePort)
		if _, ok := b.seenSvc[exposedNodePort{port: nodePort}]; ok {
			// backend for this service and NodePort has already been added to this backend group
			continue
		}

		backend := &apploadbalancer.HttpBackend{
			Name:          b.names.Backend(b.tag, svc.Namespace, svc.Name, port.Port, port.NodePort),
			BackendWeight: &wrappers.Int64Value{Value: crdBackend.Weight},
			Port:          nodePort,
			BackendType: &apploadbalancer.HttpBackend_TargetGroups{
				TargetGroups: &apploadbalancer.TargetGroupsBackend{
					TargetGroupIds: []string{
						tg.Id,
					},
				},
			},
			Healthchecks:        buildHealthChecks(crdBackend, svcBackendPorts),
			UseHttp2:            crdBackend.UseHTTP2,
			LoadBalancingConfig: balancingConfig,
		}
		if crdBackend.TLS != nil {
			backend.Tls = &apploadbalancer.BackendTls{
				Sni: crdBackend.TLS.Sni,
			}
			if len(crdBackend.TLS.TrustedCa) > 0 {
				backend.Tls.ValidationContext = &apploadbalancer.ValidationContext{
					TrustedCa: &apploadbalancer.ValidationContext_TrustedCaBytes{
						TrustedCaBytes: crdBackend.TLS.TrustedCa,
					},
				}
			}
		}

		ret = append(ret, backend)
		b.seenSvc[exposedNodePort{port: nodePort}] = struct{}{}
	}
	return ret, nil
}

func convertDuration(s *metav1.Duration) *durationpb.Duration {
	if s == nil {
		return nil
	}
	return durationpb.New(s.Duration)
}

func buildHealthChecks(backend *v1alpha1.HttpBackend, svcPorts []core.ServicePort) []*apploadbalancer.HealthCheck {
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

func parseBalancingConfigFromString(mode string) (*apploadbalancer.LoadBalancingConfig, error) {
	if mode == "" {
		return nil, nil
	}
	return parseBalancingConfig(mode)
}

func parseBalancingConfigFromCRDConfig(config *v1alpha1.LoadBalancingConfig) (*apploadbalancer.LoadBalancingConfig, error) {
	if config == nil {
		return nil, nil
	}
	return parseBalancingConfig(config.BalancerMode)
}

func parseBalancingConfig(mode string) (*apploadbalancer.LoadBalancingConfig, error) {
	balancingMode, ok := apploadbalancer.LoadBalancingMode_value[mode]
	if !ok {
		return nil, fmt.Errorf("unknown balancing mode: %s", mode)
	}

	return &apploadbalancer.LoadBalancingConfig{
		Mode: apploadbalancer.LoadBalancingMode(balancingMode),
	}, nil
}

func (b *BackendGroupForCRDBuilder) buildBackendForBucket(namespace string, crdBackend *v1alpha1.HttpBackend) (*apploadbalancer.HttpBackend, error) {
	if _, ok := b.seenBucket[crdBackend.StorageBucket.Name]; ok {
		return nil, nil
	}
	b.seenBucket[crdBackend.StorageBucket.Name] = struct{}{}

	balancingConfig, err := parseBalancingConfigFromCRDConfig(crdBackend.LoadBalancingConfig)
	if err != nil {
		return nil, err
	}

	return &apploadbalancer.HttpBackend{
		Name:          b.names.Backend(b.tag, namespace, crdBackend.StorageBucket.Name, 0, 0), // TODO: fix naming
		BackendWeight: &wrappers.Int64Value{Value: crdBackend.Weight},
		BackendType: &apploadbalancer.HttpBackend_StorageBucket{
			StorageBucket: &apploadbalancer.StorageBucketBackend{Bucket: crdBackend.StorageBucket.Name},
		},
		LoadBalancingConfig: balancingConfig,
	}, nil
}

func parseGRPCSessionAffinityFromOpts(o BackendResolveOpts) (apploadbalancer.GrpcBackendGroup_SessionAffinity, error) {
	switch {
	case o.affinityOpts.header != nil:
		return &apploadbalancer.GrpcBackendGroup_Header{
			Header: o.affinityOpts.header,
		}, nil
	case o.affinityOpts.connection != nil:
		return &apploadbalancer.GrpcBackendGroup_Connection{
			Connection: o.affinityOpts.connection,
		}, nil
	case o.affinityOpts.cookie != nil:
		return &apploadbalancer.GrpcBackendGroup_Cookie{
			Cookie: o.affinityOpts.cookie,
		}, nil
	default:
		return nil, nil
	}
}

func parseHTTPSessionAffinityFromOpts(o BackendResolveOpts) (apploadbalancer.HttpBackendGroup_SessionAffinity, error) {
	switch {
	case o.affinityOpts.header != nil:
		return &apploadbalancer.HttpBackendGroup_Header{
			Header: o.affinityOpts.header,
		}, nil
	case o.affinityOpts.connection != nil:
		return &apploadbalancer.HttpBackendGroup_Connection{
			Connection: o.affinityOpts.connection,
		}, nil
	case o.affinityOpts.cookie != nil:
		return &apploadbalancer.HttpBackendGroup_Cookie{
			Cookie: o.affinityOpts.cookie,
		}, nil
	default:
		return nil, nil
	}
}

func nodePortsForServicePort(portName string, portNumber int32, servicePorts []core.ServicePort) []core.ServicePort {
	var nodePorts []core.ServicePort
	if len(portName) > 0 {
		for _, svcPort := range servicePorts {
			if svcPort.Name == portName {
				nodePorts = []core.ServicePort{svcPort}
				break
			}
		}
	} else {
		for _, svcPort := range servicePorts {
			if svcPort.Port == portNumber {
				nodePorts = append(nodePorts, svcPort)
			}
		}
	}
	return nodePorts
}
