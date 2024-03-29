// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.17.3
// source: yandex/cloud/serverless/functions/v1/network_service.proto

package functions

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	NetworkService_GetUsed_FullMethodName                = "/yandex.cloud.serverless.functions.v1.NetworkService/GetUsed"
	NetworkService_ListUsed_FullMethodName               = "/yandex.cloud.serverless.functions.v1.NetworkService/ListUsed"
	NetworkService_ListConnectedResources_FullMethodName = "/yandex.cloud.serverless.functions.v1.NetworkService/ListConnectedResources"
	NetworkService_TriggerUsedCleanup_FullMethodName     = "/yandex.cloud.serverless.functions.v1.NetworkService/TriggerUsedCleanup"
)

// NetworkServiceClient is the client API for NetworkService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type NetworkServiceClient interface {
	// Returns the specified network used in serverless resources.
	GetUsed(ctx context.Context, in *GetUsedNetworkRequest, opts ...grpc.CallOption) (*UsedNetwork, error)
	// Retrieves the list of networks in the specified scope that are used in serverless resources.
	ListUsed(ctx context.Context, in *ListUsedNetworksRequest, opts ...grpc.CallOption) (*ListUsedNetworksResponse, error)
	// Retrieves the list of serverless resources connected to any network from the specified scope.
	ListConnectedResources(ctx context.Context, in *ListConnectedResourcesRequest, opts ...grpc.CallOption) (*ListConnectedResourcesResponse, error)
	// Forces obsolete used network to start cleanup process as soon as possible.
	// Invocation does not wait for start or end of the cleanup process.
	// Second invocation with the same network does nothing until network is completely cleaned-up.
	TriggerUsedCleanup(ctx context.Context, in *TriggerUsedNetworkCleanupRequest, opts ...grpc.CallOption) (*TriggerUsedNetworkCleanupResponse, error)
}

type networkServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewNetworkServiceClient(cc grpc.ClientConnInterface) NetworkServiceClient {
	return &networkServiceClient{cc}
}

func (c *networkServiceClient) GetUsed(ctx context.Context, in *GetUsedNetworkRequest, opts ...grpc.CallOption) (*UsedNetwork, error) {
	out := new(UsedNetwork)
	err := c.cc.Invoke(ctx, NetworkService_GetUsed_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *networkServiceClient) ListUsed(ctx context.Context, in *ListUsedNetworksRequest, opts ...grpc.CallOption) (*ListUsedNetworksResponse, error) {
	out := new(ListUsedNetworksResponse)
	err := c.cc.Invoke(ctx, NetworkService_ListUsed_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *networkServiceClient) ListConnectedResources(ctx context.Context, in *ListConnectedResourcesRequest, opts ...grpc.CallOption) (*ListConnectedResourcesResponse, error) {
	out := new(ListConnectedResourcesResponse)
	err := c.cc.Invoke(ctx, NetworkService_ListConnectedResources_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *networkServiceClient) TriggerUsedCleanup(ctx context.Context, in *TriggerUsedNetworkCleanupRequest, opts ...grpc.CallOption) (*TriggerUsedNetworkCleanupResponse, error) {
	out := new(TriggerUsedNetworkCleanupResponse)
	err := c.cc.Invoke(ctx, NetworkService_TriggerUsedCleanup_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NetworkServiceServer is the server API for NetworkService service.
// All implementations should embed UnimplementedNetworkServiceServer
// for forward compatibility
type NetworkServiceServer interface {
	// Returns the specified network used in serverless resources.
	GetUsed(context.Context, *GetUsedNetworkRequest) (*UsedNetwork, error)
	// Retrieves the list of networks in the specified scope that are used in serverless resources.
	ListUsed(context.Context, *ListUsedNetworksRequest) (*ListUsedNetworksResponse, error)
	// Retrieves the list of serverless resources connected to any network from the specified scope.
	ListConnectedResources(context.Context, *ListConnectedResourcesRequest) (*ListConnectedResourcesResponse, error)
	// Forces obsolete used network to start cleanup process as soon as possible.
	// Invocation does not wait for start or end of the cleanup process.
	// Second invocation with the same network does nothing until network is completely cleaned-up.
	TriggerUsedCleanup(context.Context, *TriggerUsedNetworkCleanupRequest) (*TriggerUsedNetworkCleanupResponse, error)
}

// UnimplementedNetworkServiceServer should be embedded to have forward compatible implementations.
type UnimplementedNetworkServiceServer struct {
}

func (UnimplementedNetworkServiceServer) GetUsed(context.Context, *GetUsedNetworkRequest) (*UsedNetwork, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUsed not implemented")
}
func (UnimplementedNetworkServiceServer) ListUsed(context.Context, *ListUsedNetworksRequest) (*ListUsedNetworksResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUsed not implemented")
}
func (UnimplementedNetworkServiceServer) ListConnectedResources(context.Context, *ListConnectedResourcesRequest) (*ListConnectedResourcesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListConnectedResources not implemented")
}
func (UnimplementedNetworkServiceServer) TriggerUsedCleanup(context.Context, *TriggerUsedNetworkCleanupRequest) (*TriggerUsedNetworkCleanupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TriggerUsedCleanup not implemented")
}

// UnsafeNetworkServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to NetworkServiceServer will
// result in compilation errors.
type UnsafeNetworkServiceServer interface {
	mustEmbedUnimplementedNetworkServiceServer()
}

func RegisterNetworkServiceServer(s grpc.ServiceRegistrar, srv NetworkServiceServer) {
	s.RegisterService(&NetworkService_ServiceDesc, srv)
}

func _NetworkService_GetUsed_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUsedNetworkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NetworkServiceServer).GetUsed(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NetworkService_GetUsed_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NetworkServiceServer).GetUsed(ctx, req.(*GetUsedNetworkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NetworkService_ListUsed_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUsedNetworksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NetworkServiceServer).ListUsed(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NetworkService_ListUsed_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NetworkServiceServer).ListUsed(ctx, req.(*ListUsedNetworksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NetworkService_ListConnectedResources_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListConnectedResourcesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NetworkServiceServer).ListConnectedResources(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NetworkService_ListConnectedResources_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NetworkServiceServer).ListConnectedResources(ctx, req.(*ListConnectedResourcesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NetworkService_TriggerUsedCleanup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TriggerUsedNetworkCleanupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NetworkServiceServer).TriggerUsedCleanup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NetworkService_TriggerUsedCleanup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NetworkServiceServer).TriggerUsedCleanup(ctx, req.(*TriggerUsedNetworkCleanupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// NetworkService_ServiceDesc is the grpc.ServiceDesc for NetworkService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var NetworkService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "yandex.cloud.serverless.functions.v1.NetworkService",
	HandlerType: (*NetworkServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetUsed",
			Handler:    _NetworkService_GetUsed_Handler,
		},
		{
			MethodName: "ListUsed",
			Handler:    _NetworkService_ListUsed_Handler,
		},
		{
			MethodName: "ListConnectedResources",
			Handler:    _NetworkService_ListConnectedResources_Handler,
		},
		{
			MethodName: "TriggerUsedCleanup",
			Handler:    _NetworkService_TriggerUsedCleanup_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "yandex/cloud/serverless/functions/v1/network_service.proto",
}
