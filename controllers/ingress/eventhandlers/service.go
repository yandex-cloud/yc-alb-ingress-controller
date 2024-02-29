package eventhandlers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
)

const ServiceEventReasonFailedToListIngresses = "ServiceEventFailedToListIngresses"

type ServiceEventHandler struct {
	Log logr.Logger
	k8s.IngressLoader
	record.EventRecorder
}

func (s ServiceEventHandler) Create(event event.CreateEvent, limitingInterface workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service event detected")
	s.Common(event.Object.(*v1.Service), limitingInterface)
}

func (s ServiceEventHandler) Update(event event.UpdateEvent, limitingInterface workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.ObjectNew.GetNamespace(),
		"name", event.ObjectNew.GetName()).
		Info("Service event detected")

	s.Common(event.ObjectNew.(*v1.Service), limitingInterface)
}

func (s ServiceEventHandler) Delete(event event.DeleteEvent, limitingInterface workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service event detected")
	s.Common(event.Object.(*v1.Service), limitingInterface)
}

func (s ServiceEventHandler) Generic(event event.GenericEvent, limitingInterface workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service event detected")

	s.Common(event.Object.(*v1.Service), limitingInterface)
}

func (s ServiceEventHandler) Common(svc *v1.Service, q workqueue.RateLimitingInterface) {
	ings, err := s.IngressLoader.ListBySvc(context.Background(), *svc)
	if err != nil {
		s.EventRecorder.Event(svc, v1.EventTypeWarning, ServiceEventReasonFailedToListIngresses, fmt.Sprintf("failed to list ingresses due %e", err))
	}

	// duplicate events are cut by the queue, so we don't need them to be unique
	for _, ing := range ings {
		q.Add(ctrl.Request{NamespacedName: types.NamespacedName{
			Name: k8s.GetBalancerTag(&ing),
		}})
	}
}

func NewServiceEventHandler(logger logr.Logger, cli client.Client, eventRecorder record.EventRecorder) *ServiceEventHandler {
	return &ServiceEventHandler{Log: logger, IngressLoader: k8s.NewIngressLoader(cli), EventRecorder: eventRecorder}
}
