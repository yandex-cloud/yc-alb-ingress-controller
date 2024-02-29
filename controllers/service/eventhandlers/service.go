package eventhandlers

import (
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
)

type ServiceEventHandler struct {
	log logr.Logger
	cli client.Client
}

func (s ServiceEventHandler) Create(event event.CreateEvent, q workqueue.RateLimitingInterface) {
	s.log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service create event detected")

	s.Common(event.Object.(*v1.Service), q)
}

func (s ServiceEventHandler) Update(event event.UpdateEvent, q workqueue.RateLimitingInterface) {
	s.log.WithValues(
		"namespace", event.ObjectNew.GetNamespace(),
		"name", event.ObjectNew.GetName()).
		Info("Service update event detected")

	s.Common(event.ObjectNew.(*v1.Service), q)
}

func (s ServiceEventHandler) Delete(event event.DeleteEvent, q workqueue.RateLimitingInterface) {
	s.log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service delete event detected")

	s.Common(event.Object.(*v1.Service), q)
}

func (s ServiceEventHandler) Generic(event event.GenericEvent, q workqueue.RateLimitingInterface) {
	s.log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Generic node event detected")

	s.Common(event.Object.(*v1.Service), q)
}

func (s ServiceEventHandler) Common(svc *v1.Service, q workqueue.RateLimitingInterface) {
	q.Add(ctrl.Request{NamespacedName: k8s.NamespacedNameOf(svc)})
}

func NewServiceEventHandler(logger logr.Logger, cli client.Client) *ServiceEventHandler {
	return &ServiceEventHandler{log: logger, cli: cli}
}
