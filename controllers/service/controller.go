package service

import (
	"context"
	"fmt"

	ycerrors "github.com/yandex-cloud/yc-alb-ingress-controller/pkg/errors"
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

	"github.com/yandex-cloud/yc-alb-ingress-controller/api/v1alpha1"
	errors2 "github.com/yandex-cloud/yc-alb-ingress-controller/controllers/errors"
	"github.com/yandex-cloud/yc-alb-ingress-controller/controllers/service/eventhandlers"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/builders"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/deploy"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/k8s"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/metadata"
	ingressreconcile "github.com/yandex-cloud/yc-alb-ingress-controller/pkg/reconcile"
)

type BackendGroupFinder interface {
	FindTargetGroup(ctx context.Context, name string) (*apploadbalancer.TargetGroup, error)
	FindBackendGroup(ctx context.Context, name string) (*apploadbalancer.BackendGroup, error)
}

//+kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslices,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;

// Reconciler reconciles a Node object
type Reconciler struct {
	Repo BackendGroupFinder

	TargetGroupBuilder  *ingressreconcile.TargetGroupBuilder
	TargetGroupDeployer *deploy.TargetGroupDeployer

	BackendGroupBuilder  *builders.BackendGroupForSvcBuilder
	BackendGroupDeployer *deploy.BackendGroupDeployer

	FinalizerManager   *k8s.FinalizerManager
	GroupStatusManager *k8s.GroupStatusManager
	ServiceLoader      k8s.ServiceLoader
	IngressLoader      k8s.IngressLoader

	Names *metadata.Names

	Resolvers *builders.Resolvers

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
		return nil, fmt.Errorf("failed to load service: %w", err)
	}

	if svc.ToReconcile != nil {
		obj := svc.ToReconcile

		err = r.FinalizerManager.UpdateFinalizer(ctx, svc.ToReconcile, k8s.Finalizer)
		if err != nil {
			return obj, fmt.Errorf("failed to update finalizer: %w", err)
		}

		locations, err := r.getCommonLocations(svc.References)
		if err != nil {
			return obj, fmt.Errorf("failed to get common network: %w", err)
		}

		tg, err := r.TargetGroupBuilder.Build(ctx, req.NamespacedName, locations)
		if err != nil {
			return obj, fmt.Errorf("failed to build target group: %w", err)
		}

		tg, err = r.TargetGroupDeployer.Deploy(ctx, tg)
		if err != nil {
			return obj, fmt.Errorf("failed to deploy target group: %w", err)
		}

		ings, err := r.IngressLoader.ListBySvc(ctx, *svc.ToReconcile)
		if err != nil {
			return obj, fmt.Errorf("failed to list ingresses by service: %w", err)
		}

		if len(ings) != 0 {
			// Service is referenced directly by ingress, not by HttpBackendGroup or GrpcBackendGroup

			bg, err := r.BackendGroupBuilder.BuildForSvc(svc.ToReconcile, ings, tg.Id)
			if err != nil {
				return obj, fmt.Errorf("failed to build backend group: %w", err)
			}

			bg, err = r.BackendGroupDeployer.Deploy(ctx, bg)
			if err != nil {
				return obj, fmt.Errorf("failed to deploy backend group: %w", err)
			}
			err = r.AddBGIDToGroupStatuses(ctx, *svc.ToReconcile, bg)
			if err != nil {
				return obj, fmt.Errorf("failed to add bg id to group statuses: %w", err)
			}
		}

		err = r.AddTGIDToGroupStatuses(ctx, *svc.ToReconcile, tg)
		if err != nil {
			return obj, fmt.Errorf("failed to add tg id to group statuses: %w", err)
		}
		return obj, nil
	}

	if svc.ToDelete != nil {
		obj := svc.ToDelete

		bgName := r.Names.NewBackendGroup(req.NamespacedName)
		bg, err := r.Repo.FindBackendGroup(ctx, bgName)
		if err != nil && !errors.IsNotFound(err) {
			return obj, fmt.Errorf("failed to find backend group: %w", err)
		}
		if err == nil {
			err = r.RemoveBGIDFromGroupStatuses(ctx, *svc.ToDelete, bg)
			if err != nil {
				return obj, fmt.Errorf("failed to remove ids from group statuses: %w", err)
			}

			_, err = r.BackendGroupDeployer.Undeploy(ctx, bgName)
			if err != nil {
				return obj, fmt.Errorf("failed to undeploy backend group: %w", err)
			}
		}

		tgName := r.Names.TargetGroup(req.NamespacedName)

		tg, err := r.Repo.FindTargetGroup(ctx, tgName)
		if err != nil {
			return obj, fmt.Errorf("failed to find target group: %w", err)
		}

		err = r.RemoveTGIDFromGroupStatuses(ctx, *svc.ToDelete, tg)
		if err != nil {
			return obj, fmt.Errorf("failed to remove ids from group statuses: %w", err)
		}

		_, err = r.TargetGroupDeployer.Undeploy(ctx, tgName)
		if err != nil {
			return obj, fmt.Errorf("failed to undeploy target group: %w", err)
		}

		err = r.FinalizerManager.RemoveFinalizer(ctx, svc.ToDelete, k8s.Finalizer)
		if err != nil {
			return obj, fmt.Errorf("failed to remove finalizer: %w", err)
		}
		return obj, nil
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

func (r *Reconciler) getCommonLocations(groups map[string]k8s.IngressGroup) ([]*apploadbalancer.Location, error) {
	var commonLocations map[string]*apploadbalancer.Location
	for _, group := range groups {
		resolver := r.Resolvers.Location()
		for _, ing := range group.Items {
			err := resolver.Resolve(ing.GetAnnotations()[k8s.Subnets])
			if err != nil {
				return nil, fmt.Errorf("failed to resolve location: %w", err)
			}
		}

		_, locations, err := resolver.Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get result: %w", err)
		}
		if commonLocations == nil {
			commonLocations = make(map[string]*apploadbalancer.Location)
			for _, location := range locations {
				commonLocations[location.SubnetId] = location
			}
		} else {
			for _, location := range locations {
				if _, ok := commonLocations[location.SubnetId]; !ok {
					delete(commonLocations, location.SubnetId)
				}
			}
		}
	}

	var res []*apploadbalancer.Location
	for _, location := range commonLocations {
		res = append(res, location)
	}
	return res, nil
}

func (r *Reconciler) updateGroupStatuses(
	ctx context.Context, svc core.Service,
	updateFunc func(ctx context.Context, groupName string) error,
) error {
	ings, err := r.IngressLoader.ListBySvc(ctx, svc)
	if err != nil {
		return fmt.Errorf("failed to list ingresses by service: %w", err)
	}

	groups := getGroupNamesFromIngresses(ings)
	for group := range groups {
		err = updateFunc(ctx, group)
		if err != nil {
			return fmt.Errorf("failed to exec updateFunc: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) AddTGIDToGroupStatuses(ctx context.Context, svc core.Service, tg *apploadbalancer.TargetGroup) error {
	return r.updateGroupStatuses(ctx, svc, func(ctx context.Context, group string) error {
		status, err := r.GroupStatusManager.LoadStatus(ctx, group)
		if errors.IsNotFound(err) {
			return ycerrors.ResourceNotReadyError{
				ResourceType: "IngressGroupStatus",
				Name:         group,
			}
		}
		if err != nil {
			return fmt.Errorf("failed to load group status: %w", err)
		}

		err = r.GroupStatusManager.AddTargetGroupID(ctx, status, tg.Id)
		if err != nil {
			return fmt.Errorf("failed to add target group id: %w", err)
		}
		return err
	})
}

func (r *Reconciler) AddBGIDToGroupStatuses(ctx context.Context, svc core.Service, bg *apploadbalancer.BackendGroup) error {
	return r.updateGroupStatuses(ctx, svc, func(ctx context.Context, group string) error {
		status, err := r.GroupStatusManager.LoadStatus(ctx, group)
		if errors.IsNotFound(err) {
			return ycerrors.ResourceNotReadyError{
				ResourceType: "IngressGroupStatus",
				Name:         group,
			}
		}
		if err != nil {
			return fmt.Errorf("failed to load group status: %w", err)
		}

		err = r.GroupStatusManager.AddBackendGroupID(ctx, status, bg.Id)
		if err != nil {
			return fmt.Errorf("failed to add backend group id: %w", err)
		}
		return err
	})
}

func (r *Reconciler) RemoveTGIDFromGroupStatuses(ctx context.Context, svc core.Service, tg *apploadbalancer.TargetGroup) error {
	return r.updateGroupStatuses(ctx, svc, func(ctx context.Context, group string) error {
		status, err := r.GroupStatusManager.LoadStatus(ctx, group)
		if errors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to load group status: %w", err)
		}

		err = r.GroupStatusManager.RemoveTargetGroupID(ctx, status, tg.Id)
		if err != nil {
			return fmt.Errorf("failed to remove target group id: %w", err)
		}
		return err
	})
}

func (r *Reconciler) RemoveBGIDFromGroupStatuses(ctx context.Context, svc core.Service, bg *apploadbalancer.BackendGroup) error {
	return r.updateGroupStatuses(ctx, svc, func(ctx context.Context, group string) error {
		status, err := r.GroupStatusManager.LoadStatus(ctx, group)
		if errors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to load group status: %w", err)
		}

		err = r.GroupStatusManager.RemoveBackendGroupID(ctx, status, bg.Id)
		if err != nil {
			return fmt.Errorf("failed to remove backend group id: %w", err)
		}
		return err
	})
}

// SetupWithManager sets up the controller with the manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, useEndpointSlices bool) error {
	c, err := controller.New("service", mgr, controller.Options{
		MaxConcurrentReconciles: 1,
		Reconciler:              r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	err = c.Watch(&source.Kind{Type: &core.Service{}}, eventhandlers.NewServiceEventHandler(mgr.GetLogger(), mgr.GetClient()))
	if err != nil {
		return fmt.Errorf("failed to watch services: %w", err)
	}

	if useEndpointSlices {
		err = c.Watch(&source.Kind{Type: &discovery.EndpointSlice{}}, eventhandlers.NewEndpointSliceEventHandler(mgr.GetLogger(), mgr.GetClient()))
		if err != nil {
			return fmt.Errorf("failed to watch endpoint slices: %w", err)
		}
	} else {
		err = c.Watch(&source.Kind{Type: &core.Endpoints{}}, eventhandlers.NewEndpointsEventHandler(mgr.GetLogger(), mgr.GetClient(), mgr.GetEventRecorderFor("service")))
		if err != nil {
			return fmt.Errorf("failed to watch endpoints: %w", err)
		}
	}

	err = c.Watch(&source.Kind{Type: &networking.Ingress{}}, eventhandlers.NewIngressEventHandler(mgr.GetLogger(), mgr.GetClient()))
	if err != nil {
		return fmt.Errorf("failed to watch ingresses: %w", err)
	}

	err = c.Watch(&source.Kind{Type: &v1alpha1.HttpBackendGroup{}}, eventhandlers.NewHTTPBackendGroupEventHandler(mgr.GetLogger(), mgr.GetClient()))
	if err != nil {
		return fmt.Errorf("failed to watch http backend groups: %w", err)
	}

	r.recorder = mgr.GetEventRecorderFor(k8s.ControllerName)

	return nil
}
