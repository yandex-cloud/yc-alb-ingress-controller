package reconcile

import (
	"context"
	"fmt"
	"strings"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/k8s"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/metadata"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/yc"
)

type TargetGroupBuilder struct {
	folderID          string
	names             *metadata.Names
	labels            *metadata.Labels
	cli               client.Client
	getInstanceFn     func(context.Context, string) (*compute.Instance, error)
	useEndpointSlices bool
}

func NewTargetGroupBuilder(folderID string, cli client.Client, names *metadata.Names, labels *metadata.Labels,
	getInstanceFn func(context.Context, string) (*compute.Instance, error), useEndpointSlices bool,
) *TargetGroupBuilder {
	return &TargetGroupBuilder{
		folderID:          folderID,
		names:             names,
		cli:               cli,
		labels:            labels,
		getInstanceFn:     getInstanceFn,
		useEndpointSlices: useEndpointSlices,
	}
}

func (t *TargetGroupBuilder) Build(ctx context.Context, svc types.NamespacedName, locations []*apploadbalancer.Location) (*apploadbalancer.TargetGroup, error) {
	nodeNames, err := t.getServiceNodeNames(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("failed to get service node names: %w", err)
	}

	var k8ssvc v1.Service
	err = t.cli.Get(ctx, svc, &k8ssvc)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}
	preferIPv6 := k8ssvc.Annotations[k8s.PreferIPv6Targets] == "true"

	var suitableSubnets []string
	for _, location := range locations {
		suitableSubnets = append(suitableSubnets, location.SubnetId)
	}

	subnetsAnn := k8ssvc.Annotations[k8s.Subnets]
	if subnetsAnn != "" {
		suitableSubnets = strings.Split(subnetsAnn, ",")
	}

	var ret []*apploadbalancer.Target
	for _, nodeName := range nodeNames {
		var node v1.Node
		err = t.cli.Get(ctx, types.NamespacedName{
			Name: nodeName,
		}, &node)
		if err != nil {
			return nil, fmt.Errorf("failed to get node: %w", err)
		}

		instanceID, err := k8s.InstanceID(node)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance ID from node: %w", err)
		}
		instance, err := t.getInstanceFn(ctx, instanceID)
		if err != nil {
			// do not add a target if an underlying instance NOT FOUND
			// most likely the node is getting prepared for deletion
			if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
				continue
			}
			return nil, fmt.Errorf("failed to get instance fn: %w", err)
		}

		suitableIPs, err := k8s.GetNodeInternalIPs(&node)
		if err != nil {
			return nil, fmt.Errorf("failed to get node internal IPs: %w", err)
		}

		subnetID, ip, err := yc.SubnetIDForProviderID(instance, suitableIPs, suitableSubnets, preferIPv6)
		if err != nil {
			return nil, fmt.Errorf("failed to get subnet ID for provider ID %s of node %s: %w", node.Spec.ProviderID, node.Name, err)
		}

		ret = append(ret, &apploadbalancer.Target{
			AddressType: &apploadbalancer.Target_IpAddress{IpAddress: ip},
			SubnetId:    subnetID,
		})
	}

	return &apploadbalancer.TargetGroup{
		Name:        t.names.TargetGroup(svc),
		Description: "target group from K8S nodes",
		FolderId:    t.folderID,
		Labels:      t.labels.Default(),
		Targets:     ret,
	}, nil
}

func (t *TargetGroupBuilder) getServiceNodeNames(ctx context.Context, svc types.NamespacedName) ([]string, error) {
	if t.useEndpointSlices {
		return t.getServiceNodeNamesFromEndpointsSlice(ctx, svc)
	}

	return t.getServiceNodeNamesFromEndpoints(ctx, svc)
}

func (t *TargetGroupBuilder) getServiceNodeNamesFromEndpointsSlice(ctx context.Context, svc types.NamespacedName) ([]string, error) {
	var slList discovery.EndpointSliceList
	err := t.cli.List(ctx, &slList,
		client.MatchingLabels{
			"kubernetes.io/service-name": svc.Name,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoint slices: %w", err)
	}

	nodes := make([]string, 0)
	for _, sl := range slList.Items {
		if sl.Namespace != svc.Namespace {
			continue
		}

		for _, ep := range sl.Endpoints {
			if ep.NodeName == nil {
				continue
			}

			nodes = append(nodes, *ep.NodeName)
		}
	}

	return nodes, nil
}

func (t *TargetGroupBuilder) getServiceNodeNamesFromEndpoints(ctx context.Context, svc types.NamespacedName) ([]string, error) {
	var endpoints v1.Endpoints
	err := t.cli.Get(ctx, svc, &endpoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints: %w", err)
	}

	nodes := make([]string, 0)
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			if addr.NodeName == nil {
				continue
			}

			nodes = append(nodes, *addr.NodeName)
		}
	}

	return nodes, err
}
