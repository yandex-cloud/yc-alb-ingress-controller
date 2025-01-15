package yc

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/util/sets"
)

type OperationWaiter struct {
	*ycsdk.SDK
}

const (
	ipv4Regex                         = "((25[0-5]|(2[0-4]|1\\d|[1-9]|)\\d)\\.?\\b){4}"
	ipv6Regex                         = `(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`
	completeInternalAddrRegexpPattern = "internal_ipv4_address:\\s*{(address:\"([0-9]{1,3}\\.){3}[0-9]{1,3}\"\\s*subnet_id:\"[a-zA-Z]+\"|subnet_id:\"[a-zA-Z]+\"\\s*address:\"([0-9]{1,3}\\.){3}[0-9]{1,3}\")}"
)

func (sdk OperationWaiter) Result(op *operation.Operation, err error) (proto.Message, error) {
	o, e := sdk.WrapOperation(op, err)
	if e != nil {
		return nil, e
	}
	e = o.Wait(context.Background())
	if e != nil {
		return nil, err
	}
	resp, e := o.Response()
	if e != nil {
		return resp, err
	}
	return resp, nil
}

type UpdatePredicates struct{}

func (*UpdatePredicates) BalancerNeedsUpdate(alb, exp *apploadbalancer.LoadBalancer) bool {
	return locationsNeedUpdate(alb.AllocationPolicy.Locations, exp.AllocationPolicy.Locations) ||
		securityGroupsNeedUpdate(alb.SecurityGroupIds, exp.SecurityGroupIds) ||
		listenersNeedUpdate(alb.Listeners, exp.Listeners) ||
		logOptionsNeedUpdate(alb.LogOptions, exp.LogOptions) ||
		autoscaleNeedUpdate(alb.AutoScalePolicy, exp.AutoScalePolicy)
}

func logOptionsNeedUpdate(act, exp *apploadbalancer.LogOptions) bool {
	return !proto.Equal(act, exp)
}

func autoscaleNeedUpdate(act, exp *apploadbalancer.AutoScalePolicy) bool {
	return !proto.Equal(act, exp)
}

func securityGroupsNeedUpdate(ids1 []string, ids2 []string) bool {
	idSet1 := sets.NewString(ids1...)
	idSet2 := sets.NewString(ids2...)
	return !idSet1.Equal(idSet2)
}

func listenersNeedUpdate(listeners, specs []*apploadbalancer.Listener) bool {
	if len(listeners) != len(specs) {
		return true
	}

	m := make(map[string]int, len(listeners))
	for i, l := range listeners {
		m[l.Name] = i
	}
	for _, s := range specs {
		i, ok := m[s.Name]
		if !ok || listenerNeedsUpdate(listeners[i], s) {
			return true
		}
	}
	return false
}

func serializeExpEp(ep *apploadbalancer.Endpoint) string {
	s := ep.String()

	// only the internal address contains subnet field, so the logic to check if it's set to right value is more complex
	// if endpoint has internal address and ip is set by user, this condition will be true, so we don't need to do anything
	// otherwise we replace static part with the complete internal address regex pattern.
	// In result string we have garbage - id of substring and some other symbols, but we don't really care,
	// because the resulting pattern is still matching target string
	if !regexp.MustCompile(completeInternalAddrRegexpPattern).MatchString(s) {
		s = strings.ReplaceAll(s, "internal_ipv4_address:{subnet", completeInternalAddrRegexpPattern+"|")
	}

	r := strings.NewReplacer("external_ipv4_address:{}", fmt.Sprintf("external_ipv4_address:{address:\"%s\"}", ipv4Regex),
		"external_ipv6_address:{}", fmt.Sprintf("external_ipv6_address:{address:\"%s\"}", ipv6Regex),
	)

	return r.Replace(s)
}

func listenerEndpointsNeedUpdate(act, exp []*apploadbalancer.Endpoint) bool {
	if len(exp) != len(act) {
		return true
	}

	var actExpBuilder strings.Builder
	for _, ep := range act {
		actExpBuilder.WriteString(ep.String())
	}
	actExpString := actExpBuilder.String()

	for _, ep := range exp {
		pattern := serializeExpEp(ep)

		found, _ := regexp.MatchString(pattern, actExpString)
		if !found {
			fmt.Printf("dbg: Listener needs update: act %v, exp %v; pattern: %s, expstring: %s \n", act, exp, pattern, actExpString)

			return true
		}
	}

	return false
}

func listenerNeedsUpdate(listener, spec *apploadbalancer.Listener) bool {
	if listenerEndpointsNeedUpdate(listener.Endpoints, spec.Endpoints) {
		return true
	}

	switch l1 := listener.Listener.(type) {
	case *apploadbalancer.Listener_Http:
		l2, ok := spec.Listener.(*apploadbalancer.Listener_Http)
		return ok && !proto.Equal(l1.Http, l2.Http)
	// TODO: TLS listeners comparison is probably incorrect. implement proper TLS listener update confirmation
	case *apploadbalancer.Listener_Tls:
		l2, ok := spec.Listener.(*apploadbalancer.Listener_Tls)
		return ok && !proto.Equal(l1.Tls, l2.Tls)
	default:
		return false
	}
}

func locationsNeedUpdate(locations, locations2 []*apploadbalancer.Location) bool {
	// for simplicity assume that balancers are valid and therefore slices have no repetitions
	if len(locations) != len(locations2) {
		return true
	}
	type location struct {
		zoneID         string
		subnetID       string
		disableTraffic bool
	}
	m := make(map[location]struct{}, len(locations))
	for _, l := range locations {
		m[location{zoneID: l.ZoneId, subnetID: l.SubnetId, disableTraffic: l.DisableTraffic}] = struct{}{}
	}
	for _, l := range locations2 {
		if _, ok := m[location{zoneID: l.ZoneId, subnetID: l.SubnetId, disableTraffic: l.DisableTraffic}]; !ok {
			return true
		}
	}
	return false
}

func (*UpdatePredicates) RouterNeedsUpdate(r1, r2 *apploadbalancer.HttpRouter) bool {
	if len(r1.VirtualHosts) != len(r2.VirtualHosts) {
		return true
	}
	m := make(map[string]int, len(r1.VirtualHosts))
	for i, vh := range r1.VirtualHosts {
		m[vh.Name] = i
	}
	for _, vh := range r2.VirtualHosts {
		// TODO: refine the comparison
		if vhIndex, ok := m[vh.Name]; !ok || !proto.Equal(r1.VirtualHosts[vhIndex], vh) {
			return true
		}
	}
	return false
}

func (*UpdatePredicates) BackendGroupNeedsUpdate(g1, g2 *apploadbalancer.BackendGroup) bool {
	b1, b2 := g1.GetBackend(), g2.GetBackend()
	if b1 == nil || b2 == nil {
		return (b1 == nil) != (b2 == nil)
	}

	switch t1 := b1.(type) {
	case *apploadbalancer.BackendGroup_Http:
		t2, ok := b2.(*apploadbalancer.BackendGroup_Http)
		return !ok || ok && !proto.Equal(t1.Http, t2.Http)
	case *apploadbalancer.BackendGroup_Grpc:
		t2, ok := b2.(*apploadbalancer.BackendGroup_Grpc)
		return !ok || ok && !proto.Equal(t1.Grpc, t2.Grpc)
	}
	return false
}
