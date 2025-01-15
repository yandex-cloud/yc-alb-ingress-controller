package builders

import "github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"

type BalancerOptions struct {
	NetworkID        string
	Locations        []*apploadbalancer.Location
	SecurityGroupIDs []string
	AutoScalePolicy  *apploadbalancer.AutoScalePolicy
}

type ListenerOptions struct {
	Addresses []*apploadbalancer.Address
}

type Options struct {
	BalancerOptions
	ListenerOptions
}
