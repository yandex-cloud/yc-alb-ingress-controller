// Code generated by sdkgen. DO NOT EDIT.

// nolint
package greenplum

import (
	"context"

	"google.golang.org/grpc"

	greenplum "github.com/yandex-cloud/go-genproto/yandex/cloud/mdb/greenplum/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
)

//revive:disable

// ClusterServiceClient is a greenplum.ClusterServiceClient with
// lazy GRPC connection initialization.
type ClusterServiceClient struct {
	getConn func(ctx context.Context) (*grpc.ClientConn, error)
}

// Backup implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) Backup(ctx context.Context, in *greenplum.BackupClusterRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).Backup(ctx, in, opts...)
}

// Create implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) Create(ctx context.Context, in *greenplum.CreateClusterRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).Create(ctx, in, opts...)
}

// Delete implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) Delete(ctx context.Context, in *greenplum.DeleteClusterRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).Delete(ctx, in, opts...)
}

// Expand implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) Expand(ctx context.Context, in *greenplum.ExpandRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).Expand(ctx, in, opts...)
}

// Get implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) Get(ctx context.Context, in *greenplum.GetClusterRequest, opts ...grpc.CallOption) (*greenplum.Cluster, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).Get(ctx, in, opts...)
}

// List implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) List(ctx context.Context, in *greenplum.ListClustersRequest, opts ...grpc.CallOption) (*greenplum.ListClustersResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).List(ctx, in, opts...)
}

type ClusterIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ClusterServiceClient
	request *greenplum.ListClustersRequest

	items []*greenplum.Cluster
}

func (c *ClusterServiceClient) ClusterIterator(ctx context.Context, req *greenplum.ListClustersRequest, opts ...grpc.CallOption) *ClusterIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ClusterIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ClusterIterator) Next() bool {
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

	it.items = response.Clusters
	it.request.PageToken = response.NextPageToken
	return len(it.items) > 0
}

func (it *ClusterIterator) Take(size int64) ([]*greenplum.Cluster, error) {
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

	var result []*greenplum.Cluster

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *ClusterIterator) TakeAll() ([]*greenplum.Cluster, error) {
	return it.Take(0)
}

func (it *ClusterIterator) Value() *greenplum.Cluster {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ClusterIterator) Error() error {
	return it.err
}

// ListBackups implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) ListBackups(ctx context.Context, in *greenplum.ListClusterBackupsRequest, opts ...grpc.CallOption) (*greenplum.ListClusterBackupsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).ListBackups(ctx, in, opts...)
}

type ClusterBackupsIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ClusterServiceClient
	request *greenplum.ListClusterBackupsRequest

	items []*greenplum.Backup
}

func (c *ClusterServiceClient) ClusterBackupsIterator(ctx context.Context, req *greenplum.ListClusterBackupsRequest, opts ...grpc.CallOption) *ClusterBackupsIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ClusterBackupsIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ClusterBackupsIterator) Next() bool {
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

	response, err := it.client.ListBackups(it.ctx, it.request, it.opts...)
	it.err = err
	if err != nil {
		return false
	}

	it.items = response.Backups
	it.request.PageToken = response.NextPageToken
	return len(it.items) > 0
}

func (it *ClusterBackupsIterator) Take(size int64) ([]*greenplum.Backup, error) {
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

	var result []*greenplum.Backup

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *ClusterBackupsIterator) TakeAll() ([]*greenplum.Backup, error) {
	return it.Take(0)
}

func (it *ClusterBackupsIterator) Value() *greenplum.Backup {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ClusterBackupsIterator) Error() error {
	return it.err
}

// ListLogs implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) ListLogs(ctx context.Context, in *greenplum.ListClusterLogsRequest, opts ...grpc.CallOption) (*greenplum.ListClusterLogsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).ListLogs(ctx, in, opts...)
}

type ClusterLogsIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ClusterServiceClient
	request *greenplum.ListClusterLogsRequest

	items []*greenplum.LogRecord
}

func (c *ClusterServiceClient) ClusterLogsIterator(ctx context.Context, req *greenplum.ListClusterLogsRequest, opts ...grpc.CallOption) *ClusterLogsIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ClusterLogsIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ClusterLogsIterator) Next() bool {
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

	response, err := it.client.ListLogs(it.ctx, it.request, it.opts...)
	it.err = err
	if err != nil {
		return false
	}

	it.items = response.Logs
	it.request.PageToken = response.NextPageToken
	return len(it.items) > 0
}

func (it *ClusterLogsIterator) Take(size int64) ([]*greenplum.LogRecord, error) {
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

	var result []*greenplum.LogRecord

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *ClusterLogsIterator) TakeAll() ([]*greenplum.LogRecord, error) {
	return it.Take(0)
}

func (it *ClusterLogsIterator) Value() *greenplum.LogRecord {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ClusterLogsIterator) Error() error {
	return it.err
}

// ListMasterHosts implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) ListMasterHosts(ctx context.Context, in *greenplum.ListClusterHostsRequest, opts ...grpc.CallOption) (*greenplum.ListClusterHostsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).ListMasterHosts(ctx, in, opts...)
}

type ClusterMasterHostsIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ClusterServiceClient
	request *greenplum.ListClusterHostsRequest

	items []*greenplum.Host
}

func (c *ClusterServiceClient) ClusterMasterHostsIterator(ctx context.Context, req *greenplum.ListClusterHostsRequest, opts ...grpc.CallOption) *ClusterMasterHostsIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ClusterMasterHostsIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ClusterMasterHostsIterator) Next() bool {
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

	response, err := it.client.ListMasterHosts(it.ctx, it.request, it.opts...)
	it.err = err
	if err != nil {
		return false
	}

	it.items = response.Hosts
	it.request.PageToken = response.NextPageToken
	return len(it.items) > 0
}

func (it *ClusterMasterHostsIterator) Take(size int64) ([]*greenplum.Host, error) {
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

	var result []*greenplum.Host

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *ClusterMasterHostsIterator) TakeAll() ([]*greenplum.Host, error) {
	return it.Take(0)
}

func (it *ClusterMasterHostsIterator) Value() *greenplum.Host {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ClusterMasterHostsIterator) Error() error {
	return it.err
}

// ListOperations implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) ListOperations(ctx context.Context, in *greenplum.ListClusterOperationsRequest, opts ...grpc.CallOption) (*greenplum.ListClusterOperationsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).ListOperations(ctx, in, opts...)
}

type ClusterOperationsIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ClusterServiceClient
	request *greenplum.ListClusterOperationsRequest

	items []*operation.Operation
}

func (c *ClusterServiceClient) ClusterOperationsIterator(ctx context.Context, req *greenplum.ListClusterOperationsRequest, opts ...grpc.CallOption) *ClusterOperationsIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ClusterOperationsIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ClusterOperationsIterator) Next() bool {
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

func (it *ClusterOperationsIterator) Take(size int64) ([]*operation.Operation, error) {
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

func (it *ClusterOperationsIterator) TakeAll() ([]*operation.Operation, error) {
	return it.Take(0)
}

func (it *ClusterOperationsIterator) Value() *operation.Operation {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ClusterOperationsIterator) Error() error {
	return it.err
}

// ListSegmentHosts implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) ListSegmentHosts(ctx context.Context, in *greenplum.ListClusterHostsRequest, opts ...grpc.CallOption) (*greenplum.ListClusterHostsResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).ListSegmentHosts(ctx, in, opts...)
}

type ClusterSegmentHostsIterator struct {
	ctx  context.Context
	opts []grpc.CallOption

	err           error
	started       bool
	requestedSize int64
	pageSize      int64

	client  *ClusterServiceClient
	request *greenplum.ListClusterHostsRequest

	items []*greenplum.Host
}

func (c *ClusterServiceClient) ClusterSegmentHostsIterator(ctx context.Context, req *greenplum.ListClusterHostsRequest, opts ...grpc.CallOption) *ClusterSegmentHostsIterator {
	var pageSize int64
	const defaultPageSize = 1000
	pageSize = req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return &ClusterSegmentHostsIterator{
		ctx:      ctx,
		opts:     opts,
		client:   c,
		request:  req,
		pageSize: pageSize,
	}
}

func (it *ClusterSegmentHostsIterator) Next() bool {
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

	response, err := it.client.ListSegmentHosts(it.ctx, it.request, it.opts...)
	it.err = err
	if err != nil {
		return false
	}

	it.items = response.Hosts
	it.request.PageToken = response.NextPageToken
	return len(it.items) > 0
}

func (it *ClusterSegmentHostsIterator) Take(size int64) ([]*greenplum.Host, error) {
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

	var result []*greenplum.Host

	for it.requestedSize > 0 && it.Next() {
		it.requestedSize--
		result = append(result, it.Value())
	}

	if it.err != nil {
		return nil, it.err
	}

	return result, nil
}

func (it *ClusterSegmentHostsIterator) TakeAll() ([]*greenplum.Host, error) {
	return it.Take(0)
}

func (it *ClusterSegmentHostsIterator) Value() *greenplum.Host {
	if len(it.items) == 0 {
		panic("calling Value on empty iterator")
	}
	return it.items[0]
}

func (it *ClusterSegmentHostsIterator) Error() error {
	return it.err
}

// Restore implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) Restore(ctx context.Context, in *greenplum.RestoreClusterRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).Restore(ctx, in, opts...)
}

// Start implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) Start(ctx context.Context, in *greenplum.StartClusterRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).Start(ctx, in, opts...)
}

// Stop implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) Stop(ctx context.Context, in *greenplum.StopClusterRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).Stop(ctx, in, opts...)
}

// StreamLogs implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) StreamLogs(ctx context.Context, in *greenplum.StreamClusterLogsRequest, opts ...grpc.CallOption) (greenplum.ClusterService_StreamLogsClient, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).StreamLogs(ctx, in, opts...)
}

// Update implements greenplum.ClusterServiceClient
func (c *ClusterServiceClient) Update(ctx context.Context, in *greenplum.UpdateClusterRequest, opts ...grpc.CallOption) (*operation.Operation, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return greenplum.NewClusterServiceClient(conn).Update(ctx, in, opts...)
}
