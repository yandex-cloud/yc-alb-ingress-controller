package eventhandlers

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/yandex-cloud/yc-alb-ingress-controller/api/v1alpha1"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/algo"
)

type GRPCBackendGroupEventHandler struct {
	Log logr.Logger
	cli client.Client
}

func (s GRPCBackendGroupEventHandler) Create(event event.CreateEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service create event detected")

	s.Common(event.Object.(*v1alpha1.GrpcBackendGroup), q)
}

func (s GRPCBackendGroupEventHandler) Update(event event.UpdateEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.ObjectNew.GetNamespace(),
		"name", event.ObjectNew.GetName()).
		Info("Service update event detected")

	oldServices := parseServicesFromGRPCBG(event.ObjectOld.(*v1alpha1.GrpcBackendGroup))
	newServices := parseServicesFromGRPCBG(event.ObjectNew.(*v1alpha1.GrpcBackendGroup))

	// trigger only inserted or removed services
	toUpdate := algo.SetsExceptUnion(oldServices, newServices)
	for svc := range toUpdate {
		q.Add(ctrl.Request{NamespacedName: svc})
	}
}

func (s GRPCBackendGroupEventHandler) Delete(event event.DeleteEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service delete event detected")

	s.Common(event.Object.(*v1alpha1.GrpcBackendGroup), q)
}

func (s GRPCBackendGroupEventHandler) Generic(event event.GenericEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Generic node event detected")

	s.Common(event.Object.(*v1alpha1.GrpcBackendGroup), q)
}

func (s GRPCBackendGroupEventHandler) Common(ing *v1alpha1.GrpcBackendGroup, q workqueue.RateLimitingInterface) {
	svcs := parseServicesFromGRPCBG(ing)
	for svc := range svcs {
		q.Add(ctrl.Request{NamespacedName: svc})
	}
}

func NewGRPCBackendGroupEventHandler(logger logr.Logger, cli client.Client) *GRPCBackendGroupEventHandler {
	return &GRPCBackendGroupEventHandler{Log: logger, cli: cli}
}

func parseServicesFromGRPCBG(ing *v1alpha1.GrpcBackendGroup) map[types.NamespacedName]struct{} {
	result := make(map[types.NamespacedName]struct{})

	for _, be := range ing.Spec.Backends {
		if be.Service == nil {
			continue
		}

		result[types.NamespacedName{
			Name:      be.Service.Name,
			Namespace: ing.Namespace,
		}] = struct{}{}
	}

	return result
}
