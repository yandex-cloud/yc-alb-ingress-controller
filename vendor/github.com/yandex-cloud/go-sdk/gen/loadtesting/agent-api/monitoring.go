// Code generated by sdkgen. DO NOT EDIT.

// nolint
package agent

import (
	"context"

	"google.golang.org/grpc"

	agent "github.com/yandex-cloud/go-genproto/yandex/cloud/loadtesting/agent/v1"
)

//revive:disable

// MonitoringServiceClient is a agent.MonitoringServiceClient with
// lazy GRPC connection initialization.
type MonitoringServiceClient struct {
	getConn func(ctx context.Context) (*grpc.ClientConn, error)
}

// AddMetric implements agent.MonitoringServiceClient
func (c *MonitoringServiceClient) AddMetric(ctx context.Context, in *agent.AddMetricRequest, opts ...grpc.CallOption) (*agent.AddMetricResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}
	return agent.NewMonitoringServiceClient(conn).AddMetric(ctx, in, opts...)
}
