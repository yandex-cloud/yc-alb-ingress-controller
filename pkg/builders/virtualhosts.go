package builders

import (
	"cmp"
	"context"
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/algo"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/algo/maps"
	errors2 "github.com/yandex-cloud/yc-alb-ingress-controller/pkg/errors"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/metadata"
	"google.golang.org/protobuf/types/known/durationpb"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
)

type HTTPRouterData struct {
	Router *apploadbalancer.HttpRouter
}

type HTTPRouterBuilder struct {
	vhs map[string]*VirtualHost

	nextRouteID interface{ Next() int }
	nextVHID    interface{ Next() int }
	labels      *metadata.Labels
	names       *metadata.Names

	tag      string
	folderID string
	isTLS    bool

	vhOpts    VirtualHostResolveOpts
	routeOpts RouteResolveOpts
	ingNs     string

	backendGroupFinder BackendGroupFinder
}

type VirtualHost struct {
	host    string
	order   int
	opts    VirtualHostResolveOpts
	hpCount map[HostAndPath]int
	routes  []*apploadbalancer.Route
}

type VirtualHostResolveOpts struct {
	ModifyResponse ModifyHeaderOpts
	ModifyRequest  ModifyHeaderOpts

	SecurityProfileID string
}

type ModifyHeaderOpts struct {
	Remove  map[string]bool
	Rename  map[string]string
	Replace map[string]string
	Append  map[string]string
}

type RouteResolveOpts struct {
	Timeout        *durationpb.Duration
	IdleTimeout    *durationpb.Duration
	PrefixRewrite  string
	UpgradeTypes   []string
	BackendType    BackendType
	UseRegex       bool
	AllowedMethods []string
}

type BackendGroupFinder interface {
	FindBackendGroup(ctx context.Context, name string) (*apploadbalancer.BackendGroup, error)
}

func (b *HTTPRouterBuilder) SetOpts(
	vhOpts VirtualHostResolveOpts,
	routeOpts RouteResolveOpts,
	ingNs string,
) {
	b.vhOpts = vhOpts
	b.routeOpts = routeOpts
	b.ingNs = ingNs
}

func (b *HTTPRouterBuilder) AddRoute(hp HostAndPath, svcName string, svcPort int64) error {
	bgName := b.names.BackendGroupForSvcPort(types.NamespacedName{
		Namespace: b.ingNs,
		Name:      svcName,
	}, svcPort)
	bg, err := b.backendGroupFinder.FindBackendGroup(context.TODO(), bgName)

	if bg == nil {
		return errors2.ResourceNotReadyError{ResourceType: "BackendGroup", Name: bgName}
	}

	if err != nil {
		return fmt.Errorf("error finding backend group: %w", err)
	}

	var route *apploadbalancer.Route
	if b.routeOpts.BackendType == GRPC {
		route = grpcRoute(hp, b.routeOpts, bg.Id)
	} else {
		route = httpRoute(hp, b.routeOpts, bg.Id)
	}

	return b.appendRoute(hp, route)
}

func (b *HTTPRouterBuilder) AddRouteToResource(hp HostAndPath, resourceName string) error {
	bgName := b.names.BackendGroupForCR(b.ingNs, resourceName)
	bg, err := b.backendGroupFinder.FindBackendGroup(context.TODO(), bgName)

	if bg == nil {
		return errors2.ResourceNotReadyError{ResourceType: "BackendGroup", Name: bgName}
	}

	if err != nil {
		return fmt.Errorf("error finding backend group: %w", err)
	}

	var route *apploadbalancer.Route
	if b.routeOpts.BackendType == GRPC {
		route = grpcRoute(hp, b.routeOpts, bg.Id)
	} else {
		route = httpRoute(hp, b.routeOpts, bg.Id)
	}

	return b.appendRoute(hp, route)
}

func (b *HTTPRouterBuilder) AddHTTPDirectResponse(hp HostAndPath, directResponse *apploadbalancer.DirectResponseAction) error {
	action := &apploadbalancer.HttpRoute_DirectResponse{
		DirectResponse: directResponse,
	}
	route := httpRouteForAction(hp, action, b.routeOpts.AllowedMethods)

	return b.appendRoute(hp, route)
}

func (b *HTTPRouterBuilder) AddRedirectToHTTPS(hp HostAndPath) error {
	action := &apploadbalancer.HttpRoute_Redirect{
		Redirect: &apploadbalancer.RedirectAction{
			ReplaceScheme: "https",
			ReplacePort:   443,
			RemoveQuery:   false,
			ResponseCode:  apploadbalancer.RedirectAction_MOVED_PERMANENTLY,
		},
	}
	route := httpRouteForAction(hp, action, b.routeOpts.AllowedMethods)

	return b.appendRoute(hp, route)
}

func (b *HTTPRouterBuilder) AddRedirect(hp HostAndPath, redirect *apploadbalancer.RedirectAction) error {
	action := &apploadbalancer.HttpRoute_Redirect{
		Redirect: redirect,
	}
	route := httpRouteForAction(hp, action, b.routeOpts.AllowedMethods)

	return b.appendRoute(hp, route)
}

func (b *HTTPRouterBuilder) GetHosts() map[string]*VirtualHost {
	return b.vhs
}

func (b *HTTPRouterBuilder) Build() *HTTPRouterData {
	httpVirtualHosts := make([]*apploadbalancer.VirtualHost, len(b.vhs))
	hostOrder := make([]*VirtualHost, len(b.vhs))
	for _, vh := range b.vhs {
		hostOrder[vh.order] = vh
	}

	for i, vh := range hostOrder {
		httpVirtualHosts[i] = &apploadbalancer.VirtualHost{
			Name:         b.names.VirtualHostForID(b.tag, b.nextVHID.Next()),
			Authority:    []string{vh.host},
			Routes:       vh.routes,
			RouteOptions: buildRouteOpts(vh.opts.ModifyResponse, vh.opts.ModifyRequest, vh.opts.SecurityProfileID),
		}
	}

	routerNameFn := b.names.Router
	if b.isTLS {
		routerNameFn = b.names.RouterTLS
	}
	router := &apploadbalancer.HttpRouter{
		FolderId:     b.folderID,
		Name:         routerNameFn(b.tag),
		Description:  "router for k8s ingress with tag: " + b.tag,
		Labels:       b.labels.Default(),
		VirtualHosts: httpVirtualHosts,
	}

	return &HTTPRouterData{
		Router: router,
	}
}

func httpRoute(hp HostAndPath, opts RouteResolveOpts, bgID string) *apploadbalancer.Route {
	var action apploadbalancer.HttpRoute_Action = &apploadbalancer.HttpRoute_Route{
		Route: &apploadbalancer.HttpRouteAction{
			Timeout:        opts.Timeout,
			IdleTimeout:    opts.IdleTimeout,
			PrefixRewrite:  opts.PrefixRewrite,
			UpgradeTypes:   opts.UpgradeTypes,
			BackendGroupId: bgID,
		},
	}
	return httpRouteForAction(hp, action, opts.AllowedMethods)
}

func httpRouteForAction(hp HostAndPath, action apploadbalancer.HttpRoute_Action, methods []string) *apploadbalancer.Route {
	return &apploadbalancer.Route{
		Route: &apploadbalancer.Route_Http{
			Http: &apploadbalancer.HttpRoute{
				Match:  &apploadbalancer.HttpRouteMatch{Path: matchForPath(hp), HttpMethod: methods},
				Action: action,
			},
		},
	}
}

func grpcRoute(hp HostAndPath, opts RouteResolveOpts, bgID string) *apploadbalancer.Route {
	action := &apploadbalancer.GrpcRoute_Route{
		Route: &apploadbalancer.GrpcRouteAction{
			MaxTimeout:     opts.Timeout,
			IdleTimeout:    opts.IdleTimeout,
			BackendGroupId: bgID,
		},
	}
	return &apploadbalancer.Route{
		Route: &apploadbalancer.Route_Grpc{
			Grpc: &apploadbalancer.GrpcRoute{
				Match:  &apploadbalancer.GrpcRouteMatch{Fqmn: matchForPath(hp)},
				Action: action,
			},
		},
	}
}

func (b *HTTPRouterBuilder) appendRoute(hp HostAndPath, route *apploadbalancer.Route) error {
	err := b.buildVH(hp.Host)
	if err != nil {
		return fmt.Errorf("error building virtual host: %w", err)
	}

	route.Name = b.routeName(hp)
	b.vhs[hp.Host].routes = append(b.vhs[hp.Host].routes, route)
	b.vhs[hp.Host].hpCount[hp]++
	return nil
}

func (b *HTTPRouterBuilder) buildVH(host string) error {
	if vh, ok := b.vhs[host]; ok {
		var err error
		vh.opts, err = mergeVHOpts(vh.opts, b.vhOpts)
		if err != nil {
			return fmt.Errorf("can't merge vh options: %w", err)
		}
		return nil
	}

	b.vhs[host] = &VirtualHost{
		host:    host,
		order:   len(b.vhs),
		opts:    b.vhOpts.Clone(),
		routes:  []*apploadbalancer.Route{},
		hpCount: make(map[HostAndPath]int),
	}

	return nil
}

func (b *HTTPRouterBuilder) routeName(hp HostAndPath) string {
	vh, ok := b.vhs[hp.Host]
	// If route with specific HostPath is first in VirtualHost: add without order index
	// Because backward compatibilities
	if !ok || vh.hpCount[hp] == 0 {
		return b.names.RouteForPath(b.tag, hp.Host, hp.Path, hp.PathType)
	}

	return b.names.RouteForPath2(b.tag, hp.Host, hp.Path, hp.PathType, vh.hpCount[hp])
}

func mergeVHOpts(opts1, opts2 VirtualHostResolveOpts) (VirtualHostResolveOpts, error) {
	opts := VirtualHostResolveOpts{}
	sID1 := opts1.SecurityProfileID
	sID2 := opts2.SecurityProfileID
	if sID1 != sID2 && (sID1 != "" && sID2 != "") {
		return opts, fmt.Errorf("conflict with vh security profiles: %s and %s", opts1.SecurityProfileID, opts2.SecurityProfileID)
	}

	opts.SecurityProfileID = cmp.Or(sID1, sID2)

	mergeModifyHeader := func(opts1, opts2 ModifyHeaderOpts) (ModifyHeaderOpts, error) {
		opts := ModifyHeaderOpts{}

		var err error
		opts.Append, err = algo.MapMerge(opts1.Append, opts2.Append)
		if err != nil {
			return opts, fmt.Errorf("conflict with vh modify response append: %w", err)
		}

		opts.Remove, err = algo.MapMerge(opts1.Remove, opts2.Remove)
		if err != nil {
			return opts, fmt.Errorf("conflict with vh modify response remove: %w", err)
		}

		opts.Rename, err = algo.MapMerge(opts1.Rename, opts2.Rename)
		if err != nil {
			return opts, fmt.Errorf("conflict with vh modify response rename: %w", err)
		}

		opts.Replace, err = algo.MapMerge(opts1.Replace, opts2.Replace)
		if err != nil {
			return opts, fmt.Errorf("conflict with vh modify response replace: %w", err)
		}

		return opts, nil
	}

	var err error
	opts.ModifyResponse, err = mergeModifyHeader(opts1.ModifyResponse, opts2.ModifyResponse)
	if err != nil {
		return opts, fmt.Errorf("can't merge vh modify response: %w", err)
	}

	opts.ModifyRequest, err = mergeModifyHeader(opts1.ModifyRequest, opts2.ModifyRequest)
	if err != nil {
		return opts, fmt.Errorf("can't merge vh modify request: %w", err)
	}

	return opts, nil
}

func buildModifyHeaderOpts(modifyHeader ModifyHeaderOpts) []*apploadbalancer.HeaderModification {
	expLen := len(modifyHeader.Append) + len(modifyHeader.Rename) + len(modifyHeader.Remove) + len(modifyHeader.Replace)
	if expLen == 0 {
		return nil
	}

	modifyResponseHeaders := make(
		[]*apploadbalancer.HeaderModification, 0, expLen,
	)

	for name, remove := range modifyHeader.Remove {
		modifyResponseHeaders = append(modifyResponseHeaders, &apploadbalancer.HeaderModification{
			Name: name,
			Operation: &apploadbalancer.HeaderModification_Remove{
				Remove: remove,
			},
		})
	}

	for name, replace := range modifyHeader.Replace {
		modifyResponseHeaders = append(modifyResponseHeaders, &apploadbalancer.HeaderModification{
			Name: name,
			Operation: &apploadbalancer.HeaderModification_Replace{
				Replace: replace,
			},
		})
	}

	for name, rename := range modifyHeader.Rename {
		modifyResponseHeaders = append(modifyResponseHeaders, &apploadbalancer.HeaderModification{
			Name: name,
			Operation: &apploadbalancer.HeaderModification_Rename{
				Rename: rename,
			},
		})
	}

	for name, value := range modifyHeader.Append {
		modifyResponseHeaders = append(modifyResponseHeaders, &apploadbalancer.HeaderModification{
			Name: name,
			Operation: &apploadbalancer.HeaderModification_Append{
				Append: value,
			},
		})
	}
	return modifyResponseHeaders
}

func buildRouteOpts(modifyResponseOpts, modifyRequestOpts ModifyHeaderOpts, securityProfileID string) *apploadbalancer.RouteOptions {
	modifyResponseHeaders := buildModifyHeaderOpts(modifyResponseOpts)
	modifyRequestHeaders := buildModifyHeaderOpts(modifyRequestOpts)
	if len(modifyResponseHeaders) == 0 && securityProfileID == "" {
		return nil
	}

	return &apploadbalancer.RouteOptions{
		ModifyResponseHeaders: modifyResponseHeaders,
		ModifyRequestHeaders:  modifyRequestHeaders,
		SecurityProfileId:     securityProfileID,
	}
}

func matchForPath(hp HostAndPath) *apploadbalancer.StringMatch {
	if hp.Path == "" {
		return nil
	}

	var match apploadbalancer.StringMatch_Match
	switch {
	case hp.PathType == PathTypeRegex:
		match = &apploadbalancer.StringMatch_RegexMatch{RegexMatch: hp.Path}
	case hp.PathType == string(networking.PathTypePrefix):
		match = &apploadbalancer.StringMatch_PrefixMatch{PrefixMatch: hp.Path}
	default:
		match = &apploadbalancer.StringMatch_ExactMatch{ExactMatch: hp.Path}
	}

	return &apploadbalancer.StringMatch{Match: match}
}

func (opts VirtualHostResolveOpts) Clone() VirtualHostResolveOpts {
	return VirtualHostResolveOpts{
		SecurityProfileID: opts.SecurityProfileID,
		ModifyResponse: ModifyHeaderOpts{
			Append:  maps.Clone(opts.ModifyResponse.Append),
			Remove:  maps.Clone(opts.ModifyResponse.Remove),
			Replace: maps.Clone(opts.ModifyResponse.Replace),
			Rename:  maps.Clone(opts.ModifyResponse.Rename),
		},
		ModifyRequest: ModifyHeaderOpts{
			Append:  maps.Clone(opts.ModifyRequest.Append),
			Remove:  maps.Clone(opts.ModifyRequest.Remove),
			Replace: maps.Clone(opts.ModifyRequest.Replace),
			Rename:  maps.Clone(opts.ModifyRequest.Rename),
		},
	}
}
