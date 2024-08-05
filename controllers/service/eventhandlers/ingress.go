package eventhandlers

import (
	"github.com/go-logr/logr"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/yandex-cloud/alb-ingress/pkg/algo"
)

type IngressEventHandler struct {
	Log logr.Logger
	cli client.Client
}

func (s IngressEventHandler) Create(event event.CreateEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service create event detected")

	s.Common(event.Object.(*networking.Ingress), q)
}

func (s IngressEventHandler) Update(event event.UpdateEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.ObjectNew.GetNamespace(),
		"name", event.ObjectNew.GetName()).
		Info("Service update event detected")

	oldServices := parseServicesFromIngress(event.ObjectOld.(*networking.Ingress))
	newServices := parseServicesFromIngress(event.ObjectNew.(*networking.Ingress))

	// trigger only inserted or removed services
	toUpdate := algo.SetsExceptUnion(oldServices, newServices)
	for svc := range toUpdate {
		q.Add(ctrl.Request{NamespacedName: svc})
	}
}

func (s IngressEventHandler) Delete(event event.DeleteEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service delete event detected")

	s.Common(event.Object.(*networking.Ingress), q)
}

func (s IngressEventHandler) Generic(event event.GenericEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Generic node event detected")

	s.Common(event.Object.(*networking.Ingress), q)
}

func (s IngressEventHandler) Common(ing *networking.Ingress, q workqueue.RateLimitingInterface) {
	svcs := parseServicesFromIngress(ing)
	for svc := range svcs {
		q.Add(ctrl.Request{NamespacedName: svc})
	}
}

func NewIngressEventHandler(logger logr.Logger, cli client.Client) *IngressEventHandler {
	return &IngressEventHandler{Log: logger, cli: cli}
}

func parseServicesFromIngress(ing *networking.Ingress) map[types.NamespacedName]struct{} {
	result := make(map[types.NamespacedName]struct{})

	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
		result[types.NamespacedName{
			Name:      ing.Spec.DefaultBackend.Service.Name,
			Namespace: ing.Namespace,
		}] = struct{}{}
	}

	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}

			result[types.NamespacedName{
				Name:      path.Backend.Service.Name,
				Namespace: ing.Namespace,
			}] = struct{}{}
		}
	}

	return result
}
