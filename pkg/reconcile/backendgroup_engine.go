package reconcile

import (
	"context"
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	sdkoperation "github.com/yandex-cloud/go-sdk/operation"

	"github.com/yandex-cloud/alb-ingress/pkg/builders"
	"github.com/yandex-cloud/alb-ingress/pkg/deploy"
	ycerrors "github.com/yandex-cloud/alb-ingress/pkg/errors"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
)

var _ deploy.BackendGroupsReconcileEngine = &BackendGroupEngine{}

type BackendGroupEngine struct {
	*builders.Data
	Repo       backendGroupRepository
	Predicates backendGroupUpdatePredicate
	Names      *metadata.Names
}

func (r *BackendGroupEngine) ReconcileBackendGroup(ctx context.Context, bg *apploadbalancer.BackendGroup) (*deploy.ReconciledBackendGroups, error) {
	if r.Data == nil || r.Data.BackendGroups == nil {
		if bg == nil {
			return &deploy.ReconciledBackendGroups{}, nil
		}
		return &deploy.ReconciledBackendGroups{Garbage: []*apploadbalancer.BackendGroup{bg}}, nil
	}
	exp := r.Data.BackendGroups.BackendGroups[0]
	if bg == nil {
		op, err := r.Repo.CreateBackendGroup(ctx, exp)
		if err != nil {
			return nil, err
		}
		protoMsg, _ := sdkoperation.UnmarshalAny(op.Metadata)
		exp.Id = protoMsg.(*apploadbalancer.CreateBackendGroupMetadata).BackendGroupId
		return &deploy.ReconciledBackendGroups{Active: []*apploadbalancer.BackendGroup{exp}}, nil
	}

	exp.Id = bg.Id
	ops, err := r.Repo.ListBackendGroupOperations(ctx, bg)
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile backendgroup: %w", err)
	}
	if len(ops) != 0 {
		return nil, ycerrors.OperationIncompleteError{ID: ops[0].Id}
	}

	if r.Predicates.BackendGroupNeedsUpdate(bg, exp) {
		_, err := r.Repo.UpdateBackendGroup(context.Background(), exp)
		if err != nil {
			return nil, err
		}
	}
	return &deploy.ReconciledBackendGroups{Active: []*apploadbalancer.BackendGroup{exp}}, nil
}
