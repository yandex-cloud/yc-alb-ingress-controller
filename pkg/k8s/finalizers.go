package k8s

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FinalizerManager struct{ Client client.Client }

func (m *FinalizerManager) UpdateFinalizer(ctx context.Context, o client.Object, finalizer string) error {
	var i int
	finalizers := o.GetFinalizers()
	for ; i < len(finalizers) && finalizers[i] != finalizer; i++ {
	}
	if i < len(finalizers) {
		return nil
	} // already set
	oldIng := o.DeepCopyObject().(client.Object)
	o.SetFinalizers(append(finalizers, finalizer))
	return m.Client.Patch(ctx, o, client.MergeFrom(oldIng))
}

func (m *FinalizerManager) RemoveFinalizer(ctx context.Context, o client.Object, finalizer string) error {
	var i int
	finalizers := o.GetFinalizers()
	l := len(finalizers)
	oldIng := o.DeepCopyObject().(client.Object)
	for ; i < l; i++ {
		if finalizers[i] == finalizer {
			copy(finalizers[i:], finalizers[i+1:])
			l--
			i--
		}
	}
	if l == len(finalizers) { // already removed
		return nil
	}
	o.SetFinalizers(finalizers[:l])
	return m.Client.Patch(ctx, o, client.MergeFrom(oldIng))
}
