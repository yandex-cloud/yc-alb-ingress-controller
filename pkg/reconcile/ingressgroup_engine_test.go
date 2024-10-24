package reconcile

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	protooperation "github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	networking "k8s.io/api/networking/v1"

	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/builders"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/deploy"
	errors2 "github.com/yandex-cloud/yc-alb-ingress-controller/pkg/errors"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/reconcile/mocks"
)

type backendGroupsFixture struct {
	expbg1, expbg2, expbg3, bg2, bg3, bg4 *apploadbalancer.BackendGroup
}

type actionsFixture struct {
	httpAction1, tlsAction1, tlsAction2 *apploadbalancer.HttpRouteAction
}

type routesFixture struct {
	httpRoute1, httpRedirectRoute2, httpRedirectRoute3, tlsRoute1, tlsRoute2 *apploadbalancer.Route
}

type handlerFixture struct {
	httpHandler, tlsHandler1, tlsHandler2 *apploadbalancer.HttpHandler
	sniMatches                            []*apploadbalancer.SniMatch
}

type fixture struct {
	backendGroupsFixture
	actionsFixture
	routesFixture
	handlerFixture
}

func newFixture() *fixture {
	f := fixture{}
	f.expbg1 = &apploadbalancer.BackendGroup{Name: "backend_group_1"}
	f.expbg2 = &apploadbalancer.BackendGroup{Name: "backend_group_2"}
	f.expbg3 = &apploadbalancer.BackendGroup{Name: "backend_group_3"}
	f.bg2 = &apploadbalancer.BackendGroup{Id: "BG_2", Name: "backend_group_2"}
	f.bg3 = &apploadbalancer.BackendGroup{Id: "BG_3", Name: "backend_group_3"}
	f.bg4 = &apploadbalancer.BackendGroup{Id: "BG_4", Name: "backend_group_4"}

	f.httpAction1 = &apploadbalancer.HttpRouteAction{}
	f.tlsAction1 = &apploadbalancer.HttpRouteAction{}
	f.tlsAction2 = &apploadbalancer.HttpRouteAction{}

	f.httpRoute1 = &apploadbalancer.Route{
		Name:  "http_route_1",
		Route: &apploadbalancer.Route_Http{Http: &apploadbalancer.HttpRoute{Action: &apploadbalancer.HttpRoute_Route{Route: f.httpAction1}}},
	}
	f.httpRedirectRoute2 = &apploadbalancer.Route{
		Name:  "http_route_2",
		Route: &apploadbalancer.Route_Http{Http: &apploadbalancer.HttpRoute{Action: &apploadbalancer.HttpRoute_Redirect{Redirect: &apploadbalancer.RedirectAction{}}}},
	}
	f.httpRedirectRoute3 = &apploadbalancer.Route{
		Name:  "http_route_3",
		Route: &apploadbalancer.Route_Http{Http: &apploadbalancer.HttpRoute{Action: &apploadbalancer.HttpRoute_Redirect{Redirect: &apploadbalancer.RedirectAction{}}}},
	}
	f.tlsRoute1 = &apploadbalancer.Route{
		Name:  "tls_route_1",
		Route: &apploadbalancer.Route_Http{Http: &apploadbalancer.HttpRoute{Action: &apploadbalancer.HttpRoute_Route{Route: f.tlsAction1}}},
	}
	f.tlsRoute2 = &apploadbalancer.Route{
		Name:  "tls_route_2",
		Route: &apploadbalancer.Route_Http{Http: &apploadbalancer.HttpRoute{Action: &apploadbalancer.HttpRoute_Route{Route: f.tlsAction2}}},
	}
	f.httpHandler = &apploadbalancer.HttpHandler{}
	f.tlsHandler1 = &apploadbalancer.HttpHandler{}
	f.tlsHandler2 = &apploadbalancer.HttpHandler{}

	f.sniMatches = []*apploadbalancer.SniMatch{
		{
			ServerNames: []string{"anywhere.ru"},
			Handler: &apploadbalancer.TlsHandler{
				Handler:        &apploadbalancer.TlsHandler_HttpHandler{HttpHandler: f.tlsHandler1},
				CertificateIds: []string{"cert1"},
			},
		},
		{
			ServerNames: []string{"elsewhere.ru"},
			Handler: &apploadbalancer.TlsHandler{
				Handler:        &apploadbalancer.TlsHandler_HttpHandler{HttpHandler: f.tlsHandler2},
				CertificateIds: []string{"cert1"},
			},
		},
	}
	return &f
}

func data(f *fixture) *builders.Data {
	data := builders.Data{
		TargetGroups: []*apploadbalancer.TargetGroup{{Id: "TG_1"}},
		BackendGroups: &builders.BackendGroups{
			BackendGroups: []*apploadbalancer.BackendGroup{f.expbg1, f.expbg2, f.expbg3},
			BackendGroupByHostPath: map[builders.HostAndPath]*apploadbalancer.BackendGroup{
				{Host: "anywhere.ru", Path: "/go", PathType: string(networking.PathTypePrefix)}:     f.expbg1,
				{Host: "anywhere.ru", Path: "/wander", PathType: string(networking.PathTypePrefix)}: f.expbg2,
				{Host: "elsewhere.ru", Path: "/go", PathType: string(networking.PathTypePrefix)}:    f.expbg3,
			},
			BackendGroupByName: map[string]*apploadbalancer.BackendGroup{
				"backend_group_1": f.expbg1,
				"backend_group_2": f.expbg2,
				"backend_group_3": f.expbg3,
			},
		},
		HTTPRouter: &builders.HTTPRouterData{
			Router: &apploadbalancer.HttpRouter{
				VirtualHosts: []*apploadbalancer.VirtualHost{
					{
						Authority: []string{"anywhere.ru"},
						Routes:    []*apploadbalancer.Route{f.httpRoute1, f.httpRedirectRoute2},
					},
					{
						Authority: []string{"elsewhere.ru"},
						Routes:    []*apploadbalancer.Route{f.httpRedirectRoute3},
					},
				},
			},
		},
		TLSRouter: &builders.HTTPRouterData{
			Router: &apploadbalancer.HttpRouter{
				VirtualHosts: []*apploadbalancer.VirtualHost{
					{
						Authority: []string{"anywhere.ru"},
						Routes:    []*apploadbalancer.Route{f.tlsRoute1},
					},
					{
						Authority: []string{"elsewhere.ru"},
						Routes:    []*apploadbalancer.Route{f.tlsRoute2},
					},
				},
			},
		},
		Handler:    f.httpHandler,
		SNIMatches: f.sniMatches,
		Balancer: &apploadbalancer.LoadBalancer{
			Id: "B_1",
		},
	}
	return &data
}

func fakeMeta(t *testing.T, msg proto.Message) *anypb.Any {
	a, err := anypb.New(msg)
	require.NoError(t, err)
	return a
}

func TestIngressGroupEngine_ReconcileBalancer(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		b := &apploadbalancer.LoadBalancer{Id: "B_1", Status: apploadbalancer.LoadBalancer_ACTIVE}

		ctrl := gomock.NewController(t)

		p := mocks.NewMockUpdatePredicates(ctrl)
		p.EXPECT().BalancerNeedsUpdate(b, d.Balancer).Return(true)

		repo := mocks.NewMockRepository(ctrl)
		repo.EXPECT().UpdateLoadBalancer(gomock.Any(), d.Balancer).Return(&protooperation.Operation{
			Id:       "OP_1",
			Metadata: fakeMeta(t, &apploadbalancer.UpdateLoadBalancerMetadata{LoadBalancerId: "B_1"}),
		}, nil)
		repo.EXPECT().ListLoadBalancerOperations(gomock.Any(), gomock.Any()).Return(nil, nil)

		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		_, err := r.ReconcileBalancer(context.Background(), b)
		require.Error(t, err, "ReconcileBalancer() error = %v)", err)
		assert.True(t, errors.As(err, &errors2.OperationIncompleteError{}), "wrong error type %T", err)
	})
	t.Run("update, not ready", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		repo := mocks.NewMockRepository(ctrl)
		b := &apploadbalancer.LoadBalancer{Id: "B_1", Status: apploadbalancer.LoadBalancer_CREATING}
		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		_, err := r.ReconcileBalancer(context.Background(), b)
		require.Error(t, err, "ReconcileBalancer() error = %v)", err)
		assert.True(t, errors.As(err, &errors2.YCResourceNotReadyError{}), "wrong error type %T", err)
	})
	t.Run("create", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		repo := mocks.NewMockRepository(ctrl)
		repo.EXPECT().CreateLoadBalancer(gomock.Any(), d.Balancer).Return(&protooperation.Operation{
			Id:       "OP_1",
			Metadata: fakeMeta(t, &apploadbalancer.UpdateLoadBalancerMetadata{LoadBalancerId: "B_1"}),
		}, nil)
		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		_, err := r.ReconcileBalancer(context.Background(), nil)
		require.Error(t, err, "ReconcileBalancer() error = %v)", err)
		assert.True(t, errors.As(err, &errors2.OperationIncompleteError{}), "wrong error type %T", err)
	})
	t.Run("delete", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		d.HTTPRouter, d.TLSRouter, d.Balancer = nil, nil, nil
		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		repo := mocks.NewMockRepository(ctrl)
		b := &apploadbalancer.LoadBalancer{Id: "B_1", Status: apploadbalancer.LoadBalancer_ACTIVE}
		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		ret, err := r.ReconcileBalancer(context.Background(), b)
		require.NoError(t, err, "ReconcileBalancer() error = %v)", err)
		assert.Equal(t, &deploy.ReconciledBalancer{
			Garbage: b,
		}, ret)
	})
	t.Run("delete, status deleting", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		d.HTTPRouter, d.TLSRouter, d.Balancer = nil, nil, nil
		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		repo := mocks.NewMockRepository(ctrl)
		b := &apploadbalancer.LoadBalancer{Id: "B_1", Status: apploadbalancer.LoadBalancer_DELETING}
		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		_, err := r.ReconcileBalancer(context.Background(), b)
		require.Error(t, err, "ReconcileBalancer() error = %v)", err)
		assert.True(t, errors.As(err, &errors2.YCResourceNotReadyError{}), "wrong error type %T", err)
	})
	t.Run("no changes", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		repo := mocks.NewMockRepository(ctrl)
		b := &apploadbalancer.LoadBalancer{Id: "B_1", Status: apploadbalancer.LoadBalancer_ACTIVE}
		p.EXPECT().BalancerNeedsUpdate(b, d.Balancer).Return(false)
		repo.EXPECT().ListLoadBalancerOperations(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		ret, err := r.ReconcileBalancer(context.Background(), b)
		require.NoError(t, err, "ReconcileBalancer() error = %v)", err)
		assert.Equal(t, &deploy.ReconciledBalancer{
			Active: b,
		}, ret)
	})
}

func TestIngressGroupEngine_ReconcileHTTPRouter(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		router := &apploadbalancer.HttpRouter{Id: "HTTP_R_1"}

		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		p.EXPECT().RouterNeedsUpdate(router, d.HTTPRouter.Router).Return(true)
		repo := mocks.NewMockRepository(ctrl)

		repo.EXPECT().UpdateHTTPRouter(gomock.Any(), d.HTTPRouter.Router).Return(&protooperation.Operation{
			Id:       "OP_1",
			Metadata: fakeMeta(t, &apploadbalancer.UpdateHttpRouterMetadata{HttpRouterId: "HTTP_R_1"}),
		}, nil)
		repo.EXPECT().ListHTTPRouterOperations(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		ret, err := r.ReconcileHTTPRouter(context.Background(), router)
		require.NoError(t, err, "ReconcileHttpRouter() error = %v)", err)
		assert.Equal(t, "HTTP_R_1", ret.Active.Id)

		assert.Equal(t, "HTTP_R_1", d.HTTPRouter.Router.Id)
		assert.Equal(t, "HTTP_R_1", f.httpHandler.HttpRouterId)
	})

	t.Run("create", func(t *testing.T) {
		f := newFixture()
		d := data(f)

		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		repo := mocks.NewMockRepository(ctrl)
		repo.EXPECT().CreateHTTPRouter(gomock.Any(), d.HTTPRouter.Router).Return(&protooperation.Operation{
			Id:       "OP_1",
			Metadata: fakeMeta(t, &apploadbalancer.CreateHttpRouterMetadata{HttpRouterId: "HTTP_R_1"}),
		}, nil)
		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}

		ret, err := r.ReconcileHTTPRouter(context.Background(), nil)
		require.NoError(t, err, "ReconcileHttpRouter() error = %v)", err)
		assert.Equal(t, "HTTP_R_1", ret.Active.Id)

		assert.Equal(t, "HTTP_R_1", d.HTTPRouter.Router.Id)
		assert.Equal(t, "HTTP_R_1", f.httpHandler.HttpRouterId)
	})

	t.Run("delete", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		d.HTTPRouter = nil
		router := &apploadbalancer.HttpRouter{Id: "HTTP_R_1"}

		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		repo := mocks.NewMockRepository(ctrl)
		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		ret, err := r.ReconcileHTTPRouter(context.Background(), router)
		require.NoError(t, err, "ReconcileHttpRouter() error = %v)", err)
		assert.Equal(t, &deploy.ReconciledHTTPRouter{Garbage: router}, ret)
	})

	t.Run("no changes", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		router := &apploadbalancer.HttpRouter{Id: "HTTP_R_1"}

		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		p.EXPECT().RouterNeedsUpdate(router, d.HTTPRouter.Router).Return(false)
		repo := mocks.NewMockRepository(ctrl)
		repo.EXPECT().ListHTTPRouterOperations(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}

		ret, err := r.ReconcileHTTPRouter(context.Background(), router)
		require.NoError(t, err, "ReconcileHttpRouter() error = %v)", err)
		assert.Equal(t, "HTTP_R_1", ret.Active.Id)

		assert.Equal(t, "HTTP_R_1", d.HTTPRouter.Router.Id)
		assert.Equal(t, "HTTP_R_1", f.httpHandler.HttpRouterId)
	})
}

func TestIngressGroupEngine_ReconcileTLSRouter(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		router := &apploadbalancer.HttpRouter{Id: "HTTP_R_1"}

		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		p.EXPECT().RouterNeedsUpdate(router, d.TLSRouter.Router).Return(true)

		repo := mocks.NewMockRepository(ctrl)
		repo.EXPECT().UpdateHTTPRouter(gomock.Any(), d.TLSRouter.Router).Return(&protooperation.Operation{
			Id:       "OP_1",
			Metadata: fakeMeta(t, &apploadbalancer.UpdateHttpRouterMetadata{HttpRouterId: "HTTP_R_1"}),
		}, nil)
		repo.EXPECT().ListHTTPRouterOperations(gomock.Any(), gomock.Any()).Return(nil, nil)

		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		ret, err := r.ReconcileTLSRouter(context.Background(), router)
		require.NoError(t, err, "ReconcileTLSRouter() error = %v)", err)
		assert.Equal(t, "HTTP_R_1", ret.Active.Id)

		assert.Equal(t, "HTTP_R_1", d.TLSRouter.Router.Id)
		assert.Equal(t, "HTTP_R_1", f.tlsHandler1.HttpRouterId)
		assert.Equal(t, "HTTP_R_1", f.tlsHandler2.HttpRouterId)
	})

	t.Run("create", func(t *testing.T) {
		f := newFixture()
		d := data(f)

		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		repo := mocks.NewMockRepository(ctrl)
		repo.EXPECT().CreateHTTPRouter(gomock.Any(), d.TLSRouter.Router).Return(&protooperation.Operation{
			Id:       "OP_1",
			Metadata: fakeMeta(t, &apploadbalancer.CreateHttpRouterMetadata{HttpRouterId: "HTTP_R_1"}),
		}, nil)
		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		ret, err := r.ReconcileTLSRouter(context.Background(), nil)
		require.NoError(t, err, "ReconcileTLSRouter() error = %v)", err)
		assert.Equal(t, "HTTP_R_1", ret.Active.Id)

		assert.Equal(t, "HTTP_R_1", d.TLSRouter.Router.Id)
		assert.Equal(t, "HTTP_R_1", f.tlsHandler1.HttpRouterId)
		assert.Equal(t, "HTTP_R_1", f.tlsHandler2.HttpRouterId)
	})

	t.Run("delete", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		d.TLSRouter = nil
		router := &apploadbalancer.HttpRouter{Id: "HTTP_R_1"}

		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		repo := mocks.NewMockRepository(ctrl)
		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		ret, err := r.ReconcileTLSRouter(context.Background(), router)
		require.NoError(t, err, "ReconcileTLSRouter() error = %v)", err)
		assert.Equal(t, &deploy.ReconciledHTTPRouter{Garbage: router}, ret)
	})

	t.Run("no changes", func(t *testing.T) {
		f := newFixture()
		d := data(f)
		router := &apploadbalancer.HttpRouter{Id: "HTTP_R_1"}

		ctrl := gomock.NewController(t)
		p := mocks.NewMockUpdatePredicates(ctrl)
		p.EXPECT().RouterNeedsUpdate(router, d.TLSRouter.Router).Return(false)

		repo := mocks.NewMockRepository(ctrl)
		repo.EXPECT().ListHTTPRouterOperations(gomock.Any(), gomock.Any()).Return(nil, nil)

		r := &IngressGroupEngine{
			Data:       d,
			Repo:       repo,
			Predicates: p,
		}
		ret, err := r.ReconcileTLSRouter(context.Background(), router)
		require.NoError(t, err, "ReconcileTLSRouter() error = %v)", err)
		assert.Equal(t, "HTTP_R_1", ret.Active.Id)

		assert.Equal(t, "HTTP_R_1", d.TLSRouter.Router.Id)
		assert.Equal(t, "HTTP_R_1", f.tlsHandler1.HttpRouterId)
		assert.Equal(t, "HTTP_R_1", f.tlsHandler2.HttpRouterId)
	})
}
