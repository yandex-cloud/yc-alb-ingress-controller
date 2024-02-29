package eventhandlers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type SettingsEventHandler struct {
	IngressLoader k8s.IngressLoader
	Logger        logr.Logger
}

func (s SettingsEventHandler) Create(event event.CreateEvent, limitingInterface workqueue.RateLimitingInterface) {
	s.Common(event.Object.(*v1alpha1.IngressGroupSettings), limitingInterface)
}

func (s SettingsEventHandler) Update(event event.UpdateEvent, limitingInterface workqueue.RateLimitingInterface) {
	s.Common(event.ObjectNew.(*v1alpha1.IngressGroupSettings), limitingInterface)
}

func (s SettingsEventHandler) Delete(event event.DeleteEvent, limitingInterface workqueue.RateLimitingInterface) {
	s.Common(event.Object.(*v1alpha1.IngressGroupSettings), limitingInterface)
}

func (s SettingsEventHandler) Generic(event event.GenericEvent, limitingInterface workqueue.RateLimitingInterface) {
	s.Common(event.Object.(*v1alpha1.IngressGroupSettings), limitingInterface)
}

func (s SettingsEventHandler) Common(settings *v1alpha1.IngressGroupSettings, q workqueue.RateLimitingInterface) {
	ings, err := s.IngressLoader.List(context.Background())
	if err != nil {
		s.Logger.Error(err, "error while listing ingresses")
	}

	// duplicate events are cut by the queue, so we don't need them to be unique
	for _, ing := range ings {
		if ing.GetAnnotations()[k8s.GroupSettings] == settings.Name {
			q.Add(ctrl.Request{NamespacedName: types.NamespacedName{
				Name: k8s.GetBalancerTag(&ing),
			}})
		}
	}
}

func NewSettingsEventHandler(logger logr.Logger, cli client.Client) *SettingsEventHandler {
	return &SettingsEventHandler{Logger: logger, IngressLoader: k8s.NewIngressLoader(cli)}
}
