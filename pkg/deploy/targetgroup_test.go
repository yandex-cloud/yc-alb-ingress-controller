package deploy

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"

	"github.com/yandex-cloud/alb-ingress/pkg/deploy/mocks"
	ycerrors "github.com/yandex-cloud/alb-ingress/pkg/errors"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
)

func TestTargetGroupDeployer(t *testing.T) {
	fakeOp := &operation.Operation{Id: "1234"}
	errOpIncomplete := ycerrors.OperationIncompleteError{ID: fakeOp.Id}

	ctx := context.Background()

	t1 := &apploadbalancer.Target{
		AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "192.168.10.30"},
		SubnetId:    "subnet_1",
	}
	t2 := &apploadbalancer.Target{
		AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "192.168.10.28"},
		SubnetId:    "subnet_2",
	}
	t3 := &apploadbalancer.Target{
		AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "192.168.10.26"},
		SubnetId:    "subnet_2",
	}

	labels := &metadata.Labels{ClusterLabelName: "cluster_ref_label", ClusterID: "default"}
	makeTargetGroup := func(name string, targets ...*apploadbalancer.Target) *apploadbalancer.TargetGroup {
		return &apploadbalancer.TargetGroup{
			Name:        name,
			Description: "target group from K8S nodes",
			FolderId:    "folderID",
			Labels:      labels.Default(),
			Targets:     targets,
		}
	}

	t.Run("new target group", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockTargetGroupRepo(ctrl)

		expTG := makeTargetGroup("tg", t1)
		repo.EXPECT().FindTargetGroup(gomock.Any(), "tg").Return(nil, nil)
		repo.EXPECT().CreateTargetGroup(gomock.Any(), expTG).Return(fakeOp, nil)

		d := NewServiceDeployer(repo)
		_, err := d.Deploy(ctx, expTG)
		assert.ErrorAs(t, err, &ycerrors.OperationIncompleteError{})
	})

	t.Run("update target group", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockTargetGroupRepo(ctrl)

		tg := makeTargetGroup("tg", t1, t2)
		expTG := makeTargetGroup("tg", t1, t3)
		repo.EXPECT().FindTargetGroup(gomock.Any(), "tg").Return(tg, nil)
		repo.EXPECT().UpdateTargetGroup(gomock.Any(), expTG).Return(fakeOp, nil)
		repo.EXPECT().ListTargetGroupOperations(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

		d := NewServiceDeployer(repo)
		_, err := d.Deploy(ctx, expTG)
		assert.ErrorAs(t, err, &ycerrors.OperationIncompleteError{})
	})

	t.Run("delete target group", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockTargetGroupRepo(ctrl)

		tg := makeTargetGroup("tg", t1, t2)
		repo.EXPECT().FindTargetGroup(gomock.Any(), "tg").Return(tg, nil)
		repo.EXPECT().DeleteTargetGroup(gomock.Any(), tg).Return(ycerrors.OperationIncompleteError{ID: fakeOp.Id})

		d := NewServiceDeployer(repo)
		_, err := d.Undeploy(ctx, "tg")
		assert.ErrorAs(t, err, &ycerrors.OperationIncompleteError{})
	})

	t.Run("update failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockTargetGroupRepo(ctrl)

		tg := makeTargetGroup("tg", t1, t2)
		expTG := makeTargetGroup("tg", t1, t3)
		repo.EXPECT().FindTargetGroup(gomock.Any(), "tg").Return(tg, nil)
		repo.EXPECT().UpdateTargetGroup(gomock.Any(), expTG).Return(nil, fmt.Errorf("error"))
		repo.EXPECT().ListTargetGroupOperations(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

		d := NewServiceDeployer(repo)
		_, err := d.Deploy(ctx, expTG)
		assert.NotNil(t, err)
		assert.NotErrorIs(t, errOpIncomplete, err)
	})

	t.Run("nothing to update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockTargetGroupRepo(ctrl)

		tg := makeTargetGroup("tg", t1)
		expTG := makeTargetGroup("tg", t1)
		repo.EXPECT().FindTargetGroup(gomock.Any(), "tg").Return(tg, nil)
		repo.EXPECT().ListTargetGroupOperations(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

		d := NewServiceDeployer(repo)
		_, err := d.Deploy(ctx, expTG)
		assert.Nil(t, err)
	})
}
