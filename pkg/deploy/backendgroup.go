package deploy

import (
	"context"
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"

	ycerrors "github.com/yandex-cloud/alb-ingress/pkg/errors"
	"github.com/yandex-cloud/alb-ingress/pkg/yc"
)

type BackendGroupRepo interface {
	FindBackendGroup(ctx context.Context, name string) (*apploadbalancer.BackendGroup, error)
	CreateBackendGroup(ctx context.Context, group *apploadbalancer.BackendGroup) (*operation.Operation, error)
	UpdateBackendGroup(ctx context.Context, group *apploadbalancer.BackendGroup) (*operation.Operation, error)
	DeleteBackendGroup(context.Context, *apploadbalancer.BackendGroup) (*operation.Operation, error)
}

type ReconciledBackendGroups struct {
	Active  []*apploadbalancer.BackendGroup
	Garbage []*apploadbalancer.BackendGroup
}

type BackendGroupsReconcileEngine interface {
	ReconcileBackendGroup(*apploadbalancer.BackendGroup) (*ReconciledBackendGroups, error)
}

type BackendGroupDeployManager struct {
	Repo BackendGroupRepo
}

func NewBackendGroupDeployManager(repo BackendGroupRepo) *BackendGroupDeployManager {
	return &BackendGroupDeployManager{Repo: repo}
}

func (m BackendGroupDeployManager) Deploy(name string, engine BackendGroupsReconcileEngine) (*apploadbalancer.BackendGroup, error) {
	ctx := context.Background()
	bg, err := m.Repo.FindBackendGroup(ctx, name)
	if err != nil {
		return nil, err
	}
	bgs, err := engine.ReconcileBackendGroup(bg)
	if err != nil {
		return nil, err
	}
	return bgs.Active[0], nil
}

type BackendGroupDeployer struct {
	repo BackendGroupRepo

	predicates yc.UpdatePredicates
}

func NewBackendGroupDeployer(repo BackendGroupRepo) *BackendGroupDeployer {
	return &BackendGroupDeployer{repo: repo}
}

func (d *BackendGroupDeployer) Deploy(ctx context.Context, expected *apploadbalancer.BackendGroup) (*apploadbalancer.BackendGroup, error) {
	actual, err := d.repo.FindBackendGroup(ctx, expected.Name)
	if err != nil {
		return nil, err
	}

	// create if needed
	if actual == nil {
		op, err := d.repo.CreateBackendGroup(ctx, expected)
		if err != nil {
			return nil, fmt.Errorf("failed to create backend group: %w", err)
		}
		if op != nil {
			return nil, ycerrors.OperationIncompleteError{ID: op.Id}
		}
		return actual, nil
	}

	// update if needed
	if d.predicates.BackendGroupNeedsUpdate(expected, actual) {
		expected.Id = actual.Id
		op, err := d.repo.UpdateBackendGroup(ctx, expected)
		if err != nil {
			return nil, fmt.Errorf("failed to update backend group: %w", err)
		}
		if op != nil {
			return nil, ycerrors.OperationIncompleteError{ID: op.Id}
		}
		return actual, nil
	}

	return actual, nil
}

func (d *BackendGroupDeployer) Undeploy(ctx context.Context, name string) (*apploadbalancer.BackendGroup, error) {
	tg, err := d.repo.FindBackendGroup(ctx, name)
	if err != nil {
		return nil, err
	}

	if tg == nil {
		return nil, nil
	}

	// TODO(khodasevich): probably handle operation somehow
	_, err = d.repo.DeleteBackendGroup(ctx, tg)

	return tg, err
}
