package eventhandlers

import (
	"github.com/go-logr/logr"
	discovery "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type EndpointSliceEventHandler struct {
	Log logr.Logger
	cli client.Client
}

func (s EndpointSliceEventHandler) Create(event event.CreateEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service create event detected")

	s.Common(event.Object.(*discovery.EndpointSlice), q)
}

func (s EndpointSliceEventHandler) Update(event event.UpdateEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.ObjectNew.GetNamespace(),
		"name", event.ObjectNew.GetName()).
		Info("Service update event detected")

	s.Common(event.ObjectNew.(*discovery.EndpointSlice), q)
}

func (s EndpointSliceEventHandler) Delete(event event.DeleteEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Service delete event detected")

	s.Common(event.Object.(*discovery.EndpointSlice), q)
}

func (s EndpointSliceEventHandler) Generic(event event.GenericEvent, q workqueue.RateLimitingInterface) {
	s.Log.WithValues(
		"namespace", event.Object.GetNamespace(),
		"name", event.Object.GetName()).
		Info("Generic node event detected")

	s.Common(event.Object.(*discovery.EndpointSlice), q)
}

func (s EndpointSliceEventHandler) Common(sl *discovery.EndpointSlice, q workqueue.RateLimitingInterface) {
	name := types.NamespacedName{
		Name:      sl.Labels["kubernetes.io/service-name"],
		Namespace: sl.Namespace,
	}

	q.Add(ctrl.Request{NamespacedName: name})
}

func NewEndpointSliceEventHandler(logger logr.Logger, cli client.Client) *EndpointSliceEventHandler {
	return &EndpointSliceEventHandler{Log: logger, cli: cli}
}
