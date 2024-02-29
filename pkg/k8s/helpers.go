package k8s

import (
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
)

func IsTLS(host string, tls []networking.IngressTLS) bool {
	for _, t := range tls {
		for _, h := range t.Hosts {
			if host == h {
				return true
			}
		}
	}
	return false
}

func GetNodeInternalIP(node *corev1.Node, preferIPv6 bool) (string, error) {

	res := ""
	for _, address := range node.Status.Addresses {
		if address.Type != corev1.NodeInternalIP {
			continue
		}

		if preferIPv6 && IsIPv6(address.Address) {
			return address.Address, nil
		}

		res = address.Address
	}

	return res, nil
}
