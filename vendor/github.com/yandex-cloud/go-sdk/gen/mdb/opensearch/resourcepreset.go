// Code generated by sdkgen. DO NOT EDIT.

// nolint
package opensearch

import (
	"context"

	"google.golang.org/grpc"

	opensearch "github.com/yandex-cloud/go-genproto/yandex/cloud/mdb/opensearch/v1"
)

//revive:disable

// ResourcePresetServiceClient is a opensearch.ResourcePresetServiceClient with
// lazy GRPC connection initialization.
type ResourcePresetServiceClient struct {
	getConn func(ctx context.Context) (*grpc.ClientConn, error)
}

// Get implements opensearch.ResourcePresetServiceClient
func (c *ResourcePresetServiceClient) Get(ctx context.Context, in *opensearch.GetResourcePresetRequest, opts ...grpc.CallOption) (*opensearch.ResourcePreset, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return opensearch.NewResourcePresetServiceClient(conn).Get(ctx, in, opts...)
}

// List implements opensearch.ResourcePresetServiceClient
func (c *ResourcePresetServiceClient) List(ctx context.Context, in *opensearch.ListResourcePresetsRequest, opts ...grpc.CallOption) (*opensearch.ListResourcePresetsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return opensearch.NewResourcePresetServiceClient(conn).List(ctx, in, opts...)
}

type ResourcePresetIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ResourcePresetServiceClient
	request *opensearch.ListResourcePresetsRequest

	items []*opensearch.ResourcePreset
}

func (c *ResourcePresetServiceClient) ResourcePresetIterator(ctx context.Context, req *opensearch.ListResourcePresetsRequest, opts ...grpc.CallOption) *ResourcePresetIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ResourcePresetIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ResourcePresetIterator) Next() bool {
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

	it.items = response.ResourcePresets
	it.request.PageToken = response.NextPageToken
	return len(it.items) > 0
}

func (it *ResourcePresetIterator) Take(size int64) ([]*opensearch.ResourcePreset, error) {
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

	var result []*opensearch.ResourcePreset

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *ResourcePresetIterator) TakeAll() ([]*opensearch.ResourcePreset, error) {
	return it.Take(0)
}

func (it *ResourcePresetIterator) Value() *opensearch.ResourcePreset {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ResourcePresetIterator) Error() error {
	return it.err
}
