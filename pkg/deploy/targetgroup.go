package deploy

import (
	"context"
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	"google.golang.org/protobuf/proto"

	ycerrors "github.com/yandex-cloud/alb-ingress/pkg/errors"
)

//go:generate mockgen -destination=./mocks/targetgroup.go -package=mocks . TargetGroupRepo

type TargetGroupRepo interface {
	FindTargetGroup(context.Context, string) (*apploadbalancer.TargetGroup, error)
	CreateTargetGroup(context.Context, *apploadbalancer.TargetGroup) (*operation.Operation, error)
	UpdateTargetGroup(context.Context, *apploadbalancer.TargetGroup) (*operation.Operation, error)
	DeleteTargetGroup(context.Context, *apploadbalancer.TargetGroup) error
}

type TargetGroupDeployer struct {
	repo TargetGroupRepo
}

func NewServiceDeployer(repo TargetGroupRepo) *TargetGroupDeployer {
	return &TargetGroupDeployer{
		repo: repo,
	}
}

func (d *TargetGroupDeployer) Undeploy(ctx context.Context, name string) (*apploadbalancer.TargetGroup, error) {
	tg, err := d.repo.FindTargetGroup(ctx, name)
	if err != nil {
		return nil, err
	}

	if tg == nil {
		return nil, nil
	}

	return tg, d.repo.DeleteTargetGroup(ctx, tg)
}

func (d *TargetGroupDeployer) Deploy(ctx context.Context, expected *apploadbalancer.TargetGroup) (*apploadbalancer.TargetGroup, error) {
	actual, err := d.repo.FindTargetGroup(ctx, expected.Name)
	if err != nil {
		return nil, err
	}

	// create if needed
	if actual == nil {
		op, err := d.repo.CreateTargetGroup(ctx, expected)
		if err != nil {
			return nil, fmt.Errorf("failed to create target group: %w", err)
		}
		if op != nil {
			return nil, ycerrors.OperationIncompleteError{ID: op.Id}
		}
		return actual, nil
	}

	// update if needed
	if tgUpdateNeeded(expected.Targets, actual.Targets) {
		expected.Id = actual.Id
		op, err := d.repo.UpdateTargetGroup(ctx, expected)
		if err != nil {
			return nil, fmt.Errorf("failed to update target group: %w", err)
		}
		if op != nil {
			return nil, ycerrors.OperationIncompleteError{ID: op.Id}
		}
		return actual, nil
	}

	return actual, nil
}

func tgUpdateNeeded(actual, expected []*apploadbalancer.Target) bool {
	if len(expected) != len(actual) {
		return true
	}

	for i := 0; i < len(expected); i++ {
		if !proto.Equal(expected[i], actual[i]) {
			return true
		}
	}

	return false
}
