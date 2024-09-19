package reconcile

import (
	"context"
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	ycerrors "github.com/yandex-cloud/alb-ingress/pkg/errors"
	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
)

type GrpcBackendGroupForCrdBuilder interface { // nolint:revive
	BuildForCrd(ctx context.Context, crd *v1alpha1.GrpcBackendGroup) (*apploadbalancer.BackendGroup, error)
}

type GrpcBackendGroupReconcileHandler struct { //nolint:revive
	Builder          GrpcBackendGroupForCrdBuilder
	Deployer         BackendGroupDeployer
	Repo             BackendGroupRepo
	Predicates       UpdatePredicates
	FinalizerManager *k8s.FinalizerManager

	Names *metadata.Names
}

func (b *GrpcBackendGroupReconcileHandler) HandleResourceUpdated(ctx context.Context, o client.Object) error {
	err := b.FinalizerManager.UpdateFinalizer(ctx, o, k8s.Finalizer)
	if err != nil {
		return fmt.Errorf("failed to update finalizer: %w", err)
	}

	hbg, err := b.Builder.BuildForCrd(ctx, o.(*v1alpha1.GrpcBackendGroup))
	if err != nil {
		return fmt.Errorf("failed to build backend group for crd: %w", err)
	}
	_, err = b.Deployer.Deploy(ctx, hbg)
	if err != nil {
		return fmt.Errorf("failed to deploy backend group: %w", err)
	}

	return nil
}

func (b *GrpcBackendGroupReconcileHandler) HandleResourceDeleted(ctx context.Context, o client.Object) error {
	bg, err := b.Repo.FindBackendGroupByCR(ctx, o.GetNamespace(), o.GetName())
	if err != nil {
		return fmt.Errorf("failed to find backend group by cr: %w", err)
	}
	if bg == nil {
		return b.FinalizerManager.RemoveFinalizer(ctx, o, k8s.Finalizer)
	}
	op, err := b.Repo.DeleteBackendGroup(ctx, bg)
	if err != nil {
		return fmt.Errorf("failed to delete backend group: %w", err)
	}
	return ycerrors.OperationIncompleteError{ID: op.Id}
}

func (b *GrpcBackendGroupReconcileHandler) HandleResourceNotFound(_ context.Context, _ types.NamespacedName) error {
	/*
		Solution1: if BackendGroup built from CRs is unambiguously named using its CRD name just delete it by name,
		and if it existed -> requeue, otherwise reconciliation not needed

		Solution2: if we cannot look up BackendGroups by their CRs, we need a mechanism of finding orphaned BackendGroups
	*/
	return nil
}
