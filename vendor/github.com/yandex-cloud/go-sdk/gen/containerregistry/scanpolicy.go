// Code generated by sdkgen. DO NOT EDIT.

// nolint
package containerregistry

import (
	"context"

	"google.golang.org/grpc"

	containerregistry "github.com/yandex-cloud/go-genproto/yandex/cloud/containerregistry/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
)

//revive:disable

// ScanPolicyServiceClient is a containerregistry.ScanPolicyServiceClient with
// lazy GRPC connection initialization.
type ScanPolicyServiceClient struct {
	getConn func(ctx context.Context) (*grpc.ClientConn, error)
}

// Create implements containerregistry.ScanPolicyServiceClient
func (c *ScanPolicyServiceClient) Create(ctx context.Context, in *containerregistry.CreateScanPolicyRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return containerregistry.NewScanPolicyServiceClient(conn).Create(ctx, in, opts...)
}

// Delete implements containerregistry.ScanPolicyServiceClient
func (c *ScanPolicyServiceClient) Delete(ctx context.Context, in *containerregistry.DeleteScanPolicyRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return containerregistry.NewScanPolicyServiceClient(conn).Delete(ctx, in, opts...)
}

// Get implements containerregistry.ScanPolicyServiceClient
func (c *ScanPolicyServiceClient) Get(ctx context.Context, in *containerregistry.GetScanPolicyRequest, opts ...grpc.CallOption) (*containerregistry.ScanPolicy, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return containerregistry.NewScanPolicyServiceClient(conn).Get(ctx, in, opts...)
}

// GetByRegistry implements containerregistry.ScanPolicyServiceClient
func (c *ScanPolicyServiceClient) GetByRegistry(ctx context.Context, in *containerregistry.GetScanPolicyByRegistryRequest, opts ...grpc.CallOption) (*containerregistry.ScanPolicy, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return containerregistry.NewScanPolicyServiceClient(conn).GetByRegistry(ctx, in, opts...)
}

// Update implements containerregistry.ScanPolicyServiceClient
func (c *ScanPolicyServiceClient) Update(ctx context.Context, in *containerregistry.UpdateScanPolicyRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return containerregistry.NewScanPolicyServiceClient(conn).Update(ctx, in, opts...)
}
