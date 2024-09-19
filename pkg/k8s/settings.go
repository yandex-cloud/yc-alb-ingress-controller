package k8s

import (
	"context"
	"fmt"

	"github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GroupSettingsLoader struct {
	Client client.Client
}

func (l *GroupSettingsLoader) Load(ctx context.Context, g *IngressGroup) (*v1alpha1.IngressGroupSettings, error) {
	name, err := getSettingsName(g)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings name: %w", err)
	}

	if name == "" {
		return nil, nil
	}

	var settings v1alpha1.IngressGroupSettings
	err = l.Client.Get(ctx, types.NamespacedName{Name: name}, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to get ingress group settings: %w", err)
	}

	return &settings, nil
}

func getSettingsName(g *IngressGroup) (string, error) {
	var res string

	for _, item := range g.Items {
		name, ok := item.Annotations[GroupSettings]
		if !ok || (res != "" && name == res) {
			continue
		}

		if name == "" {
			return "", fmt.Errorf("empty group settings annotation value in the ingress: %s/%s", item.Namespace, item.Name)
		}

		if res != "" {
			return "", fmt.Errorf("more than one settings name specified in the group: %s", g.Tag)
		}

		res = name
	}

	return res, nil
}
