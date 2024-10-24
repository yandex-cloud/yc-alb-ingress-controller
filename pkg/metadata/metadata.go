package metadata

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
)

const (
	prefix = "yc-alb-ingress"
)

// Names naming service for Application Load Balancer resources.
// Resources will be created with the provided names and retrieved by them.
// For now assume Ingresses do not share resources except for TargetGroup and Certificate
type Names struct {
	ClusterID string
}

func (n *Names) sha(tag string) []byte {
	s := sha1.Sum([]byte(fmt.Sprintf("%s-%s-%s", prefix, n.ClusterID, tag)))
	return s[:]
}

func (n *Names) ALB(tag string) string {
	return fmt.Sprintf("%s-%x", "alb", n.sha(tag))
}

func (n *Names) Router(tag string) string {
	return fmt.Sprintf("%s-%x", "httprouter", n.sha(tag))
}

func (n *Names) RouterTLS(tag string) string {
	return fmt.Sprintf("%s-%x", "tlsrouter", n.sha(tag))
}

func (n *Names) Listener(tag string) string {
	return fmt.Sprintf("%s-%x", "listenerhttp", n.sha(tag))
}

func (n *Names) ListenerTLS(tag string) string {
	return fmt.Sprintf("%s-%x", "listenertls", n.sha(tag))
}

func (n *Names) VirtualHostForRule(ns, name, tag string, i int) string {
	return fmt.Sprintf("%s-%x-%d", "vh", n.sha(fmt.Sprintf("%s-%s-%s", ns, name, tag)), i)
}

func (n *Names) VirtualHostForID(tag string, i int) string {
	return fmt.Sprintf("%s-%x-%d", "vh", n.sha(tag), i)
}

func (n *Names) IsVirtualHostForIngress(ns string, name string, tag string, vhName string) bool {
	return strings.HasPrefix(vhName, fmt.Sprintf("%s-%x", "vh", n.sha(fmt.Sprintf("%s-%s-%s", ns, name, tag))))
}

func (n *Names) SNIMatchForRule(ns, name, tag string, i int) string {
	return fmt.Sprintf("%s-%x-%d", "sni", n.sha(fmt.Sprintf("%s-%s-%s", ns, name, tag)), i)
}

func (n *Names) SNIMatchForHost(tag string, host string) string {
	return fmt.Sprintf("%s-%x", "sni", n.sha(fmt.Sprintf("%s-%s", tag, host)))
}

func (n *Names) IsSNIForIngress(ns, name, tag, sniName string) bool {
	return strings.HasPrefix(sniName, fmt.Sprintf("%s-%x", "sni", n.sha(fmt.Sprintf("%s-%s-%s", ns, name, tag))))
}

// Legacy naming for backward compatibility
func (n *Names) RouteForPath(tag string, host, path, pathtype string) string {
	return fmt.Sprintf("%s-%x", "route", n.sha(fmt.Sprintf("%s-%s-%s-%s", host, path, pathtype, tag)))
}

func (n *Names) RouteForPath2(tag string, host, path, pathtype string, i int) string {
	return fmt.Sprintf("%s-%x-%d", "route", n.sha(fmt.Sprintf("%s-%s-%s-%s", host, path, pathtype, tag)), i)
}

func (n *Names) TargetGroup(name types.NamespacedName) string {
	return fmt.Sprintf("%s-%x", "tg", n.sha(fmt.Sprintf("%s-%s-%s", name.Namespace, name.Name, n.ClusterID)))
}

func (n *Names) BackendGroup(tag, host, path, pathtype string) string {
	return fmt.Sprintf("%s-%x", "bg", n.sha(fmt.Sprintf("%s-%s-%s-%s", tag, host, path, pathtype)))
}

func (n *Names) NewBackendGroup(name types.NamespacedName) string {
	return fmt.Sprintf("%s-%x", "bg", n.sha(fmt.Sprintf("%s-%s-%s", name.Namespace, name.Name, n.ClusterID)))
}

func (n *Names) Backend(tag, ns, svcName string, port, nodePort int32) string {
	return fmt.Sprintf("%s-%x-%d-%d", "backend", n.sha(fmt.Sprintf("%s-%s-%s", ns, svcName, tag)), port, nodePort)
}

func (n *Names) BackendGroupForCR(ns, name string) string {
	return fmt.Sprintf("%s-%x", "bg-cr", n.sha(fmt.Sprintf("%s-%s", ns, name)))
}

func (n *Names) Certificate(name types.NamespacedName) string {
	return fmt.Sprintf("%s-%x", "cert", n.sha(fmt.Sprintf("%s-%s", name.Namespace, name.Name)))
}

// TODO:builder
type Labels struct {
	ClusterLabelName, ClusterID string
}

func (l *Labels) Default() map[string]string {
	return map[string]string{
		"system":           prefix,
		l.ClusterLabelName: l.ClusterID,
	}
}

func (l *Labels) ForIngress(tag string) map[string]string {
	return map[string]string{
		"system":           prefix,
		l.ClusterLabelName: l.ClusterID,
		prefix + "-tag":    tag,
	}
}
