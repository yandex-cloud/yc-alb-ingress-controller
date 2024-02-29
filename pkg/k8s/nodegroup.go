package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NodeGroup struct {
	Items []v1.Node
}

type NodeLoader struct {
	cli client.Client
}

func NewNodeGroupLoader(cli client.Client) *NodeLoader {
	return &NodeLoader{cli: cli}
}

func (l *NodeLoader) Load() (*NodeGroup, error) {
	ctx := context.Background()
	var list v1.NodeList
	err := l.cli.List(ctx, &list)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve nodes: %w ", err)
	}
	var items []v1.Node
	for _, item := range list.Items {
		if item.GetDeletionTimestamp().IsZero() {
			items = append(items, item)
		}
	}
	return &NodeGroup{Items: items}, nil
}
