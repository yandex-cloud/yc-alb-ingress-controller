/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpcbackendgroup

import (
	"context"
	"errors"
	"time"

	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/k8s"
	"k8s.io/client-go/tools/record"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	albv1alpha1 "github.com/yandex-cloud/yc-alb-ingress-controller/api/v1alpha1"
	errors2 "github.com/yandex-cloud/yc-alb-ingress-controller/controllers/errors"
)

// Reconciler reconciles a GrpcBackendGroup object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	ReconcileHandler

	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=alb.yc.io,resources=grpcbackendgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=alb.yc.io,resources=grpcbackendgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=alb.yc.io,resources=grpcbackendgroups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rLog := log.FromContext(ctx).WithValues("name", req.NamespacedName, "kind", "GrpcBackendGroup")
	rLog.Info("event detected")

	var bg albv1alpha1.GrpcBackendGroup
	err := r.Get(ctx, req.NamespacedName, &bg)

	// GrpcBackendGroup removed from etcd, and we failed to retrieve it (e.g. forceful deletion)
	var statusError *k8serrors.StatusError
	if errors.As(err, &statusError) && statusError.Status().Reason == metav1.StatusReasonNotFound {
		rLog.Info("object not found, probably deleted")
		// TODO: check handler's error after handler is implemented
		_ = r.HandleResourceNotFound(ctx, req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// request failure
	if err != nil {
		rLog.Info("failed to retrieve object")
		return ctrl.Result{RequeueAfter: time.Second * errors2.FailureRequeueInterval}, err
	}

	// GrpcBackendGroup is being gracefully deleted
	if !bg.DeletionTimestamp.IsZero() {
		rLog.Info("object is being gracefully deleted")
		err = r.HandleResourceDeleted(ctx, &bg)
		errors2.HandleErrorWithObject(err, &bg, r.recorder)
		return errors2.HandleError(err, rLog)
	}

	rLog.Info("object has been created or updated")
	err = r.HandleResourceUpdated(ctx, &bg)
	errors2.HandleErrorWithObject(err, &bg, r.recorder)
	return errors2.HandleError(err, rLog)
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor(k8s.ControllerName)
	return ctrl.NewControllerManagedBy(mgr).
		For(&albv1alpha1.GrpcBackendGroup{}).
		Complete(r)
}
