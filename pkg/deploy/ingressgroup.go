package deploy

import (
	"context"
	"fmt"

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
	ReconcileHTTPRouter(context.Context, *apploadbalancer.HttpRouter) (*ReconciledHTTPRouter, error)
	ReconcileTLSRouter(context.Context, *apploadbalancer.HttpRouter) (*ReconciledHTTPRouter, error)
	ReconcileBalancer(context.Context, *apploadbalancer.LoadBalancer) (*ReconciledBalancer, error)
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
		return yc.BalancerResources{}, fmt.Errorf("failed to find resources: %w", err)
	}

	httpRouter, err := engine.ReconcileHTTPRouter(ctx, resources.Router)
	if err != nil {
		return yc.BalancerResources{}, fmt.Errorf("failed to reconcile http router: %w", err)
	}
	tlsRouter, err := engine.ReconcileTLSRouter(ctx, resources.TLSRouter)
	if err != nil {
		return yc.BalancerResources{}, fmt.Errorf("failed to reconcile tls router: %w", err)
	}
	balancer, err := engine.ReconcileBalancer(ctx, resources.Balancer)
	if err != nil {
		return yc.BalancerResources{}, fmt.Errorf("failed to reconcile balancer: %w", err)
	}

	err = m.repo.DeleteAllResources(ctx, &yc.BalancerResources{
		Balancer:  balancer.Garbage,
		Router:    httpRouter.Garbage,
		TLSRouter: tlsRouter.Garbage,
	})
	if err != nil {
		return yc.BalancerResources{}, fmt.Errorf("failed to delete resources: %w", err)
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
		return fmt.Errorf("failed to find backend groups: %w", err)
	}

	return m.repo.DeleteBackendGroups(ctx, bgs)
}
