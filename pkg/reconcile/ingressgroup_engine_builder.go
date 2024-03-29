package reconcile

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/protobuf/types/known/wrapperspb"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"

	"github.com/yandex-cloud/alb-ingress/pkg/builders"
	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	"github.com/yandex-cloud/alb-ingress/pkg/yc"
)

type DefaultEngineBuilder struct {
	factory   *builders.Factory
	resolvers *builders.Resolvers
	folderID  string

	names *metadata.Names

	certRepo yc.CertRepo
	bgFinder builders.BackendGroupFinder

	newIngressGroupEngine func(data *builders.Data) *IngressGroupEngine
}

func NewDefaultDataBuilder(
	factory *builders.Factory, resolvers *builders.Resolvers,
	newEngine func(data *builders.Data) *IngressGroupEngine, folderID string, names *metadata.Names, certRepo yc.CertRepo, bgFinder builders.BackendGroupFinder) *DefaultEngineBuilder {
	return &DefaultEngineBuilder{
		folderID:  folderID,
		factory:   factory,
		resolvers: resolvers,

		certRepo: certRepo,
		bgFinder: bgFinder,

		names:                 names,
		newIngressGroupEngine: newEngine,
	}
}

// TODO: включать/выключать трафик в зоне - ?
// TODO: httpHandler -> Http2Options, AllowHttp10 ?
func (d *DefaultEngineBuilder) Build(ctx context.Context, g *k8s.IngressGroup, settings *v1alpha1.IngressGroupSettings) (*IngressGroupEngine, error) {
	if len(g.Items) == 0 {
		return d.newIngressGroupEngine(nil), nil
	}
	networkID, locations, err := d.locations(g)
	if err != nil {
		return nil, err
	}
	addressParams := builders.AddressParams{DefaultSubnetID: locations[0].SubnetId}
	addresses, err := d.addresses(g, addressParams)
	if err != nil {
		return nil, err
	}
	securityGroupIDs := d.securityGroupIDs(g)

	opts := builders.Options{
		BalancerOptions: builders.BalancerOptions{
			NetworkID:        networkID,
			Locations:        locations,
			SecurityGroupIDs: securityGroupIDs,
		},
		ListenerOptions: builders.ListenerOptions{
			Addresses: addresses,
		},
	}

	b := builders.Data{}
	b.HTTPHosts, b.TLSHosts, err = d.buildVirtualHosts(g)
	if err != nil {
		return nil, err
	}
	b.Handler = d.buildHTTPHandler(g)
	b.SNIMatches, err = d.buildSNIMatches(ctx, g)
	if err != nil {
		return nil, err
	}
	b.LogOptions = d.buildLogOptions(settings)

	b.Balancer = d.buildBalancer(b.Handler, b.SNIMatches, b.LogOptions, g.Tag, opts)

	return d.newIngressGroupEngine(&b), nil
}

func (d *DefaultEngineBuilder) addresses(g *k8s.IngressGroup, p builders.AddressParams) ([]*apploadbalancer.Address, error) {
	resolver := d.resolvers.Addresses(p)
	for _, ing := range g.Items {
		annotations := ing.GetAnnotations()
		data := builders.AddressData{
			ExternalIPv4: annotations[k8s.ExternalIPv4Address],
			ExternalIPv6: annotations[k8s.ExternalIPv6Address],
			InternalIPv4: annotations[k8s.InternalIPv4Address],
			SubnetID:     annotations[k8s.InternalALBSubnet],
		}
		resolver.Resolve(data)
	}
	return resolver.Result()
}

func (d *DefaultEngineBuilder) locations(g *k8s.IngressGroup) (string, []*apploadbalancer.Location, error) {
	resolver := d.resolvers.Location()
	for _, ing := range g.Items {
		if err := resolver.Resolve(ing.GetAnnotations()[k8s.Subnets]); err != nil {
			return "", nil, err
		}
	}
	return resolver.Result()
}

func (d *DefaultEngineBuilder) securityGroupIDs(g *k8s.IngressGroup) []string {
	resolver := d.resolvers.SecurityGroups()
	for _, ing := range g.Items {
		resolver.Resolve(ing.GetAnnotations()[k8s.SecurityGroups])
	}
	return resolver.Result()
}

func (d *DefaultEngineBuilder) routeOpts(ing networking.Ingress) (builders.RouteResolveOpts, error) {
	r := d.resolvers.RouteOpts()
	annotations := ing.GetAnnotations()
	return r.Resolve(
		annotations[k8s.RequestTimeout],
		annotations[k8s.IdleTimeout],
		annotations[k8s.PrefixRewrite],
		annotations[k8s.UpgradeTypes],
		annotations[k8s.Protocol],
		annotations[k8s.UseRegex],
	)
}

func (d *DefaultEngineBuilder) vhOpts(ing networking.Ingress) (builders.VirtualHostResolveOpts, error) {
	r := d.resolvers.VirtualHostOpts()
	annotations := ing.GetAnnotations()
	return r.Resolve(
		annotations[k8s.ModifyResponseHeaderRemove],
		annotations[k8s.ModifyResponseHeaderRename],
		annotations[k8s.ModifyResponseHeaderAppend],
		annotations[k8s.ModifyResponseHeaderReplace],
		annotations[k8s.SecurityProfileID],
	)
}

func (d *DefaultEngineBuilder) backendOpts(ing networking.Ingress) (builders.BackendResolveOpts, error) {
	annotations := ing.GetAnnotations()
	r := d.resolvers.BackendOpts()
	return r.Resolve(
		annotations[k8s.Protocol], annotations[k8s.BalancingMode], annotations[k8s.TransportSecurity],
		annotations[k8s.SessionAffinityHeader], annotations[k8s.SessionAffinityCookie],
		annotations[k8s.SessionAffinityConnection],
	)
}

func (d *DefaultEngineBuilder) directResponses(ing networking.Ingress) (map[string]*apploadbalancer.DirectResponseAction, error) {
	result := make(map[string]*apploadbalancer.DirectResponseAction)
	annotations := ing.GetAnnotations()

	for key, value := range annotations {
		if !strings.HasPrefix(key, k8s.DirectResponsePrefix) {
			continue
		}

		configs, err := k8s.ParseConfigsFromAnnotationValue(value)
		if err != nil {
			return nil, err
		}

		statusCode, err := strconv.Atoi(configs["status"])
		if err != nil {
			return nil, err
		}

		result[strings.TrimPrefix(key, k8s.DirectResponsePrefix)] = &apploadbalancer.DirectResponseAction{
			Body:   &apploadbalancer.Payload{Payload: &apploadbalancer.Payload_Text{Text: configs["body"]}},
			Status: int64(statusCode),
		}
	}

	return result, nil
}

func (d *DefaultEngineBuilder) buildVirtualHosts(g *k8s.IngressGroup) (*builders.VirtualHostData, *builders.VirtualHostData, error) {
	d.factory.RestartVirtualHostIDGenerator()
	httpVHBuilder := d.factory.VirtualHostBuilder(g.Tag, d.bgFinder)
	tlsVHBuilder := d.factory.TLSVirtualHostBuilder(g.Tag, d.bgFinder)
	for _, ing := range g.Items {
		routeOpts, err := d.routeOpts(ing)
		if err != nil {
			return nil, nil, err
		}
		vhOpts, err := d.vhOpts(ing)
		if err != nil {
			return nil, nil, err
		}

		tlsVHBuilder.SetOpts(routeOpts, vhOpts, ing.Namespace)
		httpVHBuilder.SetOpts(routeOpts, vhOpts, ing.Namespace)

		directResponseActions, err := d.directResponses(ing)
		if err != nil {
			return nil, nil, err
		}

		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}

			for _, path := range rule.HTTP.Paths {
				if path.Backend.Resource != nil && path.Backend.Resource.Kind == "DirectResponse" {
					directResponse, found := directResponseActions[path.Backend.Resource.Name]
					if !found {
						return nil, nil, fmt.Errorf("direct response action for host %s and path %s not found", rule.Host, path.Path)
					}

					err = httpVHBuilder.AddHTTPDirectResponse(rule.Host, path, directResponse)
					if err != nil {
						return nil, nil, err
					}
					err = tlsVHBuilder.AddHTTPDirectResponse(rule.Host, path, directResponse)
					if err != nil {
						return nil, nil, err
					}
					continue
				}

				if path.Backend.Resource != nil && path.Backend.Resource.Kind == "HttpBackendGroup" {
					if !k8s.IsTLS(rule.Host, ing.Spec.TLS) {
						err = httpVHBuilder.AddRouteCR(rule.Host, path)
						if err != nil {
							return nil, nil, err
						}
						continue
					}

					err = httpVHBuilder.AddHTTPRedirect(rule.Host, path)
					if err != nil {
						return nil, nil, err
					}

					err = tlsVHBuilder.AddRouteCR(rule.Host, path)
					if err != nil {
						return nil, nil, err
					}

					continue
				}

				if path.Backend.Service != nil {
					if !k8s.IsTLS(rule.Host, ing.Spec.TLS) {
						err = httpVHBuilder.AddRoute(rule.Host, path)
						if err != nil {
							return nil, nil, err
						}
						continue
					}

					if routeOpts.BackendType != builders.GRPC {
						err = httpVHBuilder.AddHTTPRedirect(rule.Host, path)
						if err != nil {
							return nil, nil, err
						}
					}

					err = tlsVHBuilder.AddRoute(rule.Host, path)
					if err != nil {
						return nil, nil, err
					}
				}
			}
		}
	}
	return httpVHBuilder.Build(), tlsVHBuilder.Build(), nil
}

func (d *DefaultEngineBuilder) buildSNIMatches(ctx context.Context, g *k8s.IngressGroup) ([]*apploadbalancer.SniMatch, error) {
	b := d.factory.HandlerBuilder(g.Tag)
	for _, ing := range g.Items {
		for _, tls := range ing.Spec.TLS {
			if strings.HasPrefix(tls.SecretName, k8s.CertIDPrefix) {
				certID := strings.TrimPrefix(tls.SecretName, k8s.CertIDPrefix)
				b.AddCertificate(tls.Hosts, certID)
				continue
			}

			nn := types.NamespacedName{Name: tls.SecretName, Namespace: ing.Namespace}
			if nn.Namespace == "" {
				nn.Namespace = "default"
			}

			certName := d.names.Certificate(nn)
			cert, err := d.certRepo.LoadCertificate(ctx, certName)
			if err != nil {
				return nil, err
			}

			if cert == nil {
				return nil, fmt.Errorf("there is no (yet?) certificate for secret %s in cloud with name: %s", tls.SecretName, certName)
			}

			b.AddCertificate(tls.Hosts, cert.Id)
		}
	}
	return b.Build(), nil
}

func (d *DefaultEngineBuilder) buildHTTPHandler(_ *k8s.IngressGroup) *apploadbalancer.HttpHandler {
	return &apploadbalancer.HttpHandler{}
}

func (d *DefaultEngineBuilder) buildBalancer(handler *apploadbalancer.HttpHandler, matches []*apploadbalancer.SniMatch, logOpts *apploadbalancer.LogOptions,
	tag string, opts builders.Options) *apploadbalancer.LoadBalancer {
	b := d.factory.BalancerBuilder(tag)
	return b.Build(handler, matches, logOpts, opts)
}

func (d *DefaultEngineBuilder) buildLogOptions(settings *v1alpha1.IngressGroupSettings) *apploadbalancer.LogOptions {
	if settings == nil {
		return nil
	}

	logOpts := settings.LogOptions

	discardRules := make([]*apploadbalancer.LogDiscardRule, 0, len(logOpts.DiscardRules))
	for _, rule := range logOpts.DiscardRules {
		intervals := make([]apploadbalancer.HttpCodeInterval, 0, len(rule.HTTPCodeIntervals))
		for _, interval := range rule.HTTPCodeIntervals {
			intervals = append(intervals, apploadbalancer.HttpCodeInterval(apploadbalancer.HttpCodeInterval_value[interval]))
		}

		grpcCodes := make([]code.Code, 0, 0)
		for _, grpcCode := range rule.GRPCCodes {
			grpcCodes = append(grpcCodes, code.Code(code.Code_value[grpcCode]))
		}

		var discardPercent *wrapperspb.Int64Value
		if rule.DiscardPercent != nil {
			discardPercent = wrapperspb.Int64(*rule.DiscardPercent)
		}

		discardRules = append(discardRules, &apploadbalancer.LogDiscardRule{
			HttpCodes:         rule.HTTPCodes,
			HttpCodeIntervals: intervals,
			GrpcCodes:         grpcCodes,
			DiscardPercent:    discardPercent,
		})
	}

	return &apploadbalancer.LogOptions{
		LogGroupId:   logOpts.LogGroupID,
		DiscardRules: discardRules,
		Disable:      logOpts.Disable,
	}
}
