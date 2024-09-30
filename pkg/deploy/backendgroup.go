package deploy

import (
	"context"
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"

	ycerrors "github.com/yandex-cloud/yc-alb-ingress-controller/pkg/errors"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/yc"
)

type BackendGroupRepo interface {
	FindBackendGroup(ctx context.Context, name string) (*apploadbalancer.BackendGroup, error)
	CreateBackendGroup(ctx context.Context, group *apploadbalancer.BackendGroup) (*operation.Operation, error)
	UpdateBackendGroup(ctx context.Context, group *apploadbalancer.BackendGroup) (*operation.Operation, error)
	DeleteBackendGroup(context.Context, *apploadbalancer.BackendGroup) (*operation.Operation, error)
	ListBackendGroupOperations(ctx context.Context, group *apploadbalancer.BackendGroup) ([]*operation.Operation, error)
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
		return nil, fmt.Errorf("failed to find backend group: %w", err)
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
	ops, err := d.repo.ListBackendGroupOperations(ctx, actual)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy backendgroup: %w", err)
	}
	if len(ops) != 0 {
		return nil, ycerrors.OperationIncompleteError{ID: ops[0].Id}
	}

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
		return nil, fmt.Errorf("failed to find backend group: %w", err)
	}

	if tg == nil {
		return nil, nil
	}

	// TODO(khodasevich): probably handle operation somehow
	_, err = d.repo.DeleteBackendGroup(ctx, tg)

	return tg, err
}
