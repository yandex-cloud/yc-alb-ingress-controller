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

type (
	createRouteFn        func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error)
	createRouteWithSvcFn func(name string, match *apploadbalancer.StringMatch, svcName string) (*apploadbalancer.Route, error)
)

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

	isTLS          bool
	useRegex       bool
	allowedMethods []string

	createRoute   createRouteWithSvcFn
	createRouteCR createRouteWithSvcFn

	modifyResponseOpts ModifyResponseOpts
	securityProfileID  string

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
	Timeout        *durationpb.Duration
	IdleTimeout    *durationpb.Duration
	PrefixRewrite  string
	UpgradeTypes   []string
	BackendType    BackendType
	UseRegex       bool
	AllowedMethods []string
}

type VirtualHostResolveOpts struct {
	ModifyResponse ModifyResponseOpts

	SecurityProfileID string
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
	return httpRouteForAction(name, match, action, opts.AllowedMethods)
}

func httpDirectResponseRoute(name string, match *apploadbalancer.StringMatch, directResponse *apploadbalancer.DirectResponseAction, methods []string) *apploadbalancer.Route {
	action := &apploadbalancer.HttpRoute_DirectResponse{
		DirectResponse: directResponse,
	}

	return httpRouteForAction(name, match, action, methods)
}

func httpRouteForAction(name string, match *apploadbalancer.StringMatch, action apploadbalancer.HttpRoute_Action, methods []string) *apploadbalancer.Route {
	return &apploadbalancer.Route{
		Name: name,
		Route: &apploadbalancer.Route_Http{
			Http: &apploadbalancer.HttpRoute{
				Match:  &apploadbalancer.HttpRouteMatch{Path: match, HttpMethod: methods},
				Action: action,
			},
		},
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
	return &apploadbalancer.Route{
		Name: name,
		Route: &apploadbalancer.Route_Grpc{
			Grpc: &apploadbalancer.GrpcRoute{
				Match:  &apploadbalancer.GrpcRouteMatch{Fqmn: match},
				Action: action,
			},
		},
	}
}

func (b *VirtualHostBuilder) GetHosts() map[string]HostInfo {
	return b.hosts
}

func (b *VirtualHostBuilder) AddHTTPDirectResponse(hp HostAndPath, directResponse *apploadbalancer.DirectResponseAction) error {
	if _, ok := b.httpRouteMap[hp]; ok {
		return nil
	}

	return b.addRoute(hp, func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
		return httpDirectResponseRoute(name, match, directResponse, b.allowedMethods), nil
	})
}

func (b *VirtualHostBuilder) AddRedirect(hp HostAndPath, redirect *apploadbalancer.RedirectAction) error {
	// do not overwrite route action with redirect
	if _, ok := b.httpRouteMap[hp]; ok {
		return nil
	}

	createHTTPRedirect := func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
		var action apploadbalancer.HttpRoute_Action = &apploadbalancer.HttpRoute_Redirect{
			Redirect: redirect,
		}
		return httpRouteForAction(name, match, action, b.allowedMethods), nil
	}

	return b.addRoute(hp, createHTTPRedirect)
}

func (b *VirtualHostBuilder) AddHTTPRedirect(hp HostAndPath) error {
	// do not overwrite route action with redirect
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
		return httpRouteForAction(name, match, action, b.allowedMethods), nil
	}

	return b.addRoute(hp, createHTTPRedirect)
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

	b.securityProfileID = vhOpts.SecurityProfileID
	b.modifyResponseOpts = vhOpts.ModifyResponse
	b.useRegex = routeOpts.UseRegex

	b.allowedMethods = routeOpts.AllowedMethods
}

func (b *VirtualHostBuilder) AddRoute(hp HostAndPath, serviceName string) error {
	return b.addRoute(hp, func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
		return b.createRoute(name, match, serviceName)
	})
}

func (b *VirtualHostBuilder) AddRouteCR(hp HostAndPath, resourceName string) error {
	return b.addRoute(hp, func(name string, match *apploadbalancer.StringMatch) (*apploadbalancer.Route, error) {
		return b.createRouteCR(name, match, resourceName)
	})
}

func (b *VirtualHostBuilder) addRoute(hp HostAndPath, createRoute createRouteFn) error {
	name := b.names.RouteForPath(b.tag, hp.Host, hp.Path, hp.PathType)

	route, err := createRoute(name, matchForPath(hp))
	if err != nil {
		return err
	}

	if _, ok := b.hosts[hp.Host]; !ok {
		b.hosts[hp.Host] = HostInfo{
			Order:              len(b.hosts),
			Host:               hp.Host,
			ModifyResponseOpts: b.modifyResponseOpts,
		}
	}

	if _, ok := b.httpRouteMap[hp]; !ok {
		b.routeOrder = append(b.routeOrder, hp)
	}
	// overwrite with subsequent route for the same Host&Path. Shall we throw an error instead?
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

func buildRouteOpts(modifyResponseOpts ModifyResponseOpts, securityProfileID string) *apploadbalancer.RouteOptions {
	modifyResponseHeaders := buildModifyHeaderOpts(modifyResponseOpts)
	if len(modifyResponseHeaders) == 0 && securityProfileID == "" {
		return nil
	}

	return &apploadbalancer.RouteOptions{
		ModifyResponseHeaders: modifyResponseHeaders,
		SecurityProfileId:     securityProfileID,
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
			RouteOptions: buildRouteOpts(b.modifyResponseOpts, b.securityProfileID),
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

type VirtualHostData struct {
	HTTPRouteMap map[HostAndPath]*apploadbalancer.Route
	Router       *apploadbalancer.HttpRouter
}
