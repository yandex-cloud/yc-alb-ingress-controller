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

package controllers

import (
	"context"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	albv1alpha1 "github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	"github.com/yandex-cloud/alb-ingress/controllers/errors"
)

// HTTPBackendGroupReconciler reconciles a HttpBackendGroup object
type HTTPBackendGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	ReconcileHandler
}

//+kubebuilder:rbac:groups=alb.yc.io,resources=httpbackendgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=alb.yc.io,resources=httpbackendgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=alb.yc.io,resources=httpbackendgroups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *HTTPBackendGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rLog := log.FromContext(ctx).WithValues("name", req.NamespacedName, "kind", "HttpBackendGroup")
	rLog.Info("event detected")

	var bg albv1alpha1.HttpBackendGroup
	err := r.Get(ctx, req.NamespacedName, &bg)

	// HttpBackendGroup removed from etcd, and we failed to retrieve it (e.g. forceful deletion)
	if statusError, isStatus := err.(*k8serrors.StatusError); isStatus && statusError.Status().Reason == metav1.StatusReasonNotFound {
		rLog.Info("object not found, probably deleted")
		// TODO: check handler's error after handler is implemented
		_ = r.HandleResourceNotFound(ctx, req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// request failure
	if err != nil {
		rLog.Info("failed to retrieve object")
		return ctrl.Result{RequeueAfter: time.Second * errors.FailureRequeueInterval}, err
	}

	// HttpBackendGroup is being gracefully deleted
	if !bg.DeletionTimestamp.IsZero() {
		rLog.Info("object is being gracefully deleted")
		err = r.HandleResourceDeleted(ctx, &bg)
		return errors.HandleError(err, rLog)
	}

	rLog.Info("object has been created or updated")
	err = r.HandleResourceUpdated(ctx, &bg)
	return errors.HandleError(err, rLog)
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPBackendGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&albv1alpha1.HttpBackendGroup{}).
		Complete(r)
}
