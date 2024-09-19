package builders

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/types/known/durationpb"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
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

func SetupDefaultHealthChecks(enable bool) {
	if enable {
		defaultHealthChecks = []*apploadbalancer.HealthCheck{healthCheckTemplate()}
	} else {
		defaultHealthChecks = []*apploadbalancer.HealthCheck{}
	}
}

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

type BackendGroupForSvcBuilder struct {
	FolderID string
	Names    *metadata.Names
}

func (b *BackendGroupForSvcBuilder) BuildForSvc(svc *core.Service, ings []networking.Ingress, tgID string) (*apploadbalancer.BackendGroup, error) {
	if svc.Spec.Type != core.ServiceTypeNodePort {
		return nil, fmt.Errorf("type of service %s/%s used by path is not NodePort", svc.Name, svc.Namespace)
	}

	nodePorts, err := collectPortsForService(svc, ings)
	if err != nil {
		return nil, fmt.Errorf("failed to collect ports for service: %w", err)
	}

	opts, err := b.backendOpts(svc, ings)
	if err != nil {
		return nil, fmt.Errorf("failed to build backend opts: %w", err)
	}

	return b.buildForSvc(svc, nodePorts, tgID, opts)
}

func (b *BackendGroupForSvcBuilder) buildForSvc(svc *core.Service, nodePorts []core.ServicePort, tgID string, opts BackendResolveOpts) (*apploadbalancer.BackendGroup, error) {
	balancingConfig, err := parseBalancingConfigFromString(opts.BalancingMode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse balancing config: %w", err)
	}

	var tls *apploadbalancer.BackendTls
	if opts.Secure {
		tls = &apploadbalancer.BackendTls{}
	}

	var backend apploadbalancer.BackendGroup_Backend
	if opts.BackendType == GRPC {
		backends, err := b.buildGrpcBackends(
			svc, tgID, nodePorts, balancingConfig,
			tls, opts.healthChecks,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build grpc backends: %w", err)
		}

		sessionAffinity, err := parseGRPCSessionAffinityFromOpts(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to parse grpc session affinity: %w", err)
		}

		backend = &apploadbalancer.BackendGroup_Grpc{Grpc: &apploadbalancer.GrpcBackendGroup{Backends: backends, SessionAffinity: sessionAffinity}}
	} else {
		backends, err := b.buildHTTPBackends(
			svc, tgID, nodePorts, balancingConfig,
			tls, opts.BackendType == HTTP2, opts.healthChecks,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build http backends: %w", err)
		}

		sessionAffinity, err := parseHTTPSessionAffinityFromOpts(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to parse http session affinity: %w", err)
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

func (b *BackendGroupForSvcBuilder) backendOpts(svc *core.Service, ings []networking.Ingress) (BackendResolveOpts, error) {
	annotations := svc.GetAnnotations()
	r := BackendOptsResolver{}

	var err error
	protocol, ok := annotations[k8s.Protocol]
	if !ok {
		protocol, err = parseSvcAnnotationFromIngs(ings, k8s.Protocol)
		if err != nil {
			return BackendResolveOpts{}, fmt.Errorf("failed to parse protocol: %w", err)
		}
	}

	balancingMode, ok := annotations[k8s.BalancingMode]
	if !ok {
		balancingMode, err = parseSvcAnnotationFromIngs(ings, k8s.BalancingMode)
		if err != nil {
			return BackendResolveOpts{}, fmt.Errorf("failed to parse balancing mode: %w", err)
		}
	}

	transportSecurity, ok := annotations[k8s.TransportSecurity]
	if !ok {
		transportSecurity, err = parseSvcAnnotationFromIngs(ings, k8s.TransportSecurity)
		if err != nil {
			return BackendResolveOpts{}, fmt.Errorf("failed to parse transport security: %w", err)
		}
	}

	saHeader, ok := annotations[k8s.SessionAffinityHeader]
	if !ok {
		saHeader, err = parseSvcAnnotationFromIngs(ings, k8s.SessionAffinityHeader)
		if err != nil {
			return BackendResolveOpts{}, fmt.Errorf("failed to parse session affinity header: %w", err)
		}
	}

	saCookie, ok := annotations[k8s.SessionAffinityCookie]
	if !ok {
		saCookie, err = parseSvcAnnotationFromIngs(ings, k8s.SessionAffinityCookie)
		if err != nil {
			return BackendResolveOpts{}, fmt.Errorf("failed to parse session affinity cookie: %w", err)
		}
	}

	saConnection, ok := annotations[k8s.SessionAffinityConnection]
	if !ok {
		saConnection, err = parseSvcAnnotationFromIngs(ings, k8s.SessionAffinityConnection)
		if err != nil {
			return BackendResolveOpts{}, fmt.Errorf("failed to parse session affinity connection: %w", err)
		}
	}

	healthChecks, ok := annotations[k8s.HealthChecks]
	if !ok {
		healthChecks, err = parseSvcAnnotationFromIngs(ings, k8s.HealthChecks)
		if err != nil {
			return BackendResolveOpts{}, fmt.Errorf("failed to parse health checks: %w", err)
		}
	}

	opts, err := r.Resolve(
		protocol, balancingMode, transportSecurity,
		saHeader, saCookie, saConnection, healthChecks,
	)
	if err != nil {
		return BackendResolveOpts{}, fmt.Errorf("failed to resolve backend opts: %w", err)
	}

	return opts, nil
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

func (b *BackendGroupForSvcBuilder) buildHTTPBackends(
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

func (b *BackendGroupForSvcBuilder) buildGrpcBackends(
	svc *core.Service, id string,
	ports []core.ServicePort,
	balancingConfig *apploadbalancer.LoadBalancingConfig,
	tls *apploadbalancer.BackendTls,
	healthChecks []*apploadbalancer.HealthCheck,
) ([]*apploadbalancer.GrpcBackend, error) {
	backends := make([]*apploadbalancer.GrpcBackend, 0, len(ports))

	for _, p := range ports {
		backends = append(backends, &apploadbalancer.GrpcBackend{
			Name: b.Names.Backend("", svc.Namespace, svc.Name, p.Port, p.NodePort), // TODO(khodasevich): make better name
			Port: int64(p.NodePort),
			BackendType: &apploadbalancer.GrpcBackend_TargetGroups{
				TargetGroups: &apploadbalancer.TargetGroupsBackend{TargetGroupIds: []string{id}},
			},
			Healthchecks:        healthChecks,
			BackendWeight:       &wrappers.Int64Value{Value: 1},
			LoadBalancingConfig: balancingConfig,
			Tls:                 tls,
		})
	}

	return backends, nil
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
