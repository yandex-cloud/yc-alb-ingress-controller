package service

import (
	"context"

	ycerrors "github.com/yandex-cloud/alb-ingress/pkg/errors"
	"k8s.io/client-go/tools/record"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	core "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	errors2 "github.com/yandex-cloud/alb-ingress/controllers/errors"
	"github.com/yandex-cloud/alb-ingress/controllers/service/eventhandlers"
	"github.com/yandex-cloud/alb-ingress/pkg/builders"
	"github.com/yandex-cloud/alb-ingress/pkg/deploy"
	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
	ingressreconcile "github.com/yandex-cloud/alb-ingress/pkg/reconcile"
)

//+kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslices,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;

// Reconciler reconciles a Node object
type Reconciler struct {
	TargetGroupBuilder  *ingressreconcile.TargetGroupBuilder
	TargetGroupDeployer *deploy.TargetGroupDeployer

	BackendGroupBuilder  *builders.BackendGroupForSvcBuilder
	BackendGroupDeployer *deploy.BackendGroupDeployer

	FinalizerManager   *k8s.FinalizerManager
	GroupStatusManager *k8s.GroupStatusManager
	ServiceLoader      k8s.ServiceLoader
	IngressLoader      k8s.IngressLoader

	Names *metadata.Names

	recorder record.EventRecorder
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rLog := log.FromContext(ctx)
	rLog.Info("event detected")
	svc, err := r.doReconcile(ctx, req)
	errors2.HandleErrorWithObject(err, svc, r.recorder)
	return errors2.HandleError(err, rLog)
}

func (r *Reconciler) doReconcile(ctx context.Context, req ctrl.Request) (*core.Service, error) {
	svc, err := r.ServiceLoader.Load(ctx, req.NamespacedName)
	if err != nil {
		return nil, err
	}

	if svc.ToReconcile != nil {
		obj := svc.ToReconcile

		err = r.FinalizerManager.UpdateFinalizer(ctx, svc.ToReconcile, k8s.Finalizer)
		if err != nil {
			return obj, err
		}

		tg, err := r.TargetGroupBuilder.Build(ctx, req.NamespacedName)
		if err != nil {
			return obj, err
		}

		tg, err = r.TargetGroupDeployer.Deploy(ctx, tg)
		if err != nil {
			return obj, err
		}

		ings, err := r.IngressLoader.ListBySvc(ctx, *svc.ToReconcile)
		if err != nil {
			return obj, err
		}

		bg, err := r.BackendGroupBuilder.BuildForSvc(svc.ToReconcile, ings, tg.Id)
		if err != nil {
			return obj, err
		}

		bg, err = r.BackendGroupDeployer.Deploy(ctx, bg)
		if err != nil {
			return obj, err
		}

		err = r.AddIDsToGroupStatuses(ctx, *svc.ToReconcile, tg, bg)
		if err != nil {
			return obj, err
		}
		return obj, err
	}

	if svc.ToDelete != nil {
		obj := svc.ToDelete

		bg, err := r.BackendGroupDeployer.Undeploy(ctx, r.Names.NewBackendGroup(req.NamespacedName))
		if err != nil {
			return obj, err
		}

		tg, err := r.TargetGroupDeployer.Undeploy(ctx, r.Names.TargetGroup(req.NamespacedName))
		if err != nil {
			return obj, err
		}

		err = r.RemoveIDsFromGroupStatuses(ctx, *svc.ToDelete, tg, bg)
		if err != nil {
			return obj, err
		}

		err = r.FinalizerManager.RemoveFinalizer(ctx, svc.ToDelete, k8s.Finalizer)
		if err != nil {
			return obj, err
		}
		return obj, err
	}

	return nil, err
}

func getGroupNamesFromIngresses(ings []networking.Ingress) map[string]struct{} {
	res := make(map[string]struct{})
	for _, ing := range ings {
		res[ing.GetAnnotations()[k8s.AlbTag]] = struct{}{}
	}
	return res
}

func (r *Reconciler) updateGroupStatuses(
	ctx context.Context, svc core.Service,
	updateFunc func(ctx context.Context, groupName string) error,
) error {
	ings, err := r.IngressLoader.ListBySvc(ctx, svc)
	if err != nil {
		return err
	}

	groups := getGroupNamesFromIngresses(ings)
	for group := range groups {
		err = updateFunc(ctx, group)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) AddIDsToGroupStatuses(ctx context.Context, svc core.Service, tg *apploadbalancer.TargetGroup, bg *apploadbalancer.BackendGroup) error {
	return r.updateGroupStatuses(ctx, svc, func(ctx context.Context, group string) error {
		status, err := r.GroupStatusManager.LoadStatus(ctx, group)
		if errors.IsNotFound(err) {
			return ycerrors.ResourceNotReadyError{
				ResourceType: "IngressGroupStatus",
				Name:         group,
			}
		}
		if err != nil {
			return err
		}

		err = r.GroupStatusManager.AddTargetGroupID(ctx, status, tg.Id)
		if err != nil {
			return err
		}

		return r.GroupStatusManager.AddBackendGroupID(ctx, status, bg.Id)
	})
}

func (r *Reconciler) RemoveIDsFromGroupStatuses(ctx context.Context, svc core.Service, tg *apploadbalancer.TargetGroup, bg *apploadbalancer.BackendGroup) error {
	return r.updateGroupStatuses(ctx, svc, func(ctx context.Context, group string) error {
		status, err := r.GroupStatusManager.LoadStatus(ctx, group)

		// Do nothing if status is already deleted by group reconciler
		if errors.IsNotFound(err) {
			return nil
		}

		if err != nil {
			return err
		}

		err = r.GroupStatusManager.RemoveTargetGroupID(ctx, status, tg.Id)
		if err != nil {
			return err
		}

		return r.GroupStatusManager.RemoveBackendGroupID(ctx, status, bg.Id)
	})
}

// SetupWithManager sets up the controller with the manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, useEndpointSlices bool) error {
	c, err := controller.New("service", mgr, controller.Options{
		MaxConcurrentReconciles: 1,
		Reconciler:              r,
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &core.Service{}}, eventhandlers.NewServiceEventHandler(mgr.GetLogger(), mgr.GetClient()))
	if err != nil {
		return err
	}

	if useEndpointSlices {
		err = c.Watch(&source.Kind{Type: &discovery.EndpointSlice{}}, eventhandlers.NewEndpointSliceEventHandler(mgr.GetLogger(), mgr.GetClient()))
		if err != nil {
			return err
		}
	} else {
		err = c.Watch(&source.Kind{Type: &core.Endpoints{}}, eventhandlers.NewEndpointsEventHandler(mgr.GetLogger(), mgr.GetClient(), mgr.GetEventRecorderFor("service")))
		if err != nil {
			return err
		}
	}

	err = c.Watch(&source.Kind{Type: &networking.Ingress{}}, eventhandlers.NewIngressEventHandler(mgr.GetLogger(), mgr.GetClient()))
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1alpha1.HttpBackendGroup{}}, eventhandlers.NewHTTPBackendGroupEventHandler(mgr.GetLogger(), mgr.GetClient()))
	if err != nil {
		return err
	}

	r.recorder = mgr.GetEventRecorderFor(k8s.ControllerName)

	return nil
}
