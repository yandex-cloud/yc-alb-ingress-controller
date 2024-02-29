package reconcile

import (
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
)

type IngressStatusResolver struct{}

func (r *IngressStatusResolver) Resolve(alb *apploadbalancer.LoadBalancer) networking.IngressStatus {
	var statusIngresses []networking.IngressLoadBalancerIngress
	statusMap := make(map[string]map[int32]struct{})
	for _, l := range alb.GetListeners() {
		for _, e := range l.Endpoints {
			for _, address := range e.Addresses {
				ip := extractIPFromAddress(address)
				if len(ip) == 0 {
					continue
				}
				statusEntry, ok := statusMap[ip]
				if !ok {
					statusEntry = make(map[int32]struct{})
					statusMap[ip] = statusEntry
				}
				for _, port := range e.Ports {
					statusEntry[int32(port)] = struct{}{}
				}
			}
		}
	}
	for ip, portSet := range statusMap {
		var ports []networking.IngressPortStatus
		for port := range portSet {
			ports = append(ports, networking.IngressPortStatus{Port: port, Protocol: v1.ProtocolTCP})
		}
		statusIngresses = append(statusIngresses, networking.IngressLoadBalancerIngress{
			IP:    ip,
			Ports: ports,
		})
	}
	return networking.IngressStatus{LoadBalancer: networking.IngressLoadBalancerStatus{Ingress: statusIngresses}}
}

func extractIPFromAddress(address *apploadbalancer.Address) (ip string) {
	switch t := address.Address.(type) {
	case *apploadbalancer.Address_ExternalIpv4Address:
		ip = t.ExternalIpv4Address.Address
	case *apploadbalancer.Address_InternalIpv4Address:
		ip = t.InternalIpv4Address.Address
	case *apploadbalancer.Address_ExternalIpv6Address:
		ip = t.ExternalIpv6Address.Address
	}
	return
}
