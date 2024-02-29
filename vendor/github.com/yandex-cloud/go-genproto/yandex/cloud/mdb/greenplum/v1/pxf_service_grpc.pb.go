// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.17.3
// source: yandex/cloud/mdb/greenplum/v1/pxf_service.proto

package greenplum

import (
	context "context"
	operation "github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	PXFDatasourceService_List_FullMethodName   = "/yandex.cloud.mdb.greenplum.v1.PXFDatasourceService/List"
	PXFDatasourceService_Create_FullMethodName = "/yandex.cloud.mdb.greenplum.v1.PXFDatasourceService/Create"
	PXFDatasourceService_Update_FullMethodName = "/yandex.cloud.mdb.greenplum.v1.PXFDatasourceService/Update"
	PXFDatasourceService_Delete_FullMethodName = "/yandex.cloud.mdb.greenplum.v1.PXFDatasourceService/Delete"
)

// PXFDatasourceServiceClient is the client API for PXFDatasourceService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PXFDatasourceServiceClient interface {
	// List all PXF datasources
	List(ctx context.Context, in *ListPXFDatasourcesRequest, opts ...grpc.CallOption) (*ListPXFDatasourcesResponse, error)
	// Creates PXF datasource
	Create(ctx context.Context, in *CreatePXFDatasourceRequest, opts ...grpc.CallOption) (*operation.Operation, error)
	// Update PXF datasource
	Update(ctx context.Context, in *UpdatePXFDatasourceRequest, opts ...grpc.CallOption) (*operation.Operation, error)
	// Delete PXF datasource
	Delete(ctx context.Context, in *DeletePXFDatasourceRequest, opts ...grpc.CallOption) (*operation.Operation, error)
}

type pXFDatasourceServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPXFDatasourceServiceClient(cc grpc.ClientConnInterface) PXFDatasourceServiceClient {
	return &pXFDatasourceServiceClient{cc}
}

func (c *pXFDatasourceServiceClient) List(ctx context.Context, in *ListPXFDatasourcesRequest, opts ...grpc.CallOption) (*ListPXFDatasourcesResponse, error) {
	out := new(ListPXFDatasourcesResponse)
	err := c.cc.Invoke(ctx, PXFDatasourceService_List_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pXFDatasourceServiceClient) Create(ctx context.Context, in *CreatePXFDatasourceRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	out := new(operation.Operation)
	err := c.cc.Invoke(ctx, PXFDatasourceService_Create_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pXFDatasourceServiceClient) Update(ctx context.Context, in *UpdatePXFDatasourceRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	out := new(operation.Operation)
	err := c.cc.Invoke(ctx, PXFDatasourceService_Update_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pXFDatasourceServiceClient) Delete(ctx context.Context, in *DeletePXFDatasourceRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	out := new(operation.Operation)
	err := c.cc.Invoke(ctx, PXFDatasourceService_Delete_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PXFDatasourceServiceServer is the server API for PXFDatasourceService service.
// All implementations should embed UnimplementedPXFDatasourceServiceServer
// for forward compatibility
type PXFDatasourceServiceServer interface {
	// List all PXF datasources
	List(context.Context, *ListPXFDatasourcesRequest) (*ListPXFDatasourcesResponse, error)
	// Creates PXF datasource
	Create(context.Context, *CreatePXFDatasourceRequest) (*operation.Operation, error)
	// Update PXF datasource
	Update(context.Context, *UpdatePXFDatasourceRequest) (*operation.Operation, error)
	// Delete PXF datasource
	Delete(context.Context, *DeletePXFDatasourceRequest) (*operation.Operation, error)
}

// UnimplementedPXFDatasourceServiceServer should be embedded to have forward compatible implementations.
type UnimplementedPXFDatasourceServiceServer struct {
}

func (UnimplementedPXFDatasourceServiceServer) List(context.Context, *ListPXFDatasourcesRequest) (*ListPXFDatasourcesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedPXFDatasourceServiceServer) Create(context.Context, *CreatePXFDatasourceRequest) (*operation.Operation, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedPXFDatasourceServiceServer) Update(context.Context, *UpdatePXFDatasourceRequest) (*operation.Operation, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedPXFDatasourceServiceServer) Delete(context.Context, *DeletePXFDatasourceRequest) (*operation.Operation, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}

// UnsafePXFDatasourceServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PXFDatasourceServiceServer will
// result in compilation errors.
type UnsafePXFDatasourceServiceServer interface {
	mustEmbedUnimplementedPXFDatasourceServiceServer()
}

func RegisterPXFDatasourceServiceServer(s grpc.ServiceRegistrar, srv PXFDatasourceServiceServer) {
	s.RegisterService(&PXFDatasourceService_ServiceDesc, srv)
}

func _PXFDatasourceService_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListPXFDatasourcesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PXFDatasourceServiceServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PXFDatasourceService_List_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PXFDatasourceServiceServer).List(ctx, req.(*ListPXFDatasourcesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PXFDatasourceService_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreatePXFDatasourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PXFDatasourceServiceServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PXFDatasourceService_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PXFDatasourceServiceServer).Create(ctx, req.(*CreatePXFDatasourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PXFDatasourceService_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdatePXFDatasourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PXFDatasourceServiceServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PXFDatasourceService_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PXFDatasourceServiceServer).Update(ctx, req.(*UpdatePXFDatasourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PXFDatasourceService_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeletePXFDatasourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PXFDatasourceServiceServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PXFDatasourceService_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PXFDatasourceServiceServer).Delete(ctx, req.(*DeletePXFDatasourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// PXFDatasourceService_ServiceDesc is the grpc.ServiceDesc for PXFDatasourceService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PXFDatasourceService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "yandex.cloud.mdb.greenplum.v1.PXFDatasourceService",
	HandlerType: (*PXFDatasourceServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "List",
			Handler:    _PXFDatasourceService_List_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _PXFDatasourceService_Create_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _PXFDatasourceService_Update_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _PXFDatasourceService_Delete_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "yandex/cloud/mdb/greenplum/v1/pxf_service.proto",
}
