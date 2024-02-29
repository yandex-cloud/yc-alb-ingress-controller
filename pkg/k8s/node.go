package k8s

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"

	errors2 "github.com/yandex-cloud/alb-ingress/pkg/errors"
)

func InstanceID(node v1.Node) (string, error) {
	providerID := node.Spec.ProviderID
	if len(providerID) == 0 {
		return "", errors2.ResourceNotReadyError{ResourceType: "Node", Name: node.Name}
	}
	parsedProviderID := strings.Split(providerID, "//")
	if len(parsedProviderID) != 2 {
		return "", fmt.Errorf("unsupported or missing providerID %s", providerID)
	}
	return parsedProviderID[1], nil
}
