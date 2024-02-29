package builders

import (
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
)

//go:generate mockgen -destination=./mocks/builders.go -package=mocks . TargetGroupFinder

type DummyIDGenerator int

func (g *DummyIDGenerator) Next() int {
	*g = *g + 1
	return int(*g)
}

type Factory struct {
	folderID string
	region   string
	names    *metadata.Names
	labels   *metadata.Labels
	cli      client.Client
	routeIDs interface{ Next() int }
	hostIDs  interface{ Next() int }

	targetGroupFinder TargetGroupFinder
}

func NewFactory(folderID string, region string, names *metadata.Names, labels *metadata.Labels, cli client.Client, tgRepo TargetGroupFinder) *Factory {
	return &Factory{
		folderID:          folderID,
		region:            region,
		names:             names,
		labels:            labels,
		cli:               cli,
		targetGroupFinder: tgRepo,
	}
}

func (f *Factory) BackendGroupForCRDBuilder() *BackendGroupForCRDBuilder {
	return &BackendGroupForCRDBuilder{
		tag:        "",
		folderID:   f.folderID,
		names:      f.names,
		labels:     f.labels,
		cli:        f.cli,
		seenSvc:    make(map[exposedNodePort]struct{}),
		seenBucket: make(map[string]struct{}),

		targetGroupFinder: f.targetGroupFinder,
	}
}

func (f *Factory) RestartVirtualHostIDGenerator() {
	routeIDs, vhIDs := DummyIDGenerator(-1), DummyIDGenerator(-1)
	f.routeIDs = &routeIDs
	f.hostIDs = &vhIDs
}

func (f *Factory) VirtualHostBuilder(tag string, backendGroupFinder BackendGroupFinder) *VirtualHostBuilder {
	return &VirtualHostBuilder{
		tag:      tag,
		folderID: f.folderID,
		names:    f.names,
		labels:   f.labels,

		nextRouteID: f.routeIDs,
		nextVHID:    f.hostIDs,

		httpRouteMap: make(map[HostAndPath]*apploadbalancer.Route),
		hosts:        make(map[string]HostInfo),

		backendGroupFinder: backendGroupFinder,
	}
}

func (f *Factory) TLSVirtualHostBuilder(tag string, backendGroupFinder BackendGroupFinder) *VirtualHostBuilder {
	ret := f.VirtualHostBuilder(tag, backendGroupFinder)
	ret.isTLS = true
	return ret
}

func (f *Factory) HandlerBuilder(tag string) *HandlerBuilder {
	return &HandlerBuilder{
		tag:   tag,
		names: f.names,

		certs:        make(map[string][]string),
		hostAndCerts: make(map[hostAndCert]struct{}),
	}
}

func (f *Factory) BalancerBuilder(tag string) *BalancerBuilder {
	return &BalancerBuilder{
		tag:      tag,
		folderID: f.folderID,
		region:   f.region,
		names:    f.names,
		labels:   f.labels,
	}
}
