package reconcile

import (
	"context"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	protooperation "github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
)

//go:generate mockgen -destination=./mocks/mocks.go -package=mocks . Repository,UpdatePredicates

type backendGroupRepository interface {
	FindBackendGroup(ctx context.Context, name string) (*apploadbalancer.BackendGroup, error)

	CreateBackendGroup(context.Context, *apploadbalancer.BackendGroup) (*protooperation.Operation, error)
	UpdateBackendGroup(context.Context, *apploadbalancer.BackendGroup) (*protooperation.Operation, error)
	DeleteBackendGroup(context.Context, *apploadbalancer.BackendGroup) (*protooperation.Operation, error)

	ListBackendGroupOperations(ctx context.Context, group *apploadbalancer.BackendGroup) ([]*protooperation.Operation, error)
}

type Repository interface {
	backendGroupRepository

	CreateHTTPRouter(context.Context, *apploadbalancer.HttpRouter) (*protooperation.Operation, error)
	UpdateHTTPRouter(context.Context, *apploadbalancer.HttpRouter) (*protooperation.Operation, error)
	DeleteHTTPRouter(context.Context, *apploadbalancer.HttpRouter) (*protooperation.Operation, error)
	ListHTTPRouterIncompleteOperations(ctx context.Context, router *apploadbalancer.HttpRouter) ([]*protooperation.Operation, error)

	CreateLoadBalancer(context.Context, *apploadbalancer.LoadBalancer) (*protooperation.Operation, error)
	UpdateLoadBalancer(context.Context, *apploadbalancer.LoadBalancer) (*protooperation.Operation, error)
	DeleteLoadBalancer(context.Context, *apploadbalancer.LoadBalancer) (*protooperation.Operation, error)
	ListLoadBalancerIncompleteOperations(ctx context.Context, balancer *apploadbalancer.LoadBalancer) ([]*protooperation.Operation, error)
}

type backendGroupUpdatePredicate interface {
	BackendGroupNeedsUpdate(g *apploadbalancer.BackendGroup, exp *apploadbalancer.BackendGroup) bool
}

type UpdatePredicates interface {
	backendGroupUpdatePredicate
	BalancerNeedsUpdate(balancer *apploadbalancer.LoadBalancer, exp *apploadbalancer.LoadBalancer) bool
	RouterNeedsUpdate(router *apploadbalancer.HttpRouter, exp *apploadbalancer.HttpRouter) bool
}
