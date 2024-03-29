// Code generated by sdkgen. DO NOT EDIT.

// nolint
package compute

import (
	"context"

	"google.golang.org/grpc"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/access"
	compute "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
)

//revive:disable

// FilesystemServiceClient is a compute.FilesystemServiceClient with
// lazy GRPC connection initialization.
type FilesystemServiceClient struct {
	getConn func(ctx context.Context) (*grpc.ClientConn, error)
}

// Create implements compute.FilesystemServiceClient
func (c *FilesystemServiceClient) Create(ctx context.Context, in *compute.CreateFilesystemRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return compute.NewFilesystemServiceClient(conn).Create(ctx, in, opts...)
}

// Delete implements compute.FilesystemServiceClient
func (c *FilesystemServiceClient) Delete(ctx context.Context, in *compute.DeleteFilesystemRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return compute.NewFilesystemServiceClient(conn).Delete(ctx, in, opts...)
}

// Get implements compute.FilesystemServiceClient
func (c *FilesystemServiceClient) Get(ctx context.Context, in *compute.GetFilesystemRequest, opts ...grpc.CallOption) (*compute.Filesystem, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return compute.NewFilesystemServiceClient(conn).Get(ctx, in, opts...)
}

// List implements compute.FilesystemServiceClient
func (c *FilesystemServiceClient) List(ctx context.Context, in *compute.ListFilesystemsRequest, opts ...grpc.CallOption) (*compute.ListFilesystemsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return compute.NewFilesystemServiceClient(conn).List(ctx, in, opts...)
}

type FilesystemIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *FilesystemServiceClient
	request *compute.ListFilesystemsRequest

	items []*compute.Filesystem
}

func (c *FilesystemServiceClient) FilesystemIterator(ctx context.Context, req *compute.ListFilesystemsRequest, opts ...grpc.CallOption) *FilesystemIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &FilesystemIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *FilesystemIterator) Next() bool {
	if it.err != nil {
		return false
	}
	if len(it.items) > 1 {
		it.items[0] = nil
		it.items = it.items[1:]
		return true
	}
	it.items = nil // consume last item, if any

	if it.started && it.request.PageToken == "" {
		return false
	}
	it.started = true

	if it.requestedSize == 0 || it.requestedSize > it.pageSize {
		it.request.PageSize = it.pageSize
	} else {
		it.request.PageSize = it.requestedSize
	}

	response, err := it.client.List(it.ctx, it.request, it.opts...)
	it.err = err
	if err != nil {
		return false
	}

	it.items = response.Filesystems
	it.request.PageToken = response.NextPageToken
	return len(it.items) > 0
}

func (it *FilesystemIterator) Take(size int64) ([]*compute.Filesystem, error) {
	if it.err != nil {
		return nil, it.err
	}

	if size == 0 {
		size = 1 << 32 // something insanely large
	}
	it.requestedSize = size
	defer func() {
		// reset iterator for future calls.
		it.requestedSize = 0
	}()

	var result []*compute.Filesystem

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *FilesystemIterator) TakeAll() ([]*compute.Filesystem, error) {
	return it.Take(0)
}

func (it *FilesystemIterator) Value() *compute.Filesystem {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *FilesystemIterator) Error() error {
	return it.err
}

// ListAccessBindings implements compute.FilesystemServiceClient
func (c *FilesystemServiceClient) ListAccessBindings(ctx context.Context, in *access.ListAccessBindingsRequest, opts ...grpc.CallOption) (*access.ListAccessBindingsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return compute.NewFilesystemServiceClient(conn).ListAccessBindings(ctx, in, opts...)
}

type FilesystemAccessBindingsIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *FilesystemServiceClient
	request *access.ListAccessBindingsRequest

	items []*access.AccessBinding
}

func (c *FilesystemServiceClient) FilesystemAccessBindingsIterator(ctx context.Context, req *access.ListAccessBindingsRequest, opts ...grpc.CallOption) *FilesystemAccessBindingsIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &FilesystemAccessBindingsIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *FilesystemAccessBindingsIterator) Next() bool {
	if it.err != nil {
		return false
	}
	if len(it.items) > 1 {
		it.items[0] = nil
		it.items = it.items[1:]
		return true
	}
	it.items = nil // consume last item, if any

	if it.started && it.request.PageToken == "" {
		return false
	}
	it.started = true

	if it.requestedSize == 0 || it.requestedSize > it.pageSize {
		it.request.PageSize = it.pageSize
	} else {
		it.request.PageSize = it.requestedSize
	}

	response, err := it.client.ListAccessBindings(it.ctx, it.request, it.opts...)
	it.err = err
	if err != nil {
		return false
	}

	it.items = response.AccessBindings
	it.request.PageToken = response.NextPageToken
	return len(it.items) > 0
}

func (it *FilesystemAccessBindingsIterator) Take(size int64) ([]*access.AccessBinding, error) {
	if it.err != nil {
		return nil, it.err
	}

	if size == 0 {
		size = 1 << 32 // something insanely large
	}
	it.requestedSize = size
	defer func() {
		// reset iterator for future calls.
		it.requestedSize = 0
	}()

	var result []*access.AccessBinding

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *FilesystemAccessBindingsIterator) TakeAll() ([]*access.AccessBinding, error) {
	return it.Take(0)
}

func (it *FilesystemAccessBindingsIterator) Value() *access.AccessBinding {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *FilesystemAccessBindingsIterator) Error() error {
	return it.err
}

// ListOperations implements compute.FilesystemServiceClient
func (c *FilesystemServiceClient) ListOperations(ctx context.Context, in *compute.ListFilesystemOperationsRequest, opts ...grpc.CallOption) (*compute.ListFilesystemOperationsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return compute.NewFilesystemServiceClient(conn).ListOperations(ctx, in, opts...)
}

type FilesystemOperationsIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *FilesystemServiceClient
	request *compute.ListFilesystemOperationsRequest

	items []*operation.Operation
}

func (c *FilesystemServiceClient) FilesystemOperationsIterator(ctx context.Context, req *compute.ListFilesystemOperationsRequest, opts ...grpc.CallOption) *FilesystemOperationsIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &FilesystemOperationsIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *FilesystemOperationsIterator) Next() bool {
	if it.err != nil {
		return false
	}
	if len(it.items) > 1 {
		it.items[0] = nil
		it.items = it.items[1:]
		return true
	}
	it.items = nil // consume last item, if any

	if it.started && it.request.PageToken == "" {
		return false
	}
	it.started = true

	if it.requestedSize == 0 || it.requestedSize > it.pageSize {
		it.request.PageSize = it.pageSize
	} else {
		it.request.PageSize = it.requestedSize
	}

	response, err := it.client.ListOperations(it.ctx, it.request, it.opts...)
	it.err = err
	if err != nil {
		return false
	}

	it.items = response.Operations
	it.request.PageToken = response.NextPageToken
	return len(it.items) > 0
}

func (it *FilesystemOperationsIterator) Take(size int64) ([]*operation.Operation, error) {
	if it.err != nil {
		return nil, it.err
	}

	if size == 0 {
		size = 1 << 32 // something insanely large
	}
	it.requestedSize = size
	defer func() {
		// reset iterator for future calls.
		it.requestedSize = 0
	}()

	var result []*operation.Operation

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *FilesystemOperationsIterator) TakeAll() ([]*operation.Operation, error) {
	return it.Take(0)
}

func (it *FilesystemOperationsIterator) Value() *operation.Operation {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *FilesystemOperationsIterator) Error() error {
	return it.err
}

// SetAccessBindings implements compute.FilesystemServiceClient
func (c *FilesystemServiceClient) SetAccessBindings(ctx context.Context, in *access.SetAccessBindingsRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return compute.NewFilesystemServiceClient(conn).SetAccessBindings(ctx, in, opts...)
}

// Update implements compute.FilesystemServiceClient
func (c *FilesystemServiceClient) Update(ctx context.Context, in *compute.UpdateFilesystemRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return compute.NewFilesystemServiceClient(conn).Update(ctx, in, opts...)
}

// UpdateAccessBindings implements compute.FilesystemServiceClient
func (c *FilesystemServiceClient) UpdateAccessBindings(ctx context.Context, in *access.UpdateAccessBindingsRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return compute.NewFilesystemServiceClient(conn).UpdateAccessBindings(ctx, in, opts...)
}
