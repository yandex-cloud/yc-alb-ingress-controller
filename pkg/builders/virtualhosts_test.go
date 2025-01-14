package builders

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/proto"
	networking "k8s.io/api/networking/v1"

	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/algo"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/builders/mocks"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/metadata"
)

func TestVirtualHosts(t *testing.T) {
	const (
		tag            = "tag"
		folderID       = "my-folder"
		clusterID      = "my-cluster"
		backendGroupID = "backend-group-id"
	)

	var (
		exact                  = networking.PathTypeExact
		prefix                 = networking.PathTypePrefix
		implementationSpecific = networking.PathTypeImplementationSpecific
	)
	var (
		rule1 = &networking.IngressRule{
			Host: "example1.com",
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{
						{
							Path:     "/tread/lightly",
							PathType: &exact,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: "service-name",
								},
							},
						},
					},
				},
			},
		}
		rule2 = &networking.IngressRule{
			Host: "example1.com",
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{
						{
							Path:     "/tread",
							PathType: &prefix,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: "service-name",
								},
							},
						},
					},
				},
			},
		}
		rule3 = &networking.IngressRule{
			Host: "example2.com",
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{
						{
							Path:     "/saunter",
							PathType: &exact,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: "service-name",
								},
							},
						},
					},
				},
			},
		}
		rule4 = &networking.IngressRule{
			Host: "example1.com",
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{
						{
							Path:     "/stagger",
							PathType: &exact,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: "service-name",
								},
							},
						},
					},
				},
			},
		}
		rule1OverwriteForPrefix = &networking.IngressRule{
			Host: "example1.com",
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{
						{
							Path:     "/tread/lightly",
							PathType: &prefix,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: "service-name",
								},
							},
						},
					},
				},
			},
		}
		rule1OverwriteForExact = &networking.IngressRule{
			Host: "example1.com",
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{
						{
							Path:     "/tread/lightly",
							PathType: &exact,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: "service-name",
								},
							},
						},
					},
				},
			},
		}
		emptyPathRule = &networking.IngressRule{
			Host: "example1.com",
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{
						{
							PathType: &implementationSpecific,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: "service-name",
								},
							},
						},
					},
				},
			},
		}
	)
	var (
		route0 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-0",
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_ExactMatch{ExactMatch: "/tread/lightly"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Route{
						Route: &apploadbalancer.HttpRouteAction{
							BackendGroupId: backendGroupID,
						},
					},
				},
			},
		}
		routeRedirect0 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-1",
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_ExactMatch{ExactMatch: "/tread/lightly"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Redirect{
						Redirect: &apploadbalancer.RedirectAction{
							ReplaceScheme: "https",
							ReplacePort:   443,
							Path:          nil,
							RemoveQuery:   false,
							ResponseCode:  apploadbalancer.RedirectAction_MOVED_PERMANENTLY,
						},
					},
				},
			},
		}
		route1 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-1",
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_PrefixMatch{PrefixMatch: "/tread"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Route{
						Route: &apploadbalancer.HttpRouteAction{
							BackendGroupId: backendGroupID,
						},
					},
				},
			},
		}
		route2 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-2",
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_ExactMatch{ExactMatch: "/saunter"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Route{
						Route: &apploadbalancer.HttpRouteAction{
							BackendGroupId: backendGroupID,
						},
					},
				},
			},
		}
		route3 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-3",
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_ExactMatch{ExactMatch: "/stagger"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Redirect{
						Redirect: &apploadbalancer.RedirectAction{
							ReplaceScheme: "https",
							ReplacePort:   443,
							Path:          nil,
							RemoveQuery:   false,
							ResponseCode:  apploadbalancer.RedirectAction_MOVED_PERMANENTLY,
						},
					},
				},
			},
		}
		regexroute0 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-0",
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_RegexMatch{RegexMatch: "/tread/lightly"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Route{
						Route: &apploadbalancer.HttpRouteAction{
							BackendGroupId: backendGroupID,
						},
					},
				},
			},
		}
		regexroute1 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-1",
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_RegexMatch{RegexMatch: "/saunter"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Route{
						Route: &apploadbalancer.HttpRouteAction{
							BackendGroupId: backendGroupID,
						},
					},
				},
			},
		}
		regexroute2 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-2",
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_RegexMatch{RegexMatch: "/stagger"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Redirect{
						Redirect: &apploadbalancer.RedirectAction{
							ReplaceScheme: "https",
							ReplacePort:   443,
							Path:          nil,
							RemoveQuery:   false,
							ResponseCode:  apploadbalancer.RedirectAction_MOVED_PERMANENTLY,
						},
					},
				},
			},
		}
		appendedRoute0 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-1", // should be added second
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_PrefixMatch{PrefixMatch: "/tread/lightly"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Route{
						Route: &apploadbalancer.HttpRouteAction{
							BackendGroupId: backendGroupID,
						},
					},
				},
			},
		}
		appendedRoute1 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-1", // should be added second
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_ExactMatch{ExactMatch: "/tread/lightly"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Route{
						Route: &apploadbalancer.HttpRouteAction{
							BackendGroupId: "backend-group-id",
						},
					},
				},
			},
		}
		emptyPathRoute = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-0",
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{},
					Action: &apploadbalancer.HttpRoute_Route{
						Route: &apploadbalancer.HttpRouteAction{
							BackendGroupId: backendGroupID,
						},
					},
				},
			},
		}
	)
	var (
		routeGRPC0 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-0",
			Route: &apploadbalancer.Route_Grpc{
				Grpc: &apploadbalancer.GrpcRoute{
					Match: &apploadbalancer.GrpcRouteMatch{
						Fqmn: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_ExactMatch{ExactMatch: "/tread/lightly"},
						},
					},
					Action: &apploadbalancer.GrpcRoute_Route{
						Route: &apploadbalancer.GrpcRouteAction{
							BackendGroupId: backendGroupID,
						},
					},
				},
			},
		}
		routeGRPC1 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-1",
			Route: &apploadbalancer.Route_Grpc{
				Grpc: &apploadbalancer.GrpcRoute{
					Match: &apploadbalancer.GrpcRouteMatch{
						Fqmn: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_PrefixMatch{PrefixMatch: "/tread"},
						},
					},
					Action: &apploadbalancer.GrpcRoute_Route{
						Route: &apploadbalancer.GrpcRouteAction{
							BackendGroupId: backendGroupID,
						},
					},
				},
			},
		}
		routeGRPC2 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-2",
			Route: &apploadbalancer.Route_Grpc{
				Grpc: &apploadbalancer.GrpcRoute{
					Match: &apploadbalancer.GrpcRouteMatch{
						Fqmn: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_ExactMatch{ExactMatch: "/saunter"},
						},
					},
					Action: &apploadbalancer.GrpcRoute_Route{
						Route: &apploadbalancer.GrpcRouteAction{
							BackendGroupId: backendGroupID,
						},
					},
				},
			},
		}
		routeGRPC3 = &apploadbalancer.Route{
			Name: "route-07544a934fcd54e50ab30eacf66de8ce94960357-3",
			Route: &apploadbalancer.Route_Http{
				Http: &apploadbalancer.HttpRoute{
					Match: &apploadbalancer.HttpRouteMatch{
						Path: &apploadbalancer.StringMatch{
							Match: &apploadbalancer.StringMatch_ExactMatch{ExactMatch: "/stagger"},
						},
					},
					Action: &apploadbalancer.HttpRoute_Redirect{
						Redirect: &apploadbalancer.RedirectAction{
							ReplaceScheme: "https",
							ReplacePort:   443,
							Path:          nil,
							RemoveQuery:   false,
							ResponseCode:  apploadbalancer.RedirectAction_MOVED_PERMANENTLY,
						},
					},
				},
			},
		}
	)
	testData := []struct {
		desc          string
		rules         []*networking.IngressRule
		redirectRules []*networking.IngressRule
		routeOpts     RouteResolveOpts
		vhOpts        VirtualHostResolveOpts
		exp           HTTPRouterData
	}{
		{
			desc:          "OK",
			rules:         []*networking.IngressRule{rule1, rule2, rule3},
			redirectRules: []*networking.IngressRule{rule4},
			exp: HTTPRouterData{
				Router: &apploadbalancer.HttpRouter{
					Name:        "httprouter-07544a934fcd54e50ab30eacf66de8ce94960357",
					Description: "router for k8s ingress with tag: " + tag,
					FolderId:    folderID,
					Labels:      map[string]string{"": clusterID, "system": "yc-alb-ingress"},
					VirtualHosts: []*apploadbalancer.VirtualHost{
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-0",
							Authority: []string{"example1.com"},
							Routes:    []*apploadbalancer.Route{route0, route1, route3},
						},
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-1",
							Authority: []string{"example2.com"},
							Routes:    []*apploadbalancer.Route{route2},
						},
					},
				},
			},
		},
		{
			desc:          "OK REGEX",
			rules:         []*networking.IngressRule{rule1, rule3},
			redirectRules: []*networking.IngressRule{rule4},
			routeOpts:     RouteResolveOpts{UseRegex: true},
			exp: HTTPRouterData{
				Router: &apploadbalancer.HttpRouter{
					Name:        "httprouter-07544a934fcd54e50ab30eacf66de8ce94960357",
					Description: "router for k8s ingress with tag: " + tag,
					FolderId:    folderID,
					Labels:      map[string]string{"": clusterID, "system": "yc-alb-ingress"},
					VirtualHosts: []*apploadbalancer.VirtualHost{
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-0",
							Authority: []string{"example1.com"},
							Routes:    []*apploadbalancer.Route{regexroute0, regexroute2},
						},
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-1",
							Authority: []string{"example2.com"},
							Routes:    []*apploadbalancer.Route{regexroute1},
						},
					},
				},
			},
		},
		{
			desc:          "OK GRPC",
			rules:         []*networking.IngressRule{rule1, rule2, rule3},
			redirectRules: []*networking.IngressRule{rule4},
			routeOpts:     RouteResolveOpts{BackendType: GRPC},
			exp: HTTPRouterData{
				Router: &apploadbalancer.HttpRouter{
					Name:        "httprouter-07544a934fcd54e50ab30eacf66de8ce94960357",
					Description: "router for k8s ingress with tag: " + tag,
					FolderId:    folderID,
					Labels:      map[string]string{"": clusterID, "system": "yc-alb-ingress"},
					VirtualHosts: []*apploadbalancer.VirtualHost{
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-0",
							Authority: []string{"example1.com"},
							Routes:    []*apploadbalancer.Route{routeGRPC0, routeGRPC1, routeGRPC3},
						},
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-1",
							Authority: []string{"example2.com"},
							Routes:    []*apploadbalancer.Route{routeGRPC2},
						},
					},
				},
			},
		},
		{
			desc:  "OK Modify headers",
			rules: []*networking.IngressRule{rule1, rule2},
			vhOpts: VirtualHostResolveOpts{
				ModifyResponse: ModifyHeaderOpts{
					Append: map[string]string{
						"toAppend": "append",
					},
					Replace: map[string]string{
						"toReplace":    "replace",
						"toReplaceTwo": "replace_two",
					},
					Rename: map[string]string{
						"toRename": "rename",
					},
					Remove: map[string]bool{
						"toRemove":    true,
						"notToRemove": false,
					},
				},
				ModifyRequest: ModifyHeaderOpts{
					Append: map[string]string{
						"toAppend": "append",
					},
					Replace: map[string]string{
						"toReplace":    "replace",
						"toReplaceTwo": "replace_two",
					},
					Rename: map[string]string{
						"toRename": "rename",
					},
					Remove: map[string]bool{
						"toRemove":    true,
						"notToRemove": false,
					},
				},
			},
			exp: HTTPRouterData{
				Router: &apploadbalancer.HttpRouter{
					Name:        "httprouter-07544a934fcd54e50ab30eacf66de8ce94960357",
					Description: "router for k8s ingress with tag: " + tag,
					FolderId:    folderID,
					Labels:      map[string]string{"": clusterID, "system": "yc-alb-ingress"},
					VirtualHosts: []*apploadbalancer.VirtualHost{
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-0",
							Authority: []string{"example1.com"},
							Routes:    []*apploadbalancer.Route{route0, route1},
							RouteOptions: &apploadbalancer.RouteOptions{
								ModifyResponseHeaders: []*apploadbalancer.HeaderModification{
									{
										Name: "toRemove",
										Operation: &apploadbalancer.HeaderModification_Remove{
											Remove: true,
										},
									},
									{
										Name: "notToRemove",
										Operation: &apploadbalancer.HeaderModification_Remove{
											Remove: false,
										},
									},
									{
										Name: "toReplace",
										Operation: &apploadbalancer.HeaderModification_Replace{
											Replace: "replace",
										},
									},
									{
										Name: "toReplaceTwo",
										Operation: &apploadbalancer.HeaderModification_Replace{
											Replace: "replace_two",
										},
									},
									{
										Name: "toRename",
										Operation: &apploadbalancer.HeaderModification_Rename{
											Rename: "rename",
										},
									},
									{
										Name: "toAppend",
										Operation: &apploadbalancer.HeaderModification_Append{
											Append: "append",
										},
									},
								},
								ModifyRequestHeaders: []*apploadbalancer.HeaderModification{
									{
										Name: "toRemove",
										Operation: &apploadbalancer.HeaderModification_Remove{
											Remove: true,
										},
									},
									{
										Name: "notToRemove",
										Operation: &apploadbalancer.HeaderModification_Remove{
											Remove: false,
										},
									},
									{
										Name: "toReplace",
										Operation: &apploadbalancer.HeaderModification_Replace{
											Replace: "replace",
										},
									},
									{
										Name: "toReplaceTwo",
										Operation: &apploadbalancer.HeaderModification_Replace{
											Replace: "replace_two",
										},
									},
									{
										Name: "toRename",
										Operation: &apploadbalancer.HeaderModification_Rename{
											Rename: "rename",
										},
									},
									{
										Name: "toAppend",
										Operation: &apploadbalancer.HeaderModification_Append{
											Append: "append",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			// Even with same host/path
			desc:          "redirect add to route action",
			rules:         []*networking.IngressRule{rule1},
			redirectRules: []*networking.IngressRule{rule1},
			exp: HTTPRouterData{
				Router: &apploadbalancer.HttpRouter{
					Name:        "httprouter-07544a934fcd54e50ab30eacf66de8ce94960357",
					Description: "router for k8s ingress with tag: " + tag,
					FolderId:    folderID,
					Labels:      map[string]string{"": clusterID, "system": "yc-alb-ingress"},
					VirtualHosts: []*apploadbalancer.VirtualHost{
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-0",
							Authority: []string{"example1.com"},
							Routes:    []*apploadbalancer.Route{route0, routeRedirect0},
						},
					},
				},
			},
		},
		{
			desc:  "action for same host and path doesn't overwrite when path type different",
			rules: []*networking.IngressRule{rule1, rule1OverwriteForPrefix},
			exp: HTTPRouterData{
				Router: &apploadbalancer.HttpRouter{
					Name:        "httprouter-07544a934fcd54e50ab30eacf66de8ce94960357",
					Description: "router for k8s ingress with tag: " + tag,
					FolderId:    folderID,
					Labels:      map[string]string{"": clusterID, "system": "yc-alb-ingress"},
					VirtualHosts: []*apploadbalancer.VirtualHost{
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-0",
							Authority: []string{"example1.com"},
							Routes:    []*apploadbalancer.Route{route0, appendedRoute0},
						},
					},
				},
			},
		},
		{
			desc:  "action for same host and path doesn't overwriten when path type equal",
			rules: []*networking.IngressRule{rule1, rule1OverwriteForExact},
			exp: HTTPRouterData{
				Router: &apploadbalancer.HttpRouter{
					Name:        "httprouter-07544a934fcd54e50ab30eacf66de8ce94960357",
					Description: "router for k8s ingress with tag: " + tag,
					FolderId:    folderID,
					Labels:      map[string]string{"": clusterID, "system": "yc-alb-ingress"},
					VirtualHosts: []*apploadbalancer.VirtualHost{
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-0",
							Authority: []string{"example1.com"},
							Routes:    []*apploadbalancer.Route{route0, appendedRoute1},
						},
					},
				},
			},
		},
		{
			desc:  "empty path",
			rules: []*networking.IngressRule{emptyPathRule},
			exp: HTTPRouterData{
				Router: &apploadbalancer.HttpRouter{
					Name:        "httprouter-07544a934fcd54e50ab30eacf66de8ce94960357",
					Description: "router for k8s ingress with tag: " + tag,
					FolderId:    folderID,
					Labels:      map[string]string{"": clusterID, "system": "yc-alb-ingress"},
					VirtualHosts: []*apploadbalancer.VirtualHost{
						{
							Name:      "vh-07544a934fcd54e50ab30eacf66de8ce94960357-0",
							Authority: []string{"example1.com"},
							Routes:    []*apploadbalancer.Route{emptyPathRoute},
						},
					},
				},
			},
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.desc == "redirect doesn't overwrite route action" {
				tc.desc = "redirect doesn't overwrite route action"
			}
			ctrl := gomock.NewController(t)
			tgRepo := mocks.NewMockTargetGroupFinder(ctrl)
			f := NewFactory(folderID, "", &metadata.Names{ClusterID: clusterID}, &metadata.Labels{ClusterID: clusterID}, nil, tgRepo)

			bgFinder := mocks.NewMockBackendGroupFinder(ctrl)
			bgFinder.EXPECT().FindBackendGroup(gomock.Any(), gomock.Any()).Return(&apploadbalancer.BackendGroup{
				Id: "backend-group-id",
			}, nil).AnyTimes()

			f.RestartVirtualHostIDGenerator()
			b := f.HTTPRouterBuilder(tag, bgFinder)
			b.SetOpts(tc.vhOpts, tc.routeOpts, "ingress-namespace")
			for _, r := range tc.rules {
				for _, p := range r.HTTP.Paths {
					hp, err := HTTPIngressPathToHostAndPath(r.Host, p, tc.routeOpts.UseRegex)
					require.NoError(t, err)

					err = b.AddRoute(hp, p.Backend.Service.Name)
					require.NoError(t, err)
				}
			}
			for _, r := range tc.redirectRules {
				for _, p := range r.HTTP.Paths {
					hp, err := HTTPIngressPathToHostAndPath(r.Host, p, tc.routeOpts.UseRegex)
					require.NoError(t, err)

					err = b.AddRedirectToHTTPS(hp)
					require.NoError(t, err)
				}
			}
			d := b.Build()
			assert.Condition(t,
				func() bool {
					if len(tc.exp.Router.Labels) != len(d.Router.Labels) {
						return false
					}
					for i := range tc.exp.Router.Labels {
						if tc.exp.Router.Labels[i] != d.Router.Labels[i] {
							return false
						}
					}

					return proto.Equal(tc.exp.Router.RouteOptions, d.Router.RouteOptions) &&
						tc.exp.Router.Name == d.Router.Name &&
						tc.exp.Router.FolderId == d.Router.FolderId &&
						reflect.DeepEqual(tc.exp.Router.Labels, d.Router.Labels)
				},
				"router aren't equals\nexp %v\ngot %v", tc.exp.Router, d.Router,
			)

			require.Equal(t, len(tc.exp.Router.VirtualHosts), len(d.Router.GetVirtualHosts()))
			vhosts := make(map[string]int, len(tc.exp.Router.VirtualHosts))
			for i, exp := range tc.exp.Router.VirtualHosts {
				vhosts[exp.Name] = i
			}

			for _, act := range d.Router.VirtualHosts {
				i, ok := vhosts[act.Name]
				require.True(t, ok, "router contains vh %v, but isn't expected to contain it", act)
				comp := func() bool {
					exp := tc.exp.Router.VirtualHosts[i]

					if len(act.Routes) != len(exp.Routes) {
						return false
					}
					for i := range act.Routes {
						if !routesEqualExceptName(act.Routes[i], exp.Routes[i]) {
							return false
						}
					}

					if exp.RouteOptions != nil {
						if algo.ContainSameElements(act.RouteOptions.ModifyResponseHeaders, exp.RouteOptions.ModifyResponseHeaders) &&
							algo.ContainSameElements(act.RouteOptions.ModifyRequestHeaders, exp.RouteOptions.ModifyRequestHeaders) {
							return false
						}
					}

					return exp.Name == act.Name &&
						reflect.DeepEqual(act.Authority, exp.Authority)
				}
				assert.Condition(t, comp, "virtual hosts mismatch at position %d\nexp %v\ngot %v", i, tc.exp.Router.VirtualHosts[i], act)
			}
		},
		)
	}
}

func routesEqualExceptName(lhs, rhs *apploadbalancer.Route) bool {
	cp := proto.Clone(lhs).(*apploadbalancer.Route)
	cp.Name = rhs.Name
	return proto.Equal(cp, rhs)
}
