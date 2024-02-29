package builders

import (
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"

	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
)

type BalancerBuilder struct {
	tag      string
	folderID string
	region   string
	names    *metadata.Names
	labels   *metadata.Labels
}

func (b *BalancerBuilder) Build(handler *apploadbalancer.HttpHandler, matches []*apploadbalancer.SniMatch, logOpts *apploadbalancer.LogOptions,
	opts Options) *apploadbalancer.LoadBalancer {
	return &apploadbalancer.LoadBalancer{
		FolderId:         b.folderID,
		Name:             b.names.ALB(b.tag),
		Description:      "ALB for ingresses with tag: " + b.tag,
		Labels:           b.labels.Default(),
		RegionId:         b.region,
		NetworkId:        opts.NetworkID,
		Listeners:        b.listenerSpecs(handler, matches, b.tag, opts),
		AllocationPolicy: &apploadbalancer.AllocationPolicy{Locations: opts.Locations},
		SecurityGroupIds: opts.SecurityGroupIDs,
		LogOptions:       logOpts,
	}
}

func (b *BalancerBuilder) listenerSpecs(handler *apploadbalancer.HttpHandler, matches []*apploadbalancer.SniMatch, tag string, opts Options) []*apploadbalancer.Listener {
	var ret []*apploadbalancer.Listener
	if handler != nil {
		ret = append(ret, &apploadbalancer.Listener{
			Name: b.names.Listener(tag),
			Endpoints: []*apploadbalancer.Endpoint{{
				Addresses: opts.Addresses,
				Ports:     []int64{80},
			}},
			Listener: &apploadbalancer.Listener_Http{
				Http: &apploadbalancer.HttpListener{
					Handler: handler,
				},
			},
		})
	}
	if len(matches) > 0 {
		ret = append(ret, &apploadbalancer.Listener{
			Name: b.names.ListenerTLS(tag),
			Endpoints: []*apploadbalancer.Endpoint{{
				Addresses: opts.Addresses,
				Ports:     []int64{443},
			}},
			Listener: &apploadbalancer.Listener_Tls{
				Tls: &apploadbalancer.TlsListener{
					DefaultHandler: matches[0].GetHandler(),
					SniHandlers:    matches,
				},
			},
		})
	}
	return ret
}
