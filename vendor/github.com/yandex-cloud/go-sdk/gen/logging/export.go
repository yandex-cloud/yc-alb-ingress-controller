// Code generated by sdkgen. DO NOT EDIT.

// nolint
package logging

import (
	"context"

	"google.golang.org/grpc"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/access"
	logging "github.com/yandex-cloud/go-genproto/yandex/cloud/logging/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
)

//revive:disable

// ExportServiceClient is a logging.ExportServiceClient with
// lazy GRPC connection initialization.
type ExportServiceClient struct {
	getConn func(ctx context.Context) (*grpc.ClientConn, error)
}

// Create implements logging.ExportServiceClient
func (c *ExportServiceClient) Create(ctx context.Context, in *logging.CreateExportRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return logging.NewExportServiceClient(conn).Create(ctx, in, opts...)
}

// Delete implements logging.ExportServiceClient
func (c *ExportServiceClient) Delete(ctx context.Context, in *logging.DeleteExportRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return logging.NewExportServiceClient(conn).Delete(ctx, in, opts...)
}

// Get implements logging.ExportServiceClient
func (c *ExportServiceClient) Get(ctx context.Context, in *logging.GetExportRequest, opts ...grpc.CallOption) (*logging.Export, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return logging.NewExportServiceClient(conn).Get(ctx, in, opts...)
}

// List implements logging.ExportServiceClient
func (c *ExportServiceClient) List(ctx context.Context, in *logging.ListExportsRequest, opts ...grpc.CallOption) (*logging.ListExportsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return logging.NewExportServiceClient(conn).List(ctx, in, opts...)
}

type ExportIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ExportServiceClient
	request *logging.ListExportsRequest

	items []*logging.Export
}

func (c *ExportServiceClient) ExportIterator(ctx context.Context, req *logging.ListExportsRequest, opts ...grpc.CallOption) *ExportIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ExportIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ExportIterator) Next() bool {
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

	it.items = response.Exports
	it.request.PageToken = response.NextPageToken
	return len(it.items) > 0
}

func (it *ExportIterator) Take(size int64) ([]*logging.Export, error) {
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

	var result []*logging.Export

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *ExportIterator) TakeAll() ([]*logging.Export, error) {
	return it.Take(0)
}

func (it *ExportIterator) Value() *logging.Export {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ExportIterator) Error() error {
	return it.err
}

// ListAccessBindings implements logging.ExportServiceClient
func (c *ExportServiceClient) ListAccessBindings(ctx context.Context, in *access.ListAccessBindingsRequest, opts ...grpc.CallOption) (*access.ListAccessBindingsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return logging.NewExportServiceClient(conn).ListAccessBindings(ctx, in, opts...)
}

type ExportAccessBindingsIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ExportServiceClient
	request *access.ListAccessBindingsRequest

	items []*access.AccessBinding
}

func (c *ExportServiceClient) ExportAccessBindingsIterator(ctx context.Context, req *access.ListAccessBindingsRequest, opts ...grpc.CallOption) *ExportAccessBindingsIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ExportAccessBindingsIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ExportAccessBindingsIterator) Next() bool {
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

func (it *ExportAccessBindingsIterator) Take(size int64) ([]*access.AccessBinding, error) {
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

func (it *ExportAccessBindingsIterator) TakeAll() ([]*access.AccessBinding, error) {
	return it.Take(0)
}

func (it *ExportAccessBindingsIterator) Value() *access.AccessBinding {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ExportAccessBindingsIterator) Error() error {
	return it.err
}

// ListOperations implements logging.ExportServiceClient
func (c *ExportServiceClient) ListOperations(ctx context.Context, in *logging.ListExportOperationsRequest, opts ...grpc.CallOption) (*logging.ListExportOperationsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return logging.NewExportServiceClient(conn).ListOperations(ctx, in, opts...)
}

type ExportOperationsIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ExportServiceClient
	request *logging.ListExportOperationsRequest

	items []*operation.Operation
}

func (c *ExportServiceClient) ExportOperationsIterator(ctx context.Context, req *logging.ListExportOperationsRequest, opts ...grpc.CallOption) *ExportOperationsIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ExportOperationsIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ExportOperationsIterator) Next() bool {
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

func (it *ExportOperationsIterator) Take(size int64) ([]*operation.Operation, error) {
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

func (it *ExportOperationsIterator) TakeAll() ([]*operation.Operation, error) {
	return it.Take(0)
}

func (it *ExportOperationsIterator) Value() *operation.Operation {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ExportOperationsIterator) Error() error {
	return it.err
}

// Run implements logging.ExportServiceClient
func (c *ExportServiceClient) Run(ctx context.Context, in *logging.RunExportRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return logging.NewExportServiceClient(conn).Run(ctx, in, opts...)
}

// SetAccessBindings implements logging.ExportServiceClient
func (c *ExportServiceClient) SetAccessBindings(ctx context.Context, in *access.SetAccessBindingsRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return logging.NewExportServiceClient(conn).SetAccessBindings(ctx, in, opts...)
}

// Update implements logging.ExportServiceClient
func (c *ExportServiceClient) Update(ctx context.Context, in *logging.UpdateExportRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return logging.NewExportServiceClient(conn).Update(ctx, in, opts...)
}

// UpdateAccessBindings implements logging.ExportServiceClient
func (c *ExportServiceClient) UpdateAccessBindings(ctx context.Context, in *access.UpdateAccessBindingsRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return logging.NewExportServiceClient(conn).UpdateAccessBindings(ctx, in, opts...)
}
