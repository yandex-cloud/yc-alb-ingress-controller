package builders

import (
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
)

type Data struct {
	TargetGroups  []*apploadbalancer.TargetGroup
	BackendGroups *BackendGroups
	HTTPRouter    *HTTPRouterData
	TLSRouter     *HTTPRouterData
	Handler       *apploadbalancer.HttpHandler
	SNIMatches    []*apploadbalancer.SniMatch
	Balancer      *apploadbalancer.LoadBalancer
	LogOptions    *apploadbalancer.LogOptions
}

/* TODO: the injection of IDs after deployment is ugly
possible design improvement options:
- lazy build implementation instead of building all resources before deploy, so that IDs of dependencies are
  available for dependent resources when building the latter
- build deployment tree
- deployable resource with ID setting callback
*/

func (d *Data) InjectTLSRouterIDIntoSNIMatches(id string) {
	if d != nil {
		for _, sniMatch := range d.SNIMatches {
			sniMatch.Handler.GetHttpHandler().HttpRouterId = id
		}
	}
}

func (d *Data) InjectRouterIDIntoHandler(id string) {
	if d != nil {
		d.Handler.HttpRouterId = id
	}
}
