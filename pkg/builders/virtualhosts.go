package builders

import (
	"context"
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/types/known/durationpb"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"

	errors2 "github.com/yandex-cloud/alb-ingress/pkg/errors"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
)

//go:generate mockgen -destination=./mocks/backendgroups.go -package=mocks . BackendGroupFinder

type createRouteFn func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error)
type createRouteWithSvcFn func(name string, match *apploadbalancer.StringMatch, svcName string) (*apploadbalancer.Route, error)

type BackendGroupFinder interface {
	FindBackendGroup(ctx context.Context, name string) (*apploadbalancer.BackendGroup, error)
}

type VirtualHostBuilder struct {
	httpRouteMap map[HostAndPath]*apploadbalancer.Route
	routeOrder   []HostAndPath
	hosts        map[string]HostInfo

	nextVHID    interface{ Next() int }
	nextRouteID interface{ Next() int }

	names    *metadata.Names
	labels   *metadata.Labels
	tag      string
	folderID string

	isTLS    bool
	useRegex bool

	createRoute   createRouteWithSvcFn
	createRouteCR createRouteWithSvcFn

	modifyResponseOpts ModifyResponseOpts

	backendGroupFinder BackendGroupFinder
}

type HostInfo struct {
	Host  string
	Order int

	ModifyResponseOpts ModifyResponseOpts
}

type ModifyResponseOpts struct {
	Remove  map[string]bool
	Rename  map[string]string
	Replace map[string]string
	Append  map[string]string
}

type RouteResolveOpts struct {
	Timeout       *durationpb.Duration
	IdleTimeout   *durationpb.Duration
	PrefixRewrite string
	UpgradeTypes  []string
	BackendType   BackendType
	UseRegex      bool

	SecurityProfileID string
}

type VirtualHostResolveOpts struct {
	ModifyResponse ModifyResponseOpts
}

func httpRoute(name string, match *apploadbalancer.StringMatch, opts RouteResolveOpts, bgID string) *apploadbalancer.Route {
	var action apploadbalancer.HttpRoute_Action = &apploadbalancer.HttpRoute_Route{
		Route: &apploadbalancer.HttpRouteAction{
			Timeout:        opts.Timeout,
			IdleTimeout:    opts.IdleTimeout,
			PrefixRewrite:  opts.PrefixRewrite,
			UpgradeTypes:   opts.UpgradeTypes,
			BackendGroupId: bgID,
		},
	}

	var routeOptions = &apploadbalancer.RouteOptions{
		SecurityProfileId: opts.SecurityProfileID,
	}

	return httpRouteForActionAndOptions(name, match, action, routeOptions)
}

func httpDirectResponseRoute(name string, match *apploadbalancer.StringMatch, directResponse *apploadbalancer.DirectResponseAction, opts RouteResolveOpts) *apploadbalancer.Route {
	action := &apploadbalancer.HttpRoute_DirectResponse{
		DirectResponse: directResponse,
	}

	var routeOptions = &apploadbalancer.RouteOptions{
		SecurityProfileId: opts.SecurityProfileID,
	}

	return httpRouteForActionAndOptions(name, match, action, routeOptions)
}

func httpRouteForActionAndOptions(
	name string, match *apploadbalancer.StringMatch,
	action apploadbalancer.HttpRoute_Action,
	opts *apploadbalancer.RouteOptions,
) *apploadbalancer.Route {
	return &apploadbalancer.Route{
		Name: name,
		Route: &apploadbalancer.Route_Http{
			Http: &apploadbalancer.HttpRoute{
				Match:  &apploadbalancer.HttpRouteMatch{Path: match},
				Action: action,
			},
		},
		RouteOptions: opts,
	}
}

func grpcRoute(name string, match *apploadbalancer.StringMatch, opts RouteResolveOpts, bgID string) *apploadbalancer.Route {
	action := &apploadbalancer.GrpcRoute_Route{
		Route: &apploadbalancer.GrpcRouteAction{
			MaxTimeout:     opts.Timeout,
			IdleTimeout:    opts.IdleTimeout,
			BackendGroupId: bgID,
		},
	}

	routeOptions := &apploadbalancer.RouteOptions{
		SecurityProfileId: opts.SecurityProfileID,
	}

	return &apploadbalancer.Route{
		Name: name,
		Route: &apploadbalancer.Route_Grpc{
			Grpc: &apploadbalancer.GrpcRoute{
				Match:  &apploadbalancer.GrpcRouteMatch{Fqmn: match},
				Action: action,
			},
		},
		RouteOptions: routeOptions,
	}
}

func (b *VirtualHostBuilder) AddHTTPDirectResponse(
	host string, path networking.HTTPIngressPath,
	directResponse *apploadbalancer.DirectResponseAction,
	routeOpts RouteResolveOpts,
) error {
	hp, err := HTTPIngressPathToHostAndPath(host, path, b.useRegex)
	if err != nil {
		return err
	}

	if _, ok := b.httpRouteMap[hp]; ok {
		return nil
	}

	return b.addRoute(host, path, func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
		return httpDirectResponseRoute(name, match, directResponse, routeOpts), nil
	})
}

func (b *VirtualHostBuilder) AddRedirect(
	host string, path networking.HTTPIngressPath,
	redirect *apploadbalancer.RedirectAction,
	routeOpts RouteResolveOpts,
) error {
	hp, err := HTTPIngressPathToHostAndPath(host, path, b.useRegex)
	if err != nil {
		return err
	}

	//do not overwrite route action with redirect
	if _, ok := b.httpRouteMap[hp]; ok {
		return nil
	}

	createHTTPRedirect := func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
		var action apploadbalancer.HttpRoute_Action = &apploadbalancer.HttpRoute_Redirect{
			Redirect: redirect,
		}

		var routeOptions = &apploadbalancer.RouteOptions{
			SecurityProfileId: routeOpts.SecurityProfileID,
		}

		return httpRouteForActionAndOptions(name, match, action, routeOptions), nil
	}

	return b.addRoute(host, path, createHTTPRedirect)
}

func (b *VirtualHostBuilder) AddHTTPRedirect(host string, path networking.HTTPIngressPath, opts RouteResolveOpts) error {
	hp, err := HTTPIngressPathToHostAndPath(host, path, b.useRegex)
	if err != nil {
		return err
	}

	//do not overwrite route action with redirect
	if _, ok := b.httpRouteMap[hp]; ok {
		return nil
	}

	createHTTPRedirect := func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
		var action apploadbalancer.HttpRoute_Action = &apploadbalancer.HttpRoute_Redirect{
			Redirect: &apploadbalancer.RedirectAction{
				ReplaceScheme: "https",
				ReplacePort:   443,
				RemoveQuery:   false,
				ResponseCode:  apploadbalancer.RedirectAction_MOVED_PERMANENTLY,
			},
		}

		var routeOptions = &apploadbalancer.RouteOptions{
			SecurityProfileId: opts.SecurityProfileID,
		}

		return httpRouteForActionAndOptions(name, match, action, routeOptions), nil
	}

	return b.addRoute(host, path, createHTTPRedirect)
}

func (b *VirtualHostBuilder) SetOpts(routeOpts RouteResolveOpts, vhOpts VirtualHostResolveOpts, ingNamespace string) {
	if routeOpts.BackendType == GRPC {
		createRoute := func(name, bgName string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
			bg, err := b.backendGroupFinder.FindBackendGroup(context.TODO(), bgName)
			if bg == nil {
				return nil, errors2.ResourceNotReadyError{ResourceType: "BackendGroup", Name: bgName}
			}

			if err != nil {
				return nil, fmt.Errorf("error finding backend group: %w", err)
			}

			return grpcRoute(name, match, routeOpts, bg.Id), nil
		}

		b.createRoute = func(name string, match *apploadbalancer.StringMatch, svcName string) (*apploadbalancer.Route, error) {
			bgName := b.names.NewBackendGroup(types.NamespacedName{
				Namespace: ingNamespace,
				Name:      svcName,
			})

			return createRoute(name, bgName, match)
		}
	} else {
		createRoute := func(name, bgName string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
			bg, err := b.backendGroupFinder.FindBackendGroup(context.TODO(), bgName)
			if bg == nil {
				return nil, errors2.ResourceNotReadyError{ResourceType: "BackendGroup", Name: bgName}
			}

			if err != nil {
				return nil, fmt.Errorf("error finding backend group: %w", err)
			}

			return httpRoute(name, match, routeOpts, bg.Id), nil
		}

		b.createRoute = func(name string, match *apploadbalancer.StringMatch, svcName string) (*apploadbalancer.Route, error) {
			bgName := b.names.NewBackendGroup(types.NamespacedName{
				Namespace: ingNamespace,
				Name:      svcName,
			})

			return createRoute(name, bgName, match)
		}

		b.createRouteCR = func(name string, match *apploadbalancer.StringMatch, bgName string) (*apploadbalancer.Route, error) {
			return createRoute(name, b.names.BackendGroupForCR(ingNamespace, bgName), match)
		}
	}

	b.modifyResponseOpts = vhOpts.ModifyResponse
	b.useRegex = routeOpts.UseRegex
}

func (b *VirtualHostBuilder) AddRoute(host string, path networking.HTTPIngressPath) error {
	return b.addRoute(host, path, func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
		return b.createRoute(name, match, path.Backend.Service.Name)
	})
}

func (b *VirtualHostBuilder) AddRouteCR(host string, path networking.HTTPIngressPath) error {
	return b.addRoute(host, path, func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
		return b.createRouteCR(name, match, path.Backend.Resource.Name)
	})
}

func (b *VirtualHostBuilder) addRoute(host string, path networking.HTTPIngressPath, createRoute createRouteFn) error {
	hp, err := HTTPIngressPathToHostAndPath(host, path, b.useRegex)
	if err != nil {
		return err
	}

	name := b.names.RouteForPath(b.tag, hp.Host, hp.Path, hp.PathType)

	route, err := createRoute(name, matchForPath(path, b.useRegex))
	if err != nil {
		return err
	}

	if _, ok := b.hosts[host]; !ok {
		b.hosts[host] = HostInfo{
			Order:              len(b.hosts),
			Host:               host,
			ModifyResponseOpts: b.modifyResponseOpts,
		}
	}

	if _, ok := b.httpRouteMap[hp]; !ok {
		b.routeOrder = append(b.routeOrder, hp)
	}
	//overwrite with subsequent route for the same Host&Path. Shall we throw an error instead?
	b.httpRouteMap[hp] = route
	return nil
}

func buildModifyHeaderOpts(modifyResponse ModifyResponseOpts) []*apploadbalancer.HeaderModification {
	expLen := len(modifyResponse.Append) + len(modifyResponse.Rename) + len(modifyResponse.Remove) + len(modifyResponse.Replace)
	if expLen == 0 {
		return nil
	}

	modifyResponseHeaders := make(
		[]*apploadbalancer.HeaderModification, 0, expLen,
	)

	for name, remove := range modifyResponse.Remove {
		modifyResponseHeaders = append(modifyResponseHeaders, &apploadbalancer.HeaderModification{
			Name: name,
			Operation: &apploadbalancer.HeaderModification_Remove{
				Remove: remove,
			},
		})
	}

	for name, replace := range modifyResponse.Replace {
		modifyResponseHeaders = append(modifyResponseHeaders, &apploadbalancer.HeaderModification{
			Name: name,
			Operation: &apploadbalancer.HeaderModification_Replace{
				Replace: replace,
			},
		})
	}

	for name, rename := range modifyResponse.Rename {
		modifyResponseHeaders = append(modifyResponseHeaders, &apploadbalancer.HeaderModification{
			Name: name,
			Operation: &apploadbalancer.HeaderModification_Rename{
				Rename: rename,
			},
		})
	}

	for name, value := range modifyResponse.Append {
		modifyResponseHeaders = append(modifyResponseHeaders, &apploadbalancer.HeaderModification{
			Name: name,
			Operation: &apploadbalancer.HeaderModification_Append{
				Append: value,
			},
		})
	}
	return modifyResponseHeaders
}

func buildRouteOpts(modifyResponseOpts ModifyResponseOpts) *apploadbalancer.RouteOptions {
	modifyResponseHeaders := buildModifyHeaderOpts(modifyResponseOpts)
	if len(modifyResponseHeaders) == 0 {
		return nil
	}

	return &apploadbalancer.RouteOptions{
		ModifyResponseHeaders: modifyResponseHeaders,
	}
}

func (b *VirtualHostBuilder) Build() *VirtualHostData {
	if len(b.routeOrder) == 0 {
		return nil
	}
	routeMap := make(map[string][]*apploadbalancer.Route, len(b.hosts))
	httpVirtualHosts := make([]*apploadbalancer.VirtualHost, len(b.hosts))
	hostOrder := make([]HostInfo, len(b.hosts))
	for _, hostAndPath := range b.routeOrder {
		routeMap[hostAndPath.Host] = append(routeMap[hostAndPath.Host], b.httpRouteMap[hostAndPath])
	}
	for _, info := range b.hosts {
		hostOrder[info.Order] = info
	}
	for i, host := range hostOrder {
		httpVirtualHosts[i] = &apploadbalancer.VirtualHost{
			Name:         b.names.VirtualHostForID(b.tag, b.nextVHID.Next()),
			Authority:    []string{host.Host},
			Routes:       routeMap[host.Host],
			RouteOptions: buildRouteOpts(b.modifyResponseOpts),
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

	return &VirtualHostData{
		Router:       router,
		HTTPRouteMap: b.httpRouteMap,
	}
}

func matchForPath(path networking.HTTPIngressPath, useRegex bool) *apploadbalancer.StringMatch {
	if path.Path == "" {
		return nil
	}
	var match apploadbalancer.StringMatch_Match
	switch {
	case useRegex:
		match = &apploadbalancer.StringMatch_RegexMatch{RegexMatch: path.Path}
	case path.PathType == nil, *path.PathType == networking.PathTypePrefix:
		match = &apploadbalancer.StringMatch_PrefixMatch{PrefixMatch: path.Path}
	default:
		match = &apploadbalancer.StringMatch_ExactMatch{ExactMatch: path.Path}
	}
	return &apploadbalancer.StringMatch{Match: match}
}

type VirtualHostData struct {
	HTTPRouteMap map[HostAndPath]*apploadbalancer.Route
	Router       *apploadbalancer.HttpRouter
}
