package builders

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/vpc/v1"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/algo"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/k8s"
)

//go:generate mockgen -destination=./mocks/mocks.go -package=mocks . SubnetRepository

const sep = ","

type Resolvers struct {
	repo SubnetRepository
}

func NewResolvers(repo SubnetRepository) *Resolvers {
	return &Resolvers{repo: repo}
}

func (r *Resolvers) Addresses(p AddressParams) *AddressesResolver {
	return &AddressesResolver{defaultSubnetID: p.DefaultSubnetID}
}

func (r *Resolvers) Location() *LocationsResolver {
	return &LocationsResolver{
		repo:      r.repo,
		subnetIDs: make(map[string]struct{}),
		zoneIDs:   make(map[string]string),
	}
}

func (r *Resolvers) SecurityGroups() *SecurityGroupIDsResolver {
	return &SecurityGroupIDsResolver{ids: make(map[string]int)}
}

func (r *Resolvers) AutoScalePolicy() *AutoScalePolicyResolver {
	return &AutoScalePolicyResolver{}
}

func (r *Resolvers) RouteOpts() RouteOptsResolver {
	return RouteOptsResolver{}
}

func (r *Resolvers) VirtualHostOpts() VirtualHostOptsResolver {
	return VirtualHostOptsResolver{}
}

func (r *Resolvers) BackendOpts() BackendOptsResolver {
	return BackendOptsResolver{}
}

type BackendOptsResolver struct{}

type SessionAffinityOpts struct {
	cookie     *apploadbalancer.CookieSessionAffinity
	header     *apploadbalancer.HeaderSessionAffinity
	connection *apploadbalancer.ConnectionSessionAffinity
}

func parseHeaderSessionAffinity(affinity string) (*apploadbalancer.HeaderSessionAffinity, error) {
	m, err := k8s.ParseConfigsFromAnnotationValue(affinity)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header session affinity: %w", err)
	}

	headername, ok := m["name"]
	if !ok {
		return nil, fmt.Errorf("name shoud be specified in header session affinity")
	}

	return &apploadbalancer.HeaderSessionAffinity{
		HeaderName: headername,
	}, nil
}

func parseCookieSessionAffinity(affinity string) (*apploadbalancer.CookieSessionAffinity, error) {
	m, err := k8s.ParseConfigsFromAnnotationValue(affinity)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cookie session affinity: %w", err)
	}

	cookiename, ok := m["name"]
	if !ok {
		return nil, fmt.Errorf("name shoud be specified in cookie session affinity")
	}

	var ttl *durationpb.Duration
	if ttlString, ok := m["ttl"]; ok {
		ttlTime, err := time.ParseDuration(ttlString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ttl: %w", err)
		}

		ttl = durationpb.New(ttlTime)
	}

	return &apploadbalancer.CookieSessionAffinity{
		Name: cookiename,
		Ttl:  ttl,
	}, nil
}

func parseConnectionSessionAffinity(affinity string) (*apploadbalancer.ConnectionSessionAffinity, error) {
	m, err := k8s.ParseConfigsFromAnnotationValue(affinity)
	if err != nil {
		return nil, err
	}

	sourceIP, ok := m["source-ip"]
	if !ok {
		return nil, fmt.Errorf("name shoud be specified in cookie session affinity")
	}

	if sourceIP != "true" && sourceIP != "false" {
		return nil, fmt.Errorf("session-affinity-connection-source-ip must be true or false, found %s", sourceIP)
	}

	return &apploadbalancer.ConnectionSessionAffinity{
		SourceIp: sourceIP == "true",
	}, nil
}

func parseHealthChecks(healthChecks string) ([]*apploadbalancer.HealthCheck, error) {
	m, err := k8s.ParseConfigsFromAnnotationValue(healthChecks)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	healthCheck := healthCheckTemplate()

	port, err := strconv.ParseInt(m["port"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("port should be specified in health-checks")
	}
	healthCheck.HealthcheckPort = port

	if httpPath, ok := m["http-path"]; ok {
		healthCheck.Healthcheck = &apploadbalancer.HealthCheck_Http{
			Http: &apploadbalancer.HealthCheck_HttpHealthCheck{
				Path: httpPath,
			},
		}
	}

	if grpcServiceName, ok := m["grpc-service-name"]; ok {
		healthCheck.Healthcheck = &apploadbalancer.HealthCheck_Grpc{
			Grpc: &apploadbalancer.HealthCheck_GrpcHealthCheck{
				ServiceName: grpcServiceName,
			},
		}
	}

	if timeout, ok := m["timeout"]; ok {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("timeout must be time value, found: %s", timeout)
		}
		healthCheck.Timeout = convertDuration(&metav1.Duration{Duration: duration})
	}

	if interval, ok := m["interval"]; ok {
		duration, err := time.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("interval must be time value, found: %s", interval)
		}
		healthCheck.Interval = convertDuration(&metav1.Duration{Duration: duration})
	}

	if healthyThreshold, ok := m["healthy-threshold"]; ok {
		healthCheck.HealthyThreshold, err = strconv.ParseInt(healthyThreshold, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("healthy-threshold must be number value, found: %s", healthyThreshold)
		}
	}

	if unhealthyThreshold, ok := m["unhealthy-threshold"]; ok {
		healthCheck.UnhealthyThreshold, err = strconv.ParseInt(unhealthyThreshold, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unhealthy-threshold must be number value, found: %s", unhealthyThreshold)
		}
	}

	return []*apploadbalancer.HealthCheck{healthCheck}, nil
}

func (r *BackendOptsResolver) Resolve(
	protocol, balancingMode, balancingPanicThreshold, balancingLocalityAwareRouting,
	transportSecurity, affinityHeader, affinityCookie, affinityConnection, healthChecks string,
) (BackendResolveOpts, error) {
	ret := BackendResolveOpts{}

	ret.LoadBalancingConfig.Mode = balancingMode

	var err error
	if balancingPanicThreshold != "" {
		ret.LoadBalancingConfig.PanicThreshold, err = strconv.ParseInt(balancingPanicThreshold, 10, 64)
		if err != nil {
			return BackendResolveOpts{}, fmt.Errorf("balancing-panic-threshold must be number value, found: %s", balancingPanicThreshold)
		}
	}

	if balancingLocalityAwareRouting != "" {
		ret.LoadBalancingConfig.LocalityAwareRouting, err = strconv.ParseInt(balancingLocalityAwareRouting, 10, 64)
		if err != nil {
			return BackendResolveOpts{}, fmt.Errorf("balancing-locality-aware-routing must be number value, found: %s", balancingLocalityAwareRouting)
		}
	}

	if transportSecurity == "tls" {
		ret.Secure = true
	}

	switch protocol {
	case "http", "":
		ret.BackendType = HTTP
	case "http2":
		ret.BackendType = HTTP2
	case "grpc":
		ret.BackendType = GRPC
	}

	if algo.Count([]string{affinityConnection, affinityCookie, affinityHeader},
		func(s string) bool {
			return s != ""
		}) > 1 {
		return BackendResolveOpts{}, fmt.Errorf("no more than one session affinity type must be specified")
	}

	if affinityCookie != "" {
		ret.affinityOpts.cookie, err = parseCookieSessionAffinity(affinityCookie)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	}

	if affinityHeader != "" {
		ret.affinityOpts.header, err = parseHeaderSessionAffinity(affinityHeader)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	}

	if affinityConnection != "" {
		ret.affinityOpts.connection, err = parseConnectionSessionAffinity(affinityConnection)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	}

	if healthChecks != "" {
		ret.healthChecks, err = parseHealthChecks(healthChecks)
		if err != nil {
			return BackendResolveOpts{}, err
		}
	} else {
		ret.healthChecks = defaultHealthChecks
	}

	return ret, nil
}

const (
	autoAddress = "auto"
	autoIP      = ""
)

type AddressParams struct {
	DefaultSubnetID string
}

type AddressData struct {
	ExternalIPv4, ExternalIPv6, InternalIPv4, SubnetID string
}

type AddressesResolver struct {
	defaultSubnetID string
	err             error
	data            AddressData
}

func (r *AddressesResolver) Resolve(data AddressData) {
	if data.InternalIPv4 != "" {
		if data.SubnetID == "" {
			data.SubnetID = r.defaultSubnetID
		}
	} else if data.SubnetID != "" {
		r.err = fmt.Errorf("subnet provided without internal address")
		return
	}
	r.resolveInto(data.ExternalIPv4, &r.data.ExternalIPv4, "external IPv4")
	r.resolveInto(data.ExternalIPv6, &r.data.ExternalIPv6, "external IPv6")
	r.resolveInto(data.InternalIPv4, &r.data.InternalIPv4, "internal IPv4")
	r.resolveInto(data.SubnetID, &r.data.SubnetID, "subnet for internal IPv4")
}

func (r *AddressesResolver) resolveInto(src string, dst *string, fieldName string) {
	if r.err != nil || src == "" || src == *dst {
		return
	}
	if *dst == "" {
		*dst = src
	} else {
		r.err = fmt.Errorf("different values provided for %s: %s, %s", fieldName, *dst, src)
	}
}

func (r *AddressesResolver) Result() ([]*apploadbalancer.Address, error) {
	addrs := make([]*apploadbalancer.Address, 0)
	var (
		externalIPv4 = func(s string) {
			addrs = append(addrs, &apploadbalancer.Address{Address: &apploadbalancer.Address_ExternalIpv4Address{
				ExternalIpv4Address: &apploadbalancer.ExternalIpv4Address{Address: s},
			}})
		}
		externalIPv6 = func(s string) {
			addrs = append(addrs, &apploadbalancer.Address{Address: &apploadbalancer.Address_ExternalIpv6Address{
				ExternalIpv6Address: &apploadbalancer.ExternalIpv6Address{Address: s},
			}})
		}
		internalIPv4 = func(s string) {
			addrs = append(addrs, &apploadbalancer.Address{
				Address: &apploadbalancer.Address_InternalIpv4Address{
					InternalIpv4Address: &apploadbalancer.InternalIpv4Address{
						Address:  s,
						SubnetId: r.data.SubnetID,
					},
				},
			})
		}
		addressFn = func(s string, fn func(string)) {
			if !(s == "" || r.err != nil) {
				if s == autoAddress {
					s = autoIP
				}
				fn(s)
			}
		}
	)
	addressFn(r.data.ExternalIPv4, externalIPv4)
	addressFn(r.data.ExternalIPv6, externalIPv6)
	addressFn(r.data.InternalIPv4, internalIPv4)
	if r.err != nil {
		return nil, r.err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no balancer address provided")
	}
	return addrs, nil
}

type SubnetRepository interface {
	FindSubnetByID(context.Context, string) (*vpc.Subnet, error)
}

type LocationsResolver struct {
	repo      SubnetRepository
	subnetIDs map[string]struct{}
	zoneIDs   map[string]string
	networkID string
}

func (r *LocationsResolver) Resolve(subnetStr string) error {
	// TODO: may be no need to fail on resolve, returning error from Result() is enough
	subnetIDs := strings.Split(subnetStr, sep)

	for i, subnetID := range subnetIDs {
		if _, ok := r.subnetIDs[subnetID]; ok || subnetID == "" {
			continue
		}

		subnet, err := r.repo.FindSubnetByID(context.Background(), subnetID)
		if err != nil {
			return fmt.Errorf("error retrieving subnet %s: %w ", subnetID, err)
		}
		r.subnetIDs[subnetID] = struct{}{}
		if _, ok := r.zoneIDs[subnet.ZoneId]; ok {
			return fmt.Errorf("at least two subnets belong to the same zone %s", r.zoneIDs[subnet.ZoneId])
		}
		r.zoneIDs[subnet.ZoneId] = subnetID
		if i == 0 {
			r.networkID = subnet.NetworkId
		} else if r.networkID != subnet.NetworkId {
			return fmt.Errorf("subnets are from at least two different networks: %s, %s", r.networkID, subnet.NetworkId)
		}
	}
	return nil
}

func (r *LocationsResolver) Result() (string, []*apploadbalancer.Location, error) {
	if len(r.zoneIDs) == 0 {
		return "", nil, fmt.Errorf("no subnets provided")
	}
	var locations []*apploadbalancer.Location
	for zoneID, subnetID := range r.zoneIDs {
		locations = append(locations, &apploadbalancer.Location{
			ZoneId:   zoneID,
			SubnetId: subnetID,
		})
	}
	sort.Slice(locations, func(i, j int) bool { return locations[i].ZoneId < locations[j].ZoneId })
	return r.networkID, locations, nil
}

type SecurityGroupIDsResolver struct {
	ids map[string]int
}

func (r *SecurityGroupIDsResolver) Resolve(securityGroupStr string) {
	if len(securityGroupStr) == 0 {
		return
	}
	groupIDs := strings.Split(securityGroupStr, sep)
	for _, groupID := range groupIDs {
		if _, ok := r.ids[groupID]; !ok && len(groupID) > 0 {
			r.ids[groupID] = len(r.ids)
		}
	}
}

func (r *SecurityGroupIDsResolver) Result() (ids []string) {
	if len(r.ids) > 0 {
		ids = make([]string, len(r.ids))
		for id, i := range r.ids {
			ids[i] = id
		}
	}
	return
}

type AutoScalePolicyResolver struct {
	MinZoneSize *int64
	MaxSize     *int64
}

func (r *AutoScalePolicyResolver) Resolve(minZoneSize, maxSize string) error {
	if minZoneSize != "" {
		m, err := strconv.ParseInt(minZoneSize, 10, 64)
		if err != nil {
			return err
		}
		if r.MinZoneSize != nil && *r.MinZoneSize != m {
			return fmt.Errorf("different values provided for min size: %d, %d", *r.MinZoneSize, m)
		}
		r.MinZoneSize = &m
	}
	if maxSize != "" {
		m, err := strconv.ParseInt(maxSize, 10, 64)
		if err != nil {
			return err
		}
		if r.MaxSize != nil && *r.MaxSize != m {
			return fmt.Errorf("different values provided for max size: %d, %d", *r.MaxSize, m)
		}
		r.MaxSize = &m
	}
	return nil
}

func (r *AutoScalePolicyResolver) Result() *apploadbalancer.AutoScalePolicy {
	if r.MinZoneSize == nil && r.MaxSize == nil {
		return nil
	}
	p := &apploadbalancer.AutoScalePolicy{}
	if r.MinZoneSize != nil {
		p.MinZoneSize = *r.MinZoneSize
	}
	if r.MaxSize != nil {
		p.MaxSize = *r.MaxSize
	}
	return p
}

type RouteOptsResolver struct{}

func (r RouteOptsResolver) Resolve(
	timeout, idleTimeout, prefixRewrite, upgradeTypes,
	proto, useRegex, allowedMethods string,
) (RouteResolveOpts, error) {
	var ret RouteResolveOpts
	if len(timeout) > 0 {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return RouteResolveOpts{}, err
		}
		ret.Timeout = durationpb.New(d)
	}

	if len(idleTimeout) > 0 {
		d, err := time.ParseDuration(idleTimeout)
		if err != nil {
			return RouteResolveOpts{}, err
		}
		ret.IdleTimeout = durationpb.New(d)
	}

	ret.PrefixRewrite = prefixRewrite

	if len(upgradeTypes) > 0 {
		ret.UpgradeTypes = strings.Split(upgradeTypes, ",")
	}

	switch proto {
	case "grpc":
		ret.BackendType = GRPC
	case "http2":
		ret.BackendType = HTTP2
	case "http", "":
		ret.BackendType = HTTP
	default:
		return RouteResolveOpts{}, fmt.Errorf("unsupported backend protocol %s", proto)
	}

	switch useRegex {
	case "true":
		ret.UseRegex = true
	case "false", "":
		ret.UseRegex = false
	default:
		return RouteResolveOpts{}, fmt.Errorf("unsupported useRegex flag format %s", useRegex)
	}

	if len(allowedMethods) > 0 {
		ret.AllowedMethods = strings.Split(allowedMethods, ",")
	}

	return ret, nil
}

type VirtualHostOptsResolver struct{}

func (r *VirtualHostOptsResolver) Resolve(
	removeResponseHeader, renameResponseHeader, appendResponseHeader, replaceResponseHeader,
	removeRequestHeader, renameRequestHeader, appendRequestHeader, replaceRequestHeader,
	securityProfileID string,
) (VirtualHostResolveOpts, error) {
	var ret VirtualHostResolveOpts

	resolveModifyHeader := func(removeHeader, renameHeader, appendHeader, replaceHeader string) (ModifyHeaderOpts, error) {
		var err error
		ret := ModifyHeaderOpts{}

		ret.Append, err = k8s.ParseModifyHeadersFromAnnotationValue(appendHeader)
		if err != nil {
			return ModifyHeaderOpts{}, err
		}

		ret.Rename, err = k8s.ParseModifyHeadersFromAnnotationValue(renameHeader)
		if err != nil {
			return ModifyHeaderOpts{}, err
		}

		ret.Replace, err = k8s.ParseModifyHeadersFromAnnotationValue(replaceHeader)
		if err != nil {
			return ModifyHeaderOpts{}, err
		}

		removeOpts, err := k8s.ParseModifyHeadersFromAnnotationValue(removeHeader)
		if err != nil {
			return ModifyHeaderOpts{}, err
		}

		if len(removeOpts) > 0 {
			ret.Remove = make(map[string]bool, len(removeOpts))
			for name, value := range removeOpts {
				if value != "true" && value != "false" {
					return ModifyHeaderOpts{}, fmt.Errorf("wrong modify-response-rewrite format, should be \"true\" or \"false\", got %s", value)
				}

				ret.Remove[name] = value == "true"
			}
		}

		return ret, nil
	}

	var err error
	ret.ModifyResponse, err = resolveModifyHeader(removeResponseHeader, renameResponseHeader, appendResponseHeader, replaceResponseHeader)
	if err != nil {
		return VirtualHostResolveOpts{}, err
	}

	ret.ModifyRequest, err = resolveModifyHeader(removeRequestHeader, renameRequestHeader, appendRequestHeader, replaceRequestHeader)
	if err != nil {
		return VirtualHostResolveOpts{}, err
	}
	ret.SecurityProfileID = securityProfileID
	return ret, nil
}
