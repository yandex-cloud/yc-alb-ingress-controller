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
	newEngine func(data *builders.Data) *IngressGroupEngine, folderID string, names *metadata.Names, certRepo yc.CertRepo, bgFinder builders.BackendGroupFinder,
) *DefaultEngineBuilder {
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
		annotations[k8s.AllowedMethods],
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

func parseIntValue(s string) (int, error) {
	if s == "" {
		return 0, nil
	}

	return strconv.Atoi(s)
}

func parseBoolValue(s string) (bool, error) {
	if s == "" {
		return false, nil
	}

	return strconv.ParseBool(s)
}

func (d *DefaultEngineBuilder) redirects(ing networking.Ingress) (map[string]*apploadbalancer.RedirectAction, error) {
	result := make(map[string]*apploadbalancer.RedirectAction)
	annotations := ing.GetAnnotations()

	for key, value := range annotations {
		if !strings.HasPrefix(key, k8s.RedirectPrefix) {
			continue
		}

		configs, err := k8s.ParseConfigsFromAnnotationValue(value)
		if err != nil {
			return nil, fmt.Errorf("error parsing annotation %s value %s for ingress %s/%s: %w", ing.Name, ing.Namespace, key, value, err)
		}

		replacePort, err := parseIntValue(configs["replace_port"])
		if err != nil {
			return nil, fmt.Errorf("error parsing replace_port for ingress %s/%s : %w", ing.Namespace, ing.Name, err)
		}

		var path apploadbalancer.RedirectAction_Path
		if configs["path"] == "replace_path" {
			path = &apploadbalancer.RedirectAction_ReplacePath{
				ReplacePath: configs["replace_path"],
			}
		}
		if configs["path"] == "replace_prefix" {
			path = &apploadbalancer.RedirectAction_ReplacePrefix{
				ReplacePrefix: configs["replace_prefix"],
			}
		}

		removeQuery, err := parseBoolValue(configs["remove_query"])
		if err != nil {
			return nil, fmt.Errorf("error parsing remove_query for ingress %s/%s : %w", ing.Namespace, ing.Name, err)
		}

		responseCode, ok := apploadbalancer.RedirectAction_RedirectResponseCode_value[configs["response_code"]]
		if !ok {
			return nil, fmt.Errorf("unknown redirect response_code for ingress %s/%s : %s", ing.Namespace, ing.Name, configs["response_code"])
		}

		result[strings.TrimPrefix(key, k8s.RedirectPrefix)] = &apploadbalancer.RedirectAction{
			ReplaceScheme: configs["replace_scheme"],
			ReplaceHost:   configs["replace_host"],
			ReplacePort:   int64(replacePort),
			Path:          path,
			RemoveQuery:   removeQuery,
			ResponseCode:  apploadbalancer.RedirectAction_RedirectResponseCode(responseCode),
		}
	}

	return result, nil
}

func (d *DefaultEngineBuilder) buildVirtualHosts(g *k8s.IngressGroup) (*builders.VirtualHostData, *builders.VirtualHostData, error) {
	d.factory.RestartVirtualHostIDGenerator()
	httpVHBuilder := d.factory.VirtualHostBuilder(g.Tag, d.bgFinder)
	tlsVHBuilder := d.factory.TLSVirtualHostBuilder(g.Tag, d.bgFinder)

	handleBackend := func(
		backend networking.IngressBackend,
		hp builders.HostAndPath,
		isTlS bool,
		backendType builders.BackendType,
		directResponseActions map[string]*apploadbalancer.DirectResponseAction,
		redirectActions map[string]*apploadbalancer.RedirectAction,
	) error {
		if backend.Resource != nil && backend.Resource.Kind == "DirectResponse" {
			directResponse, found := directResponseActions[backend.Resource.Name]
			if !found {
				return fmt.Errorf("direct response action for host %s and path %s not found", hp.Host, hp.Path)
			}

			err := httpVHBuilder.AddHTTPDirectResponse(hp, directResponse)
			if err != nil {
				return err
			}
			return tlsVHBuilder.AddHTTPDirectResponse(hp, directResponse)
		}

		if backend.Resource != nil && backend.Resource.Kind == "Redirect" {
			redirect, found := redirectActions[backend.Resource.Name]
			if !found {
				return fmt.Errorf("redirect action for host %s and path %s not found", hp.Host, hp.Path)
			}

			err := httpVHBuilder.AddRedirect(hp, redirect)
			if err != nil {
				return err
			}
			return tlsVHBuilder.AddRedirect(hp, redirect)
		}

		if backend.Resource != nil &&
			(backend.Resource.Kind == "HttpBackendGroup" || backend.Resource.Kind == "GrpcBackendGroup") {
			if !isTlS {
				return httpVHBuilder.AddRouteCR(hp, backend.Resource.Name)
			}

			err := httpVHBuilder.AddHTTPRedirect(hp)
			if err != nil {
				return err
			}

			return tlsVHBuilder.AddRouteCR(hp, backend.Resource.Name)
		}

		if backend.Service != nil {
			if !isTlS {
				return httpVHBuilder.AddRoute(hp, backend.Service.Name)
			}

			if backendType != builders.GRPC {
				err := httpVHBuilder.AddHTTPRedirect(hp)
				if err != nil {
					return err
				}
			}

			return tlsVHBuilder.AddRoute(hp, backend.Service.Name)
		}

		return nil
	}

	var ingWithDefaultBackend *networking.Ingress

	for _, ing := range g.Items {
		if ing.Spec.DefaultBackend != nil && ingWithDefaultBackend != nil {
			return nil, nil, fmt.Errorf("default backend can be specified only once, ingress-group: %s", g.Tag)
		}

		if ing.Spec.DefaultBackend != nil {
			ingWithDefaultBackend = &ing
		}

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

		redirectActions, err := d.redirects(ing)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting redirectActions: %w", err)
		}

		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}

			for _, path := range rule.HTTP.Paths {
				hp, err := builders.HTTPIngressPathToHostAndPath(rule.Host, path, routeOpts.UseRegex)
				if err != nil {
					return nil, nil, err
				}

				err = handleBackend(path.Backend, hp, k8s.IsTLS(hp.Host, ing.Spec.TLS), routeOpts.BackendType, directResponseActions, redirectActions)
				if err != nil {
					return nil, nil, err
				}
			}
		}
	}

	if ingWithDefaultBackend != nil {
		ing := *ingWithDefaultBackend

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

		redirectActions, err := d.redirects(ing)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting redirectActions: %w", err)
		}

		// host and path to match all the request, with hosts different from others
		hp := builders.HostAndPath{
			Host:     "*",
			Path:     "/",
			PathType: string(networking.PathTypePrefix),
		}

		err = handleBackend(*ing.Spec.DefaultBackend, hp, k8s.IsTLS("*", ing.Spec.TLS), routeOpts.BackendType, directResponseActions, redirectActions)
		if err != nil {
			return nil, nil, err
		}

		// handlers with pats, matching all the requests, with hosts specified in ingresses
		for host := range httpVHBuilder.GetHosts() {
			hp = builders.HostAndPath{
				Host:     host,
				Path:     "/",
				PathType: string(networking.PathTypePrefix),
			}

			err = handleBackend(*ing.Spec.DefaultBackend, hp, k8s.IsTLS(host, ing.Spec.TLS), routeOpts.BackendType, directResponseActions, redirectActions)
			if err != nil {
				return nil, nil, err
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
	tag string, opts builders.Options,
) *apploadbalancer.LoadBalancer {
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

		grpcCodes := make([]code.Code, 0)
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
