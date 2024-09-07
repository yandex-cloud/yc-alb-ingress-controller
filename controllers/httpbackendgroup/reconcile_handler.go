package httpbackendgroup

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReconcileHandler interface {
	HandleResourceUpdated(ctx context.Context, o client.Object) error
	HandleResourceDeleted(ctx context.Context, o client.Object) error
	HandleResourceNotFound(ctx context.Context, name types.NamespacedName) error
}
