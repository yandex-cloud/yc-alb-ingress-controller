package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yandex-cloud/yc-alb-ingress-controller/api/v1alpha1"
)

type ServiceLoader interface {
	Load(context.Context, types.NamespacedName) (ServiceToReconcile, error)
}

type DefaultServiceLoader struct {
	Client client.Client
}

type ServiceToReconcile struct {
	References  map[string]IngressGroup
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
		return ServiceToReconcile{}, fmt.Errorf("failed to get service: %w", err)
	}

	deleted := svc.DeletionTimestamp != nil
	hasfinalizer := hasFinalizer(&svc, Finalizer)
	refs, err := getServiceIngressRefs(ctx, l.Client, svc)
	if err != nil {
		return ServiceToReconcile{}, fmt.Errorf("failed to get service ingress refs: %w", err)
	}

	managed := len(refs) > 0

	if !managed && !hasfinalizer {
		return ServiceToReconcile{}, nil
	}

	if (!managed || deleted) && hasfinalizer {
		return ServiceToReconcile{ToDelete: &svc, References: refs}, nil
	}

	return ServiceToReconcile{ToReconcile: &svc, References: refs}, nil
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

func getServiceIngressRefs(ctx context.Context, cli client.Client, svc v1.Service) (map[string]IngressGroup, error) {
	ings, err := getManagedIngresses(ctx, cli)
	if err != nil {
		return nil, fmt.Errorf("failed to get managed ingresses: %w", err)
	}

	refs := make(map[string]struct{})

	for _, ing := range ings {
		if IsServiceReferencedByIngress(svc, ing) {
			refs[ing.Annotations[AlbTag]] = struct{}{}
		}
	}

	httpBgRefs, err := getServiceHTTPBGRefs(ctx, cli, svc)
	if err != nil {
		return nil, fmt.Errorf("failed to get service http backend group refs: %w", err)
	}
	grpcBgRefs, err := getServiceGRPCBGRefs(ctx, cli, svc)
	if err != nil {
		return nil, fmt.Errorf("failed to get service grpc backend group refs: %w", err)
	}

	for _, ing := range ings {
		for _, bg := range httpBgRefs {
			if isBGReferencedByIngress(ing, "HttpBackendGroup", NamespacedNameOf(&bg)) {
				refs[ing.Annotations[AlbTag]] = struct{}{}
			}
		}
		for _, bg := range grpcBgRefs {
			if isBGReferencedByIngress(ing, "GrpcBackendGroup", NamespacedNameOf(&bg)) {
				refs[ing.Annotations[AlbTag]] = struct{}{}
			}
		}
	}

	groups := make(map[string]IngressGroup)
	for _, ing := range ings {
		tag := ing.Annotations[AlbTag]
		if _, ok := refs[tag]; ok {
			if _, ok := groups[tag]; !ok {
				groups[tag] = IngressGroup{
					Tag: tag,
				}
			}
			g := groups[tag]
			g.Items = append(g.Items, ing)
			groups[tag] = g
		}
	}

	return groups, nil
}

func getServiceHTTPBGRefs(ctx context.Context, cli client.Client, svc v1.Service) ([]v1alpha1.HttpBackendGroup, error) {
	var bgs v1alpha1.HttpBackendGroupList
	err := cli.List(ctx, &bgs)
	if err != nil {
		return nil, fmt.Errorf("failed to list http backend groups: %w", err)
	}

	var refs []v1alpha1.HttpBackendGroup
	for _, bg := range bgs.Items {
		if bg.Namespace != svc.Namespace {
			continue
		}

		for _, be := range bg.Spec.Backends {
			if be.Service == nil {
				continue
			}

			if be.Service.Name == svc.Name {
				refs = append(refs, bg)
			}
		}
	}

	return refs, nil
}

func getServiceGRPCBGRefs(ctx context.Context, cli client.Client, svc v1.Service) ([]v1alpha1.GrpcBackendGroup, error) {
	var bgs v1alpha1.GrpcBackendGroupList
	err := cli.List(ctx, &bgs)
	if err != nil {
		return nil, fmt.Errorf("failed to list grpc backend groups: %w", err)
	}

	var refs []v1alpha1.GrpcBackendGroup
	for _, bg := range bgs.Items {
		if bg.Namespace != svc.Namespace {
			continue
		}

		for _, be := range bg.Spec.Backends {
			if be.Service == nil {
				continue
			}

			if be.Service.Name == svc.Name {
				refs = append(refs, bg)
			}
		}
	}

	return refs, nil
}

func getManagedIngresses(ctx context.Context, cli client.Client) ([]networking.Ingress, error) {
	var ingList networking.IngressList
	err := cli.List(ctx, &ingList)
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	var ingClassList networking.IngressClassList
	err = cli.List(ctx, &ingClassList)
	if err != nil {
		return nil, fmt.Errorf("failed to list ingress classes: %w", err)
	}

	validIngs := make([]networking.Ingress, 0)
	for _, ing := range ingList.Items {
		if !IsIngressManagedByThisController(ing, ingClassList) || ing.DeletionTimestamp != nil {
			continue
		}

		validIngs = append(validIngs, ing)
	}

	return validIngs, nil
}

func isBGReferencedByIngress(ing networking.Ingress, bgKind string, bgName types.NamespacedName) bool {
	if ing.Namespace != bgName.Namespace {
		return false
	}

	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Resource != nil {
		res := ing.Spec.DefaultBackend.Resource
		if res.Kind == bgKind && res.Name == bgName.Name {
			return true
		}
	}

	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			if path.Backend.Resource == nil {
				continue
			}

			if path.Backend.Resource.Kind == bgKind && path.Backend.Resource.Name == bgName.Name {
				return true
			}
		}
	}

	return false
}
