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

func GetNodeInternalIPs(node *corev1.Node) ([]string, error) {
	var res []string
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			res = append(res, address.Address)
		}
	}

	return res, nil
}
