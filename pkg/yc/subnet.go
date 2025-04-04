package yc

import (
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
)

func SubnetIDForProviderID(instance *compute.Instance, suitableIPs []string, suitableSubnets []string, preferIPv6 bool) (string, string, error) {
	subnets := make(map[string]struct{})
	for _, s := range suitableSubnets {
		subnets[s] = struct{}{}
	}

	var resIP string
	var resSubnetID string

	for _, networkInterface := range instance.NetworkInterfaces {
		ipv4 := networkInterface.PrimaryV4Address
		ipv6 := networkInterface.PrimaryV6Address
		if _, ok := subnets[networkInterface.SubnetId]; !ok {
			continue
		}

		for _, ip := range suitableIPs {
			if ipv6 != nil && ipv6.Address == ip {
				resIP = ip
				resSubnetID = networkInterface.SubnetId
				if preferIPv6 {
					return resSubnetID, resIP, nil
				}
			}
			if ipv4 != nil && ipv4.Address == ip {
				resIP = ip
				resSubnetID = networkInterface.SubnetId
			}
		}
	}

	if resIP != "" {
		return resSubnetID, resIP, nil
	}
	return "", "", fmt.Errorf("internal: mismatch between node's address and instance network interfaces: interfaces:%v, ips:%v, subnets:%v", instance.NetworkInterfaces, suitableIPs, suitableSubnets)
}
