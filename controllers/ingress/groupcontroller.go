package ingress

import (
	"context"
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	"github.com/yandex-cloud/alb-ingress/controllers/errors"
	"github.com/yandex-cloud/alb-ingress/pkg/yc"

	"github.com/yandex-cloud/alb-ingress/controllers/ingress/eventhandlers"
	"github.com/yandex-cloud/alb-ingress/pkg/deploy"
	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	reconcile2 "github.com/yandex-cloud/alb-ingress/pkg/reconcile"
)

//go:generate mockgen -destination=./mocks/mocks.go -package=mocks . GroupLoader,EngineBuilder,Deployer,StatusResolver

//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=update;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;update;create

//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/finalizers,verbs=update

//+kubebuilder:rbac:groups=alb.yc.io,resources=ingressgroupstatuses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=alb.yc.io,resources=ingressgroupsettings,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch;create;update;patch;delete

type GroupLoader interface {
	Load(context.Context, types.NamespacedName) (*k8s.IngressGroup, error)
}

type EngineBuilder interface {
	Build(ctx context.Context, group *k8s.IngressGroup, settings *v1alpha1.IngressGroupSettings) (*reconcile2.IngressGroupEngine, error)
}

type Deployer interface {
	Deploy(ctx context.Context, tag string, re deploy.ReconcileEngine) (yc.BalancerResources, error)
	UndeployOldBG(ctx context.Context, tag string) error
}

type StatusResolver interface {
	Resolve(*apploadbalancer.LoadBalancer) networking.IngressStatus
}

type SettingsLoader interface {
	Load(ctx context.Context, g *k8s.IngressGroup) (*v1alpha1.IngressGroupSettings, error)
}

// GroupReconciler reconciles an IngressGroup object
type GroupReconciler struct {
	Loader   GroupLoader
	Builder  EngineBuilder
	Deployer Deployer

	SecretsManager k8s.SecretManager

	StatusUpdater      *k8s.StatusUpdater
	FinalizerManager   *k8s.FinalizerManager
	GroupStatusManager *k8s.GroupStatusManager

	StatusResolver StatusResolver
	SettingsLoader SettingsLoader

	Scheme *runtime.Scheme
}

func (r *GroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rLog := log.FromContext(ctx).WithValues("name", req.NamespacedName, "kind", "IngressGroup")
	rLog.Info("Group event Detected")
	err := r.doReconcile(ctx, req)
	return errors.HandleError(err, rLog)
}

// SetupWithManager sets up the controller with the manager.
func (r *GroupReconciler) SetupWithManager(
	mgr ctrl.Manager,
	clientSet *kubernetes.Clientset,
	secretEventChan chan event.GenericEvent,
) error {
	c, err := controller.New("ingressgroup", mgr, controller.Options{
		MaxConcurrentReconciles: 1,
		Reconciler:              r,
	})
	if err != nil {
		return err
	}

	eventRecorder := mgr.GetEventRecorderFor("ingress-group")

	err = c.Watch(&source.Kind{Type: &v1.Service{}}, eventhandlers.NewServiceEventHandler(mgr.GetLogger(), mgr.GetClient(), eventRecorder))
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1alpha1.IngressGroupSettings{}}, eventhandlers.NewSettingsEventHandler(mgr.GetLogger(), mgr.GetClient()))
	if err != nil {
		return err
	}

	err = r.setupIngressWatches(c)
	if err != nil {
		return err
	}

	r.SecretsManager = k8s.NewSecretManager(clientSet, secretEventChan)

	cli := mgr.GetClient()
	return r.setupIngressClassesWatch(c, cli, eventRecorder)
}

func (r *GroupReconciler) doReconcile(ctx context.Context, req ctrl.Request) error {
	g, err := r.Loader.Load(ctx, req.NamespacedName)
	if err != nil || g == nil {
		return err
	}
	err = r.updateGroupFinalizer(ctx, g)
	if err != nil {
		return err
	}

	r.SecretsManager.ManageGroup(ctx, g)

	settings, err := r.SettingsLoader.Load(ctx, g)
	if err != nil {
		return err
	}

	reconcileEngine, err := r.Builder.Build(ctx, g, settings)
	if err != nil {
		return err
	}
	balancerResources, err := r.Deployer.Deploy(ctx, g.Tag, reconcileEngine)
	if err != nil {
		return err
	}

	err = r.setGroupStatus(ctx, g, balancerResources)
	if err != nil {
		return err
	}

	err = r.deleteOldBackendGroups(ctx, g.Tag)
	if err != nil {
		return err
	}

	err = r.removeGroupFinalizer(ctx, g)
	if err != nil {
		return err
	}
	return nil
}

func (r *GroupReconciler) deleteOldBackendGroups(ctx context.Context, tag string) error {
	return r.Deployer.UndeployOldBG(ctx, tag)
}

func (r *GroupReconciler) updateGroupFinalizer(ctx context.Context, g *k8s.IngressGroup) error {
	for _, item := range g.Items {
		if err := r.FinalizerManager.UpdateFinalizer(ctx, &item, k8s.Finalizer); err != nil {
			return err
		}
	}
	return nil
}

func (r *GroupReconciler) removeGroupFinalizer(ctx context.Context, g *k8s.IngressGroup) error {
	for _, item := range g.Deleted {
		if err := r.FinalizerManager.RemoveFinalizer(ctx, &item, k8s.Finalizer); err != nil {
			return err
		}
	}
	return nil
}

func (r *GroupReconciler) setGroupStatus(ctx context.Context, g *k8s.IngressGroup, resources yc.BalancerResources) error {
	albStatus := r.StatusResolver.Resolve(resources.Balancer)
	for _, item := range g.Items {
		if err := r.StatusUpdater.SetIngressStatus(&item, albStatus); err != nil {
			return err
		}
	}

	if len(g.Items) == 0 {
		return r.GroupStatusManager.DeleteStatus(ctx, g.Tag)
	}

	groupStatus, err := r.GroupStatusManager.LoadOrCreateStatus(ctx, g.Tag)
	if err != nil {
		return err
	}

	var ids k8s.ResourcesIDs
	if resources.Balancer != nil {
		ids.BalancerID = resources.Balancer.Id
	}
	if resources.TLSRouter != nil {
		ids.TLSRouterID = resources.TLSRouter.Id
	}
	if resources.Router != nil {
		ids.RouterID = resources.Router.Id
	}

	err = r.GroupStatusManager.SetBalancerResourcesIDs(ctx, groupStatus, ids)
	if err != nil {
		return err
	}

	return nil
}

func (r *GroupReconciler) setupIngressClassesWatch(c controller.Controller, cli client.Client, recorder record.EventRecorder) error {
	mapFn := func(a client.Object) []reconcile.Request {
		class := a.(*networking.IngressClass)

		ingList := &networking.IngressList{}
		if err := cli.List(context.Background(), ingList); err != nil {
			recorder.Event(class, v1.EventTypeWarning, "FailedToLoadIngress", fmt.Sprintf("failed to load ingresses due %e", err))
			return nil
		}

		result := make([]reconcile.Request, 0)
		for _, item := range ingList.Items {
			if item.Spec.IngressClassName == nil && class.Annotations[k8s.DefaultIngressClass] != "true" {
				continue
			}

			if item.Spec.IngressClassName != nil && *item.Spec.IngressClassName != class.Name {
				continue
			}

			result = append(result, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: k8s.GetBalancerTag(&item),
				},
			})
		}

		return result
	}
	filterFn := func(o client.Object) bool {
		class := o.(*networking.IngressClass)
		return class.Spec.Controller == k8s.ControllerName
	}

	return c.Watch(&source.Kind{Type: &networking.IngressClass{}}, handler.EnqueueRequestsFromMapFunc(mapFn), predicate.NewPredicateFuncs(filterFn))
}

func (r *GroupReconciler) setupIngressWatches(c controller.Controller) error {
	// Define a mapping from the object in the event to one or more
	// objects to Reconcile
	mapFn := func(a client.Object) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name: k8s.GetBalancerTag(a),
				},
			},
		}
	}
	filterFn := func(o client.Object) bool {
		_, ok := o.GetAnnotations()[k8s.AlbTag]
		return ok
	}
	err := c.Watch(&source.Kind{Type: &networking.Ingress{}}, handler.EnqueueRequestsFromMapFunc(mapFn), predicate.NewPredicateFuncs(filterFn))
	if err != nil {
		return err
	}
	return nil
}
