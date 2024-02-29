package eventhandlers

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	"github.com/yandex-cloud/alb-ingress/pkg/algo"
)

type HTTPBackendGroupEventHandler struct {
	Log logr.Logger
	cli client.Client
}

func (s HTTPBackendGroupEventHandler) Create(event event.CreateEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service create event detected")

	s.Common(event.Object.(*v1alpha1.HttpBackendGroup), q)
}

func (s HTTPBackendGroupEventHandler) Update(event event.UpdateEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.ObjectNew.GetNamespace(),
		"name", event.ObjectNew.GetName()).
		Info("Service update event detected")

	oldServices := parseServicesFromBG(event.ObjectOld.(*v1alpha1.HttpBackendGroup))
	newServices := parseServicesFromBG(event.ObjectNew.(*v1alpha1.HttpBackendGroup))

	// trigger only inserted or removed services
	toUpdate := algo.SetsExceptUnion(oldServices, newServices)
	for svc := range toUpdate {
		q.Add(ctrl.Request{NamespacedName: svc})
	}
}

func (s HTTPBackendGroupEventHandler) Delete(event event.DeleteEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service delete event detected")

	s.Common(event.Object.(*v1alpha1.HttpBackendGroup), q)
}

func (s HTTPBackendGroupEventHandler) Generic(event event.GenericEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Generic node event detected")

	s.Common(event.Object.(*v1alpha1.HttpBackendGroup), q)
}

func (s HTTPBackendGroupEventHandler) Common(ing *v1alpha1.HttpBackendGroup, q workqueue.RateLimitingInterface) {
	svcs := parseServicesFromBG(ing)
	for svc := range svcs {
		q.Add(ctrl.Request{NamespacedName: svc})
	}
}

func NewHTTPBackendGroupEventHandler(logger logr.Logger, cli client.Client) *HTTPBackendGroupEventHandler {
	return &HTTPBackendGroupEventHandler{Log: logger, cli: cli}
}

func parseServicesFromBG(ing *v1alpha1.HttpBackendGroup) map[types.NamespacedName]struct{} {
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
