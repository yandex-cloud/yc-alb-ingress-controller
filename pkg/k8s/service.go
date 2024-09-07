package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
)

type ServiceLoader interface {
	Load(context.Context, types.NamespacedName) (ServiceToReconcile, error)
}

type DefaultServiceLoader struct {
	Client client.Client
}

type ServiceToReconcile struct {
	ToReconcile *v1.Service
	ToDelete    *v1.Service
}

func (l *DefaultServiceLoader) Load(ctx context.Context, name types.NamespacedName) (ServiceToReconcile, error) {
	var svc v1.Service
	err := l.Client.Get(ctx, name, &svc)
	if errors.IsNotFound(err) {
		return ServiceToReconcile{}, nil
	}

	if err != nil {
		return ServiceToReconcile{}, err
	}

	deleted := svc.DeletionTimestamp != nil
	hasfinalizer := hasFinalizer(&svc, Finalizer)
	managed, err := IsServiceManaged(ctx, l.Client, svc)
	if err != nil {
		return ServiceToReconcile{}, err
	}

	if !managed && !hasfinalizer {
		return ServiceToReconcile{}, nil
	}

	if (!managed || deleted) && hasfinalizer {
		return ServiceToReconcile{ToDelete: &svc}, nil
	}

	return ServiceToReconcile{ToReconcile: &svc}, nil
}

func IsServiceReferencedByIngress(svc v1.Service, ing networking.Ingress) bool {
	if svc.Namespace != ing.Namespace {
		return false
	}

	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil &&
		ing.Spec.DefaultBackend.Service.Name == svc.Name {
		return true
	}

	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service != nil && path.Backend.Service.Name == svc.Name {
				return true
			}
		}
	}
	return false
}

func IsServiceManaged(ctx context.Context, cli client.Client, svc v1.Service) (bool, error) {
	managedIng, err := isServiceReferencedByIngress(ctx, cli, svc)
	if err != nil {
		return false, err
	}
	if managedIng {
		return true, nil
	}

	managedBG, err := isServiceReferencedByHTTPBackendGroup(ctx, cli, svc)
	if err != nil {
		return false, err
	}
	if managedBG {
		return true, nil
	}

	managedBG, err = isServiceReferencedByGRPCBackendGroup(ctx, cli, svc)
	if err != nil {
		return false, err
	}

	return managedBG, nil
}

func isServiceReferencedByHTTPBackendGroup(ctx context.Context, cli client.Client, svc v1.Service) (bool, error) {
	var bgs v1alpha1.HttpBackendGroupList
	err := cli.List(ctx, &bgs)
	if err != nil {
		return false, err
	}

	for _, bg := range bgs.Items {
		if bg.Namespace != svc.Namespace {
			continue
		}

		for _, be := range bg.Spec.Backends {
			if be.Service == nil {
				continue
			}

			if be.Service.Name == svc.Name {
				return true, nil
			}
		}
	}

	return false, nil
}

func isServiceReferencedByGRPCBackendGroup(ctx context.Context, cli client.Client, svc v1.Service) (bool, error) {
	var bgs v1alpha1.GrpcBackendGroupList
	err := cli.List(ctx, &bgs)
	if err != nil {
		return false, err
	}

	for _, bg := range bgs.Items {
		if bg.Namespace != svc.Namespace {
			continue
		}

		for _, be := range bg.Spec.Backends {
			if be.Service == nil {
				continue
			}

			if be.Service.Name == svc.Name {
				return true, nil
			}
		}
	}

	return false, nil
}

func isServiceReferencedByIngress(ctx context.Context, cli client.Client, svc v1.Service) (bool, error) {
	var ingList networking.IngressList
	err := cli.List(ctx, &ingList)
	if err != nil {
		return false, err
	}

	var ingClassList networking.IngressClassList
	err = cli.List(ctx, &ingClassList)
	if err != nil {
		return false, err
	}

	for _, ing := range ingList.Items {
		if !IsIngressManagedByThisController(ing, ingClassList) || ing.DeletionTimestamp != nil {
			continue
		}

		if IsServiceReferencedByIngress(svc, ing) {
			return true, nil
		}
	}

	return false, nil
}
