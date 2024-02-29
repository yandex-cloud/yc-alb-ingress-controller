package reconcile

import (
	"context"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	sdkoperation "github.com/yandex-cloud/go-sdk/operation"

	"github.com/yandex-cloud/alb-ingress/pkg/builders"
	"github.com/yandex-cloud/alb-ingress/pkg/deploy"
	ycerrors "github.com/yandex-cloud/alb-ingress/pkg/errors"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
)

type IngressGroupEngine struct {
	*builders.Data
	Repo       Repository
	Predicates UpdatePredicates
	Names      *metadata.Names
}

func (r *IngressGroupEngine) ReconcileHTTPRouter(router *apploadbalancer.HttpRouter) (*deploy.ReconciledHTTPRouter, error) {
	var hostData *builders.VirtualHostData
	if r.Data != nil {
		hostData = r.HTTPHosts
	}
	ret, err := reconcileHTTPRouter(r.Repo, router, hostData, r.Predicates)
	if err != nil {
		return nil, err
	}

	if ret.Active != nil {
		r.InjectRouterIDIntoHandler(ret.Active.Id)
	}

	return ret, nil
}

func (r *IngressGroupEngine) ReconcileTLSRouter(router *apploadbalancer.HttpRouter) (*deploy.ReconciledHTTPRouter, error) {
	var hostData *builders.VirtualHostData
	if r.Data != nil {
		hostData = r.TLSHosts
	}
	ret, err := reconcileHTTPRouter(r.Repo, router, hostData, r.Predicates)
	if err != nil {
		return nil, err
	}

	if ret.Active != nil {
		r.InjectTLSRouterIDIntoSNIMatches(ret.Active.Id)
	}

	return ret, nil
}

func (r IngressGroupEngine) ReconcileBalancer(balancer *apploadbalancer.LoadBalancer) (*deploy.ReconciledBalancer, error) {
	if r.Data == nil || (r.HTTPHosts == nil || len(r.HTTPHosts.HTTPRouteMap) == 0) && (r.TLSHosts == nil || len(r.TLSHosts.HTTPRouteMap) == 0) { // assume no routes means no ingresses -> delete
		if balancer.GetStatus() == apploadbalancer.LoadBalancer_DELETING {
			return nil, ycerrors.YCResourceNotReadyError{
				ResourceType: "ALB",
				Name:         balancer.Name,
			}
		}
		return &deploy.ReconciledBalancer{Garbage: balancer}, nil
	}

	if balancer == nil { //create
		op, err := r.Repo.CreateLoadBalancer(context.Background(), r.Data.Balancer)
		if err != nil {
			return nil, err
		}
		return nil, ycerrors.OperationIncompleteError{ID: op.Id}
	}

	// TODO: consider re-creating balancer if balancer.NetworkID != b.NetworkID
	// TODO: flexible update mask

	if balancer.Status != apploadbalancer.LoadBalancer_ACTIVE {
		return nil, ycerrors.YCResourceNotReadyError{
			ResourceType: "ALB",
			Name:         balancer.Name,
		}
	}

	r.Data.Balancer.Id = balancer.Id
	if r.Predicates.BalancerNeedsUpdate(balancer, r.Data.Balancer) {
		op, err := r.Repo.UpdateLoadBalancer(context.Background(), r.Data.Balancer)
		if err != nil {
			return nil, err
		}
		return nil, ycerrors.OperationIncompleteError{ID: op.Id}
	}
	return &deploy.ReconciledBalancer{Active: balancer}, nil
}

func reconcileHTTPRouter(repo Repository, currentRouter *apploadbalancer.HttpRouter, d *builders.VirtualHostData, predicates UpdatePredicates) (*deploy.ReconciledHTTPRouter, error) {
	if d == nil || d.Router == nil || len(d.Router.VirtualHosts) == 0 {
		return &deploy.ReconciledHTTPRouter{Garbage: currentRouter}, nil
	}
	if currentRouter == nil {
		op, err := repo.CreateHTTPRouter(context.Background(), d.Router)
		if err != nil {
			return nil, err
		}
		protoMsg, _ := sdkoperation.UnmarshalAny(op.Metadata)
		d.Router.Id = protoMsg.(*apploadbalancer.CreateHttpRouterMetadata).HttpRouterId
		return &deploy.ReconciledHTTPRouter{Active: d.Router}, nil
	}

	d.Router.Id = currentRouter.Id
	if predicates.RouterNeedsUpdate(currentRouter, d.Router) {
		_, err := repo.UpdateHTTPRouter(context.Background(), d.Router)
		if err != nil {
			return nil, err
		}
	}
	return &deploy.ReconciledHTTPRouter{Active: currentRouter}, nil
}
