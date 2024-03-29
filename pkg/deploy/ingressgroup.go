package deploy

import (
	"context"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"

	"github.com/yandex-cloud/alb-ingress/pkg/yc"
)

// TODO: data structures below should be in some other package
// deps defined as interfaces use these data structures and this results in an import cycle in tests after direct mock
// generation (i.e. mockgen -destination=./mocks/ingressgroup.go -package=mocks . ReconcileEngine,ResourceFinder),
// which is a sign of incorrect type placement
type (
	ReconciledHTTPRouter struct {
		Active  *apploadbalancer.HttpRouter
		Garbage *apploadbalancer.HttpRouter
	}

	ReconciledBalancer struct {
		Active  *apploadbalancer.LoadBalancer
		Garbage *apploadbalancer.LoadBalancer
	}
)

type ReconcileEngine interface {
	ReconcileHTTPRouter(*apploadbalancer.HttpRouter) (*ReconciledHTTPRouter, error)
	ReconcileTLSRouter(*apploadbalancer.HttpRouter) (*ReconciledHTTPRouter, error)
	ReconcileBalancer(*apploadbalancer.LoadBalancer) (*ReconciledBalancer, error)
}

type ResourceFinder interface {
	FindAllResources(ctx context.Context, tag string) (*yc.BalancerResources, error)
	DeleteAllResources(background context.Context, b *yc.BalancerResources) error

	// FindBackendGroups and DeleteBackendGroups are needed to remove backend groups from old schema
	// TODO: remove when majority of users use newer version of controller
	FindBackendGroups(ctx context.Context, tag string) ([]*apploadbalancer.BackendGroup, error)
	DeleteBackendGroups(ctx context.Context, groups []*apploadbalancer.BackendGroup) error
}

type IngressGroupDeployManager struct {
	repo ResourceFinder
}

func NewIngressGroupDeployManager(repo ResourceFinder) *IngressGroupDeployManager {
	return &IngressGroupDeployManager{
		repo: repo,
	}
}

func (m *IngressGroupDeployManager) Deploy(ctx context.Context, tag string, engine ReconcileEngine) (yc.BalancerResources, error) {
	resources, err := m.repo.FindAllResources(ctx, tag)
	if err != nil {
		return yc.BalancerResources{}, err
	}

	httpRouter, err := engine.ReconcileHTTPRouter(resources.Router)
	if err != nil {
		return yc.BalancerResources{}, err
	}
	tlsRouter, err := engine.ReconcileTLSRouter(resources.TLSRouter)
	if err != nil {
		return yc.BalancerResources{}, err
	}
	balancer, err := engine.ReconcileBalancer(resources.Balancer)
	if err != nil {
		return yc.BalancerResources{}, err
	}

	err = m.repo.DeleteAllResources(context.Background(), &yc.BalancerResources{
		Balancer:  balancer.Garbage,
		Router:    httpRouter.Garbage,
		TLSRouter: tlsRouter.Garbage,
	})

	if err != nil {
		return yc.BalancerResources{}, err
	}
	return yc.BalancerResources{
		Balancer:  balancer.Active,
		TLSRouter: tlsRouter.Active,
		Router:    httpRouter.Active,
	}, nil
}

func (m *IngressGroupDeployManager) UndeployOldBG(ctx context.Context, tag string) error {
	bgs, err := m.repo.FindBackendGroups(ctx, tag)
	if err != nil {
		return err
	}

	return m.repo.DeleteBackendGroups(ctx, bgs)
}
