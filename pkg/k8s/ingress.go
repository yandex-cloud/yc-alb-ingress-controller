package k8s

import (
	"context"

	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngressLoader interface {
	List(ctx context.Context, opts ...client.ListOption) ([]networking.Ingress, error)

	ListBySvc(ctx context.Context, svc core.Service) ([]networking.Ingress, error)
}

type ingressLoader struct {
	cli client.Client
}

func NewIngressLoader(cli client.Client) IngressLoader {
	return &ingressLoader{cli: cli}
}

func (l *ingressLoader) List(ctx context.Context, opts ...client.ListOption) ([]networking.Ingress, error) {
	var ingList networking.IngressList
	err := l.cli.List(ctx, &ingList, opts...)
	if err != nil {
		return nil, err
	}

	var classList networking.IngressClassList
	err = l.cli.List(ctx, &classList)
	if err != nil {
		return nil, err
	}

	result := make([]networking.Ingress, 0)
	for _, item := range ingList.Items {
		managed := IsIngressManagedByThisController(item, classList)
		deleted := !item.GetDeletionTimestamp().IsZero()

		if managed && !deleted {
			result = append(result, item)
			continue
		}
	}

	return result, nil
}

func (l *ingressLoader) ListBySvc(ctx context.Context, svc core.Service) ([]networking.Ingress, error) {
	ings, err := l.List(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]networking.Ingress, 0)
	for _, ing := range ings {
		if IsServiceReferencedByIngress(svc, ing) {
			res = append(res, ing)
		}
	}
	return res, nil
}
