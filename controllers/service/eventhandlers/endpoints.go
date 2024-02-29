package eventhandlers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
)

type EndpointsEventHandler struct {
	Log logr.Logger
	cli client.Client
	record.EventRecorder
}

func (s EndpointsEventHandler) Create(event event.CreateEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service create event detected")

	s.Common(event.Object.(*core.Endpoints), q)
}

func (s EndpointsEventHandler) Update(event event.UpdateEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.ObjectNew.GetNamespace(),
		"name", event.ObjectNew.GetName()).
		Info("Service update event detected")

	s.Common(event.ObjectNew.(*core.Endpoints), q)
}

func (s EndpointsEventHandler) Delete(event event.DeleteEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service delete event detected")

	s.Common(event.Object.(*core.Endpoints), q)
}

func (s EndpointsEventHandler) Generic(event event.GenericEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Generic node event detected")

	s.Common(event.Object.(*core.Endpoints), q)
}

func (s EndpointsEventHandler) Common(sl *core.Endpoints, q workqueue.RateLimitingInterface) {
	var svc core.Service
	err := s.cli.Get(context.Background(), k8s.NamespacedNameOf(sl), &svc)
	if errors.IsNotFound(err) {
		// this is not services endpoints
		return
	}
	if err != nil {
		s.EventRecorder.Event(sl, core.EventTypeWarning, "FailedToGetService", fmt.Sprintf("failed to get service due %e", err))
		return
	}

	q.Add(ctrl.Request{NamespacedName: k8s.NamespacedNameOf(sl)})
}

func NewEndpointsEventHandler(logger logr.Logger, cli client.Client, eventRecorder record.EventRecorder) *EndpointsEventHandler {
	return &EndpointsEventHandler{Log: logger, cli: cli, EventRecorder: eventRecorder}
}
