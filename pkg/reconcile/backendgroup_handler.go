package reconcile

import (
	"context"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	"github.com/yandex-cloud/alb-ingress/pkg/deploy"
	ycerrors "github.com/yandex-cloud/alb-ingress/pkg/errors"
	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
)

// TODO: move where it can exported, or refactor FinalizerManager to have it as a field
const finalizer = "ingress.ya.ru/final"

type Builder interface {
	Build(g *v1alpha1.HttpBackendGroup) (*apploadbalancer.BackendGroup, error)
}

type BackendGroupRepo interface {
	FindBackendGroupByCR(ctx context.Context, ns, name string) (*apploadbalancer.BackendGroup, error)
	Repository
}

type BackendGroupEngineBuilder interface {
	Build(crd *v1alpha1.HttpBackendGroup) (*BackendGroupEngine, error)
}

type Deployer interface {
	Deploy(ctx context.Context, name string, re deploy.BackendGroupsReconcileEngine) (*apploadbalancer.BackendGroup, error)
}

type BackendGroupReconcileHandler struct {
	Builder          BackendGroupEngineBuilder
	Deployer         Deployer
	Repo             BackendGroupRepo
	Predicates       UpdatePredicates
	FinalizerManager *k8s.FinalizerManager

	Names *metadata.Names
}

func (b *BackendGroupReconcileHandler) HandleResourceUpdated(ctx context.Context, o client.Object) error {
	err := b.FinalizerManager.UpdateFinalizer(ctx, o, finalizer)
	if err != nil {
		return err
	}
	engine, err := b.Builder.Build(o.(*v1alpha1.HttpBackendGroup))
	if err != nil {
		return err
	}
	_, err = b.Deployer.Deploy(ctx, b.Names.BackendGroupForCR(o.GetNamespace(), o.GetName()), engine)
	if err != nil {
		return err
	}
	return nil
}

func (b *BackendGroupReconcileHandler) HandleResourceDeleted(ctx context.Context, o client.Object) error {
	bg, err := b.Repo.FindBackendGroupByCR(ctx, o.GetNamespace(), o.GetName())
	if err != nil {
		return err
	}
	if bg == nil {
		return b.FinalizerManager.RemoveFinalizer(ctx, o, finalizer)
	}
	op, err := b.Repo.DeleteBackendGroup(ctx, bg)
	if err != nil {
		return err
	}
	return ycerrors.OperationIncompleteError{ID: op.Id}
}

func (b *BackendGroupReconcileHandler) HandleResourceNotFound(_ context.Context, name types.NamespacedName) error {
	/*
		Solution1: if BackendGroup built from CRs is unambiguously named using its CRD name just delete it by name,
		and if it existed -> requeue, otherwise reconciliation not needed

		Solution2: if we cannot look up BackendGroups by their CRs, we need a mechanism of finding orphaned BackendGroups
	*/
	return nil
}
