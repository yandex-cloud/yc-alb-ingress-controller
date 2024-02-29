package k8s

import (
	"context"

	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
)

type GroupStatusManager struct {
	cli client.Client
}

func NewGroupStatusManager(cli client.Client) *GroupStatusManager {
	return &GroupStatusManager{cli: cli}
}

func (h *GroupStatusManager) AddTargetGroupID(ctx context.Context, status *v1alpha1.IngressGroupStatus, tgID string) error {
	for _, id := range status.TargetGroupIDs {
		if id == tgID {
			return nil
		}
	}

	oldStatus := status.DeepCopy()
	status.TargetGroupIDs = append(status.TargetGroupIDs, tgID)
	return h.cli.Patch(ctx, status, client.MergeFrom(oldStatus))
}

func (h *GroupStatusManager) AddBackendGroupID(ctx context.Context, status *v1alpha1.IngressGroupStatus, bgID string) error {
	for _, id := range status.BackendGroupIDs {
		if id == bgID {
			return nil
		}
	}

	oldStatus := status.DeepCopy()
	status.TargetGroupIDs = append(status.TargetGroupIDs, bgID)
	return h.cli.Patch(ctx, status, client.MergeFrom(oldStatus))
}

func (h *GroupStatusManager) RemoveTargetGroupID(ctx context.Context, status *v1alpha1.IngressGroupStatus, tgID string) error {
	oldStatus := status.DeepCopy()

	for i, id := range status.TargetGroupIDs {
		if id != tgID {
			continue
		}

		status.TargetGroupIDs = append(status.TargetGroupIDs[:i], status.TargetGroupIDs[i+1:]...)
		break
	}

	return h.cli.Patch(ctx, status, client.MergeFrom(oldStatus))
}

func (h *GroupStatusManager) RemoveBackendGroupID(ctx context.Context, status *v1alpha1.IngressGroupStatus, bgID string) error {
	oldStatus := status.DeepCopy()

	for i, id := range status.BackendGroupIDs {
		if id != bgID {
			continue
		}

		status.BackendGroupIDs = append(status.BackendGroupIDs[:i], status.BackendGroupIDs[i+1:]...)
		break
	}

	return h.cli.Patch(ctx, status, client.MergeFrom(oldStatus))
}

type ResourcesIDs struct {
	BalancerID  string
	RouterID    string
	TLSRouterID string
}

func (h *GroupStatusManager) SetBalancerResourcesIDs(ctx context.Context, status *v1alpha1.IngressGroupStatus, resources ResourcesIDs) error {
	oldStatus := status.DeepCopy()
	status.LoadBalancerID = resources.BalancerID
	status.TLSRouterID = resources.TLSRouterID
	status.HTTPRouterID = resources.RouterID

	return h.cli.Patch(ctx, status, client.MergeFrom(oldStatus))
}

func (h *GroupStatusManager) LoadStatus(ctx context.Context, name string) (*v1alpha1.IngressGroupStatus, error) {
	var status v1alpha1.IngressGroupStatus
	err := h.cli.Get(ctx, types.NamespacedName{Name: name}, &status)
	return &status, err
}

func (h *GroupStatusManager) LoadOrCreateStatus(ctx context.Context, name string) (*v1alpha1.IngressGroupStatus, error) {
	status, err := h.LoadStatus(ctx, name)
	if errors.IsNotFound(err) {
		status = &v1alpha1.IngressGroupStatus{}
		status.Name = name
		err = h.cli.Create(ctx, status)
	}
	return status, err
}

func (h *GroupStatusManager) DeleteStatus(ctx context.Context, name string) error {
	status, err := h.LoadStatus(ctx, name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return h.cli.Delete(ctx, status)
}

type StatusUpdater struct {
	Client client.Client
}

// TODO: update only if needed
func (e *StatusUpdater) SetIngressStatus(ing *networking.Ingress, status networking.IngressStatus) error {
	oldIng := ing.DeepCopy()
	ing.Status = status
	p := client.MergeFrom(oldIng)
	return e.Client.Status().Patch(context.Background(), ing, p)
}
