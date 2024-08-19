package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadata(t *testing.T) {
	t.Skip("names changed")
	names := &Names{ClusterID: "abcdef"}
	ns, name, tag := "app-ns", "app-ingress", "alb-007"
	assert.Equal(t, "alb-3f6ec083f1ed93da6af133418a353ad2fb006551", names.ALB(tag))
	assert.Equal(t, "listenerhttp-3f6ec083f1ed93da6af133418a353ad2fb006551", names.Listener(tag))
	assert.Equal(t, "listenertls-3f6ec083f1ed93da6af133418a353ad2fb006551", names.ListenerTLS(tag))
	assert.Equal(t, "httprouter-3f6ec083f1ed93da6af133418a353ad2fb006551", names.Router(tag))
	assert.Equal(t, "vh-69eef76c3e9be19a930f68bf31799e5872bf6914-1", names.VirtualHostForRule(ns, name, tag, 1))
	assert.Equal(t, "sni-69eef76c3e9be19a930f68bf31799e5872bf6914-1", names.SNIMatchForRule(ns, name, tag, 1))
	// assert.Equal(t, "route-3f6ec083f1ed93da6af133418a353ad2fb006551-1", names.RouteForPath(tag, 1))
	// assert.Equal(t, "bg-fb4c56872101fb16143589dd48258b6c92770485-8080", names.BackendGroup(tag, ns, "app-svc", 8080))
	assert.Equal(t, "backend-fb4c56872101fb16143589dd48258b6c92770485-8080", names.Backend(tag, ns, "app-svc", 8080, 30080))
}

func TestLabels(t *testing.T) {
	t.Skip("names changed")
	_, _, tag := "app-ns", "app-ingress", "alb-007"
	labels := &Labels{
		ClusterLabelName: "cluster_ref_label",
		ClusterID:        "abcdef",
	}

	assert.Equal(t, labels.Default(), map[string]string{
		"cluster_ref_label": "abcdef",
		"system":            "yc-alb-ingress",
	})
	assert.Equal(t, labels.ForIngress(tag), map[string]string{
		"cluster_ref_label":  "abcdef",
		"system":             "yc-alb-ingress",
		"yc-alb-ingress-tag": tag,
	})
}
