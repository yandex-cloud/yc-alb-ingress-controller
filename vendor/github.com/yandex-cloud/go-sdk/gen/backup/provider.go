// Code generated by sdkgen. DO NOT EDIT.

// nolint
package backup

import (
	"context"

	"google.golang.org/grpc"

	backup "github.com/yandex-cloud/go-genproto/yandex/cloud/backup/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
)

//revive:disable

// ProviderServiceClient is a backup.ProviderServiceClient with
// lazy GRPC connection initialization.
type ProviderServiceClient struct {
	getConn func(ctx context.Context) (*grpc.ClientConn, error)
}

// Activate implements backup.ProviderServiceClient
func (c *ProviderServiceClient) Activate(ctx context.Context, in *backup.ActivateProviderRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return backup.NewProviderServiceClient(conn).Activate(ctx, in, opts...)
}

// ListActivated implements backup.ProviderServiceClient
func (c *ProviderServiceClient) ListActivated(ctx context.Context, in *backup.ListActivatedProvidersRequest, opts ...grpc.CallOption) (*backup.ListActivatedProvidersResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return backup.NewProviderServiceClient(conn).ListActivated(ctx, in, opts...)
}

type ProviderActivatedIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ProviderServiceClient
	request *backup.ListActivatedProvidersRequest

	items []string
}

func (c *ProviderServiceClient) ProviderActivatedIterator(ctx context.Context, req *backup.ListActivatedProvidersRequest, opts ...grpc.CallOption) *ProviderActivatedIterator {
	var pageSize int64
	const defaultPageSize = 1000

	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ProviderActivatedIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ProviderActivatedIterator) Next() bool {
	if it.err != nil {
		return false
	}
	if len(it.items) > 1 {
		it.items = it.items[1:]
		return true
	}
	it.items = nil // consume last item, if any

	if it.started {
		return false
	}
	it.started = true

	response, err := it.client.ListActivated(it.ctx, it.request, it.opts...)
	it.err = err
	if err != nil {
		return false
	}

	it.items = response.Names
	return len(it.items) > 0
}

func (it *ProviderActivatedIterator) Take(size int64) ([]string, error) {
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

	var result []string

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *ProviderActivatedIterator) TakeAll() ([]string, error) {
	return it.Take(0)
}

func (it *ProviderActivatedIterator) Value() string {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ProviderActivatedIterator) Error() error {
	return it.err
}
