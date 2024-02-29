package yc

import (
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
)

func SubnetIDForProviderID(ip string, instance *compute.Instance) (string, error) {
	for _, networkInterface := range instance.NetworkInterfaces {
		if networkInterface.PrimaryV4Address.Address == ip || networkInterface.PrimaryV6Address.Address == ip {
			return instance.NetworkInterfaces[0].SubnetId, nil
		}
	}
	return "", fmt.Errorf("internal: mismatch between node's address and instance network interfaces")
}
