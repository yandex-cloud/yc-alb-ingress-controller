package yc

import (
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
)

func SubnetIDForProviderID(ip string, instance *compute.Instance) (string, error) {
	for _, networkInterface := range instance.NetworkInterfaces {
		ipv4 := networkInterface.PrimaryV4Address
		ipv6 := networkInterface.PrimaryV6Address
		if (ipv4 != nil && ipv4.Address == ip) || (ipv6 != nil && ipv6.Address == ip) {
			return networkInterface.SubnetId, nil
		}
	}
	return "", fmt.Errorf("internal: mismatch between node's address and instance network interfaces")
}
