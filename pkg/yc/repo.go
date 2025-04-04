package yc

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/pkg/errors"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/vpc/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/sdkresolvers"

	ycsdkerrors "github.com/yandex-cloud/yc-alb-ingress-controller/pkg/errors"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/metadata"
)

type Repository struct {
	sdk      *ycsdk.SDK
	names    *metadata.Names
	folderID string
}

func (r *Repository) FindSubnetByID(ctx context.Context, id string) (*vpc.Subnet, error) {
	return r.sdk.VPC().Subnet().Get(ctx, &vpc.GetSubnetRequest{
		SubnetId: id,
	})
}

func (r *Repository) ListSubnetsByNetworkID(ctx context.Context, id string) ([]*vpc.Subnet, error) {
	resp, err := r.sdk.VPC().Network().ListSubnets(ctx, &vpc.ListNetworkSubnetsRequest{
		NetworkId: id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list subnets: %w", err)
	}
	return resp.Subnets, nil
}

func NewRepository(sdk *ycsdk.SDK, names *metadata.Names, folderID string) *Repository {
	return &Repository{
		sdk:      sdk,
		names:    names,
		folderID: folderID,
	}
}

func (r *Repository) CreateBackendGroup(ctx context.Context, group *apploadbalancer.BackendGroup) (*operation.Operation, error) {
	var b apploadbalancer.CreateBackendGroupRequest_Backend
	switch {
	case group.GetHttp() != nil:
		b = &apploadbalancer.CreateBackendGroupRequest_Http{
			Http: group.GetHttp(),
		}
	case group.GetGrpc() != nil:
		b = &apploadbalancer.CreateBackendGroupRequest_Grpc{
			Grpc: group.GetGrpc(),
		}
	default:
		return nil, fmt.Errorf("unsupported type of backend group %s", group.GetName())
	}
	return r.sdk.ApplicationLoadBalancer().BackendGroup().Create(ctx, &apploadbalancer.CreateBackendGroupRequest{
		FolderId:    group.FolderId,
		Name:        group.Name,
		Description: group.Description,
		Labels:      group.Labels,
		Backend:     b,
	})
}

func (r *Repository) UpdateBackendGroup(ctx context.Context, group *apploadbalancer.BackendGroup) (*operation.Operation, error) {
	var updateMask fieldmaskpb.FieldMask
	var b apploadbalancer.UpdateBackendGroupRequest_Backend
	switch {
	case group.GetHttp() != nil:
		b = &apploadbalancer.UpdateBackendGroupRequest_Http{
			Http: group.GetHttp(),
		}
		updateMask.Paths = append(updateMask.Paths, "http")
	case group.GetGrpc() != nil:
		b = &apploadbalancer.UpdateBackendGroupRequest_Grpc{
			Grpc: group.GetGrpc(),
		}
		updateMask.Paths = append(updateMask.Paths, "grpc")
	default:
		return nil, fmt.Errorf("unsupported type of backend group %s", group.GetName())
	}
	return r.sdk.ApplicationLoadBalancer().BackendGroup().Update(ctx, &apploadbalancer.UpdateBackendGroupRequest{
		BackendGroupId: group.Id,
		Backend:        b,
		UpdateMask:     &updateMask,
	})
}

func (r *Repository) RenameBackendGroup(ctx context.Context, groupID string, newName string) (*operation.Operation, error) {
	return r.sdk.ApplicationLoadBalancer().BackendGroup().Update(ctx, &apploadbalancer.UpdateBackendGroupRequest{
		BackendGroupId: groupID,
		Name:           newName,
		UpdateMask:     &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	})
}

func (r *Repository) ListBackendGroupOperations(ctx context.Context, group *apploadbalancer.BackendGroup) ([]*operation.Operation, error) {
	resp, err := r.sdk.ApplicationLoadBalancer().BackendGroup().ListOperations(ctx, &apploadbalancer.ListBackendGroupOperationsRequest{
		BackendGroupId: group.Id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list operations for group %s: %w", group.Id, err)
	}

	return filterIncompleteOperations(resp.Operations), nil
}

func (r *Repository) DeleteBackendGroup(ctx context.Context, group *apploadbalancer.BackendGroup) (*operation.Operation, error) {
	return r.sdk.ApplicationLoadBalancer().BackendGroup().Delete(ctx, &apploadbalancer.DeleteBackendGroupRequest{
		BackendGroupId: group.Id,
	})
}

func (r *Repository) DeleteBackendGroups(ctx context.Context, groups []*apploadbalancer.BackendGroup) error {
	var lastOp *operation.Operation
	var lastErr error
	for _, bg := range groups {
		op, err := r.sdk.ApplicationLoadBalancer().BackendGroup().Delete(ctx, &apploadbalancer.DeleteBackendGroupRequest{BackendGroupId: bg.Id})
		if err != nil {
			// TODO:handle
			lastErr = err
			continue
		}
		lastOp = op
	}
	if lastOp != nil {
		return ycsdkerrors.OperationIncompleteError{ID: lastOp.Id}
	}
	if lastErr != nil {
		return lastErr
	}
	return nil
}

func (r *Repository) CreateHTTPRouter(ctx context.Context, router *apploadbalancer.HttpRouter) (*operation.Operation, error) {
	return r.sdk.ApplicationLoadBalancer().HttpRouter().Create(ctx, &apploadbalancer.CreateHttpRouterRequest{
		FolderId:     router.FolderId,
		Name:         router.Name,
		Description:  router.Description,
		Labels:       router.Labels,
		VirtualHosts: router.VirtualHosts,
	})
}

func (r *Repository) UpdateHTTPRouter(ctx context.Context, router *apploadbalancer.HttpRouter) (*operation.Operation, error) {
	return r.sdk.ApplicationLoadBalancer().HttpRouter().Update(ctx, &apploadbalancer.UpdateHttpRouterRequest{
		HttpRouterId: router.Id,
		VirtualHosts: router.VirtualHosts,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{
			"virtual_hosts",
		}},
	})
}

func (r *Repository) ListHTTPRouterIncompleteOperations(ctx context.Context, router *apploadbalancer.HttpRouter) ([]*operation.Operation, error) {
	resp, err := r.sdk.ApplicationLoadBalancer().HttpRouter().ListOperations(ctx, &apploadbalancer.ListHttpRouterOperationsRequest{
		HttpRouterId: router.Id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list operations for router %s: %w", router.Id, err)
	}

	return filterIncompleteOperations(resp.Operations), nil
}

func (r *Repository) DeleteHTTPRouter(ctx context.Context, router *apploadbalancer.HttpRouter) (*operation.Operation, error) {
	return r.sdk.ApplicationLoadBalancer().HttpRouter().Delete(ctx, &apploadbalancer.DeleteHttpRouterRequest{
		HttpRouterId: router.Id,
	})
}

func (r *Repository) CreateLoadBalancer(ctx context.Context, balancer *apploadbalancer.LoadBalancer) (*operation.Operation, error) {
	return r.sdk.ApplicationLoadBalancer().LoadBalancer().Create(ctx, &apploadbalancer.CreateLoadBalancerRequest{
		FolderId:         balancer.FolderId,
		Name:             balancer.Name,
		Description:      balancer.Description,
		Labels:           balancer.Labels,
		RegionId:         balancer.RegionId,
		NetworkId:        balancer.NetworkId,
		ListenerSpecs:    listenerSpecs(balancer.Listeners),
		AllocationPolicy: balancer.AllocationPolicy,
		AutoScalePolicy:  balancer.AutoScalePolicy,
		SecurityGroupIds: balancer.SecurityGroupIds,
		LogOptions:       balancer.LogOptions,
	})
}

func listenerSpecs(listeners []*apploadbalancer.Listener) []*apploadbalancer.ListenerSpec {
	ret := make([]*apploadbalancer.ListenerSpec, len(listeners))
	for i, l := range listeners {
		ret[i] = &apploadbalancer.ListenerSpec{
			Name:          l.Name,
			EndpointSpecs: endpointSpecs(l.Endpoints),
			Listener:      listenerSpecListener(l.Listener),
		}
	}
	return ret
}

func addressSpecs(addresses []*apploadbalancer.Address) []*apploadbalancer.AddressSpec {
	ret := make([]*apploadbalancer.AddressSpec, len(addresses))
	for i, address := range addresses {
		var spec apploadbalancer.AddressSpec_AddressSpec
		switch a := address.Address.(type) {
		case *apploadbalancer.Address_ExternalIpv4Address:
			spec = &apploadbalancer.AddressSpec_ExternalIpv4AddressSpec{ExternalIpv4AddressSpec: &apploadbalancer.ExternalIpv4AddressSpec{
				Address: a.ExternalIpv4Address.Address,
			}}
		case *apploadbalancer.Address_InternalIpv4Address:
			spec = &apploadbalancer.AddressSpec_InternalIpv4AddressSpec{InternalIpv4AddressSpec: &apploadbalancer.InternalIpv4AddressSpec{
				Address:  a.InternalIpv4Address.Address,
				SubnetId: a.InternalIpv4Address.SubnetId,
			}}
		case *apploadbalancer.Address_ExternalIpv6Address:
			spec = &apploadbalancer.AddressSpec_ExternalIpv6AddressSpec{ExternalIpv6AddressSpec: &apploadbalancer.ExternalIpv6AddressSpec{
				Address: a.ExternalIpv6Address.Address,
			}}
		}
		ret[i] = &apploadbalancer.AddressSpec{AddressSpec: spec}
	}
	return ret
}

func listenerSpecListener(listener apploadbalancer.Listener_Listener) apploadbalancer.ListenerSpec_Listener {
	var ret apploadbalancer.ListenerSpec_Listener
	switch l := listener.(type) {
	case *apploadbalancer.Listener_Http:
		ret = &apploadbalancer.ListenerSpec_Http{Http: l.Http}
	case *apploadbalancer.Listener_Tls:
		ret = &apploadbalancer.ListenerSpec_Tls{Tls: l.Tls}
	}
	return ret
}

func endpointSpecs(endpoints []*apploadbalancer.Endpoint) []*apploadbalancer.EndpointSpec {
	ret := make([]*apploadbalancer.EndpointSpec, len(endpoints))
	for i, e := range endpoints {
		ret[i] = &apploadbalancer.EndpointSpec{
			AddressSpecs: addressSpecs(e.Addresses),
			Ports:        e.Ports,
		}
	}
	return ret
}

func (r *Repository) UpdateLoadBalancer(ctx context.Context, balancer *apploadbalancer.LoadBalancer) (*operation.Operation, error) {
	return r.sdk.ApplicationLoadBalancer().LoadBalancer().Update(ctx, &apploadbalancer.UpdateLoadBalancerRequest{
		LoadBalancerId: balancer.Id,

		ListenerSpecs:    listenerSpecs(balancer.Listeners),
		AllocationPolicy: balancer.AllocationPolicy,
		SecurityGroupIds: balancer.SecurityGroupIds,
		LogOptions:       balancer.LogOptions,
		AutoScalePolicy:  balancer.AutoScalePolicy,

		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"listener_specs", "allocation_policy", "security_group_ids", "log_options", "auto_scale_policy"},
		},
	})
}

func (r *Repository) ListLoadBalancerIncompleteOperations(ctx context.Context, balancer *apploadbalancer.LoadBalancer) ([]*operation.Operation, error) {
	operations, err := r.sdk.ApplicationLoadBalancer().LoadBalancer().ListOperations(ctx, &apploadbalancer.ListLoadBalancerOperationsRequest{
		LoadBalancerId: balancer.Id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list operations for load balancer %s: %w", balancer.Id, err)
	}

	return filterIncompleteOperations(operations.Operations), nil
}

func (r *Repository) DeleteLoadBalancer(ctx context.Context, balancer *apploadbalancer.LoadBalancer) (*operation.Operation, error) {
	return r.sdk.ApplicationLoadBalancer().LoadBalancer().Delete(ctx, &apploadbalancer.DeleteLoadBalancerRequest{
		LoadBalancerId: balancer.Id,
	})
}

func (r *Repository) findBalancer(ctx context.Context, tag string) (*apploadbalancer.LoadBalancer, error) {
	resp, err := r.sdk.ApplicationLoadBalancer().LoadBalancer().List(ctx, &apploadbalancer.ListLoadBalancersRequest{
		FolderId: r.folderID,
		Filter:   sdkresolvers.CreateResolverFilter("name", r.names.ALB(tag)),
		PageSize: sdkresolvers.DefaultResolverPageSize,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.LoadBalancers) == 0 {
		return nil, nil
	}
	return resp.LoadBalancers[0], nil
}

func (r *Repository) findHTTPRouter(ctx context.Context, tag string) (*apploadbalancer.HttpRouter, error) {
	resp, err := r.sdk.ApplicationLoadBalancer().HttpRouter().List(ctx, &apploadbalancer.ListHttpRoutersRequest{
		FolderId: r.folderID,
		Filter:   sdkresolvers.CreateResolverFilter("name", r.names.Router(tag)),
		PageSize: sdkresolvers.DefaultResolverPageSize,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.HttpRouters) == 0 {
		return nil, nil
	}
	return resp.HttpRouters[0], nil
}

func (r *Repository) findTLSRouter(ctx context.Context, tag string) (*apploadbalancer.HttpRouter, error) {
	resp, err := r.sdk.ApplicationLoadBalancer().HttpRouter().List(ctx, &apploadbalancer.ListHttpRoutersRequest{
		FolderId: r.folderID,
		Filter:   sdkresolvers.CreateResolverFilter("name", r.names.RouterTLS(tag)),
		PageSize: sdkresolvers.DefaultResolverPageSize,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.HttpRouters) == 0 {
		return nil, nil
	}
	return resp.HttpRouters[0], nil
}

// FindBackendGroups find all backend groups for balancer tagged with the provided tag
func (r *Repository) FindBackendGroups(ctx context.Context, tag string) ([]*apploadbalancer.BackendGroup, error) {
	var ret []*apploadbalancer.BackendGroup
	it := r.sdk.ApplicationLoadBalancer().BackendGroup().BackendGroupIterator(ctx, &apploadbalancer.ListBackendGroupsRequest{
		FolderId: r.folderID,
	})
	for it.Next() {
		v := it.Value()
		if err := it.Error(); err != nil {
			return nil, err
		}
		if v.Labels["yc-alb-ingress-tag"] == tag {
			ret = append(ret, v)
		}
	}
	return ret, nil
}

func (r *Repository) FindTargetGroup(ctx context.Context, name string) (*apploadbalancer.TargetGroup, error) {
	resp, err := r.sdk.ApplicationLoadBalancer().TargetGroup().List(ctx, &apploadbalancer.ListTargetGroupsRequest{
		FolderId: r.folderID,
		Filter:   sdkresolvers.CreateResolverFilter("name", name),
		PageSize: sdkresolvers.DefaultResolverPageSize,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.TargetGroups) == 0 {
		return nil, nil
	}
	return resp.TargetGroups[0], nil
}

type BalancerResources struct {
	Balancer  *apploadbalancer.LoadBalancer
	Router    *apploadbalancer.HttpRouter
	TLSRouter *apploadbalancer.HttpRouter
}

func (r *Repository) FindAllResources(ctx context.Context, tag string) (*BalancerResources, error) {
	ret := &BalancerResources{}
	var err error
	if ret.Balancer, err = r.findBalancer(ctx, tag); err != nil {
		return nil, err
	}
	if ret.Router, err = r.findHTTPRouter(ctx, tag); err != nil {
		return nil, err
	}
	if ret.TLSRouter, err = r.findTLSRouter(ctx, tag); err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *Repository) DeleteAllResources(ctx context.Context, b *BalancerResources) error {
	if err := r.DeleteBalancer(ctx, b.Balancer); err != nil {
		return err
	}
	if err := r.DeleteRouters(ctx, []*apploadbalancer.HttpRouter{b.Router, b.TLSRouter}); err != nil {
		return err
	}
	return nil
}

func (r *Repository) DeleteBalancer(ctx context.Context, balancer *apploadbalancer.LoadBalancer) error {
	if balancer == nil {
		return nil
	}
	op, err := r.sdk.ApplicationLoadBalancer().LoadBalancer().Delete(ctx, &apploadbalancer.DeleteLoadBalancerRequest{LoadBalancerId: balancer.Id})
	if err != nil {
		return errors.Wrapf(err, "failed to delete balancer %s", balancer.Id)
	}
	return ycsdkerrors.OperationIncompleteError{ID: op.Id}
}

func (r *Repository) DeleteRouters(ctx context.Context, routers []*apploadbalancer.HttpRouter) error {
	var lastOp *operation.Operation
	var lastErr error
	for _, router := range routers {
		if router == nil {
			continue
		}
		op, err := r.sdk.ApplicationLoadBalancer().HttpRouter().Delete(ctx, &apploadbalancer.DeleteHttpRouterRequest{HttpRouterId: router.Id})
		if err != nil {
			// TODO:handle
			lastErr = err
			continue
		}
		lastOp = op
	}
	if lastOp != nil {
		return ycsdkerrors.OperationIncompleteError{ID: lastOp.Id}
	}
	if lastErr != nil {
		return lastErr
	}
	return nil
}

func (r *Repository) DeleteTargetGroup(ctx context.Context, group *apploadbalancer.TargetGroup) error {
	op, err := r.sdk.ApplicationLoadBalancer().TargetGroup().Delete(ctx, &apploadbalancer.DeleteTargetGroupRequest{
		TargetGroupId: group.Id,
	})
	if err != nil {
		return fmt.Errorf("failed to delete target group %s: %w", group.Id, err)
	}
	return ycsdkerrors.OperationIncompleteError{ID: op.Id}
}

func (r *Repository) CreateTargetGroup(ctx context.Context, group *apploadbalancer.TargetGroup) (*operation.Operation, error) {
	return r.sdk.ApplicationLoadBalancer().TargetGroup().Create(ctx, &apploadbalancer.CreateTargetGroupRequest{
		FolderId:    r.folderID,
		Name:        group.Name,
		Description: group.Description,
		Labels:      group.Labels,
		Targets:     group.Targets,
	})
}

func (r *Repository) UpdateTargetGroup(ctx context.Context, group *apploadbalancer.TargetGroup) (*operation.Operation, error) {
	return r.sdk.ApplicationLoadBalancer().TargetGroup().Update(ctx, &apploadbalancer.UpdateTargetGroupRequest{
		TargetGroupId: group.Id,
		Name:          group.Name,
		Description:   group.Description,
		Labels:        group.Labels,
		Targets:       group.Targets,
	})
}

func (r *Repository) ListTargetGroupIncompleteOperations(ctx context.Context, group *apploadbalancer.TargetGroup) ([]*operation.Operation, error) {
	resp, err := r.sdk.ApplicationLoadBalancer().TargetGroup().ListOperations(ctx, &apploadbalancer.ListTargetGroupOperationsRequest{
		TargetGroupId: group.Id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list operations for target group %s: %w", group.Id, err)
	}

	return filterIncompleteOperations(resp.Operations), nil
}

func (r *Repository) FindBackendGroup(ctx context.Context, name string) (*apploadbalancer.BackendGroup, error) {
	resp, err := r.sdk.ApplicationLoadBalancer().BackendGroup().List(ctx, &apploadbalancer.ListBackendGroupsRequest{
		FolderId: r.folderID,
		Filter:   sdkresolvers.CreateResolverFilter("name", name),
		PageSize: sdkresolvers.DefaultResolverPageSize,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.BackendGroups) == 0 {
		return nil, nil
	}
	return resp.BackendGroups[0], nil
}

func (r *Repository) FindBackendGroupByCR(ctx context.Context, ns, name string) (*apploadbalancer.BackendGroup, error) {
	return r.FindBackendGroup(ctx, r.names.BackendGroupForCR(ns, name))
}

func (r *Repository) FindInstanceByID(ctx context.Context, id string) (*compute.Instance, error) {
	return r.sdk.Compute().Instance().Get(ctx, &compute.GetInstanceRequest{
		InstanceId: id,
	})
}

func filterIncompleteOperations(ops []*operation.Operation) []*operation.Operation {
	result := make([]*operation.Operation, 0)
	for _, op := range ops {
		if !op.Done {
			result = append(result, op)
		}
	}
	return result
}
