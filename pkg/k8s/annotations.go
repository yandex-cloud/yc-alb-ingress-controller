package k8s

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	prefix            = "ingress.alb.yc.io"
	AlbTag            = prefix + "/group-name"
	SecurityGroups    = prefix + "/security-groups"
	SecurityProfileID = prefix + "/security-profile-id"
	Subnets           = prefix + "/subnets"

	AutoscalingPrefix      = prefix + "/autoscale-"
	AutoscalingMinZoneSize = AutoscalingPrefix + "min-zone-size"
	AutoscalingMaxSize     = AutoscalingPrefix + "max-size"

	ExternalIPv4Address = prefix + "/external-ipv4-address"
	ExternalIPv6Address = prefix + "/external-ipv6-address"
	InternalIPv4Address = prefix + "/internal-ipv4-address"
	InternalALBSubnet   = prefix + "/internal-alb-subnet"

	RequestTimeout = prefix + "/request-timeout"
	IdleTimeout    = prefix + "/idle-timeout"
	PrefixRewrite  = prefix + "/prefix-rewrite"
	UpgradeTypes   = prefix + "/upgrade-types"
	AllowedMethods = prefix + "/allowed-methods"

	Protocol          = prefix + "/protocol"
	TransportSecurity = prefix + "/transport-security"
	HealthChecks      = prefix + "/health-checks"

	UseRegex     = prefix + "/use-regex"
	OrderInGroup = prefix + "/group-order"

	BalancingPrefix               = prefix + "/balancing-"
	BalancingMode                 = BalancingPrefix + "mode"
	BalancingPanicThreshold       = BalancingPrefix + "panic-threshold"
	BalancingLocalityAwareRouting = BalancingPrefix + "locality-aware-routing"

	SessionAffinityPrefix     = prefix + "/session-affinity-"
	SessionAffinityHeader     = SessionAffinityPrefix + "header"
	SessionAffinityCookie     = SessionAffinityPrefix + "cookie"
	SessionAffinityConnection = SessionAffinityPrefix + "connection"

	GroupSettings = prefix + "/group-settings-name"

	defaultTag = "default"

	ModifyResponseHeaderPrefix  = prefix + "/modify-header-response-"
	ModifyResponseHeaderReplace = ModifyResponseHeaderPrefix + "replace"
	ModifyResponseHeaderAppend  = ModifyResponseHeaderPrefix + "append"
	ModifyResponseHeaderRename  = ModifyResponseHeaderPrefix + "rename"
	ModifyResponseHeaderRemove  = ModifyResponseHeaderPrefix + "remove"

	ModifyRequestHeaderPrefix  = prefix + "/modify-header-request-"
	ModifyRequestHeaderReplace = ModifyRequestHeaderPrefix + "replace"
	ModifyRequestHeaderAppend  = ModifyRequestHeaderPrefix + "append"
	ModifyRequestHeaderRename  = ModifyRequestHeaderPrefix + "rename"
	ModifyRequestHeaderRemove  = ModifyRequestHeaderPrefix + "remove"

	DirectResponsePrefix = prefix + "/direct-response."
	RedirectPrefix       = prefix + "/redirect."

	DefaultIngressClass = "ingressclass.kubernetes.io/is-default-class"

	PreferIPv6Targets = prefix + "/prefer-ipv6-targets"
)

func GetBalancerTag(o metav1.Object) string {
	if tag, ok := o.GetAnnotations()[AlbTag]; ok {
		return tag
	}
	return defaultTag
}

func HasBalancerTag(o metav1.Object) bool {
	_, ok := o.GetAnnotations()[AlbTag]
	return ok
}

func ParseConfigsFromAnnotationValue(s string) (map[string]string, error) {
	if len(s) == 0 {
		return nil, nil
	}

	result := make(map[string]string)

	elements := strings.Split(s, ",")
	for _, element := range elements {
		words := strings.Split(element, "=")
		if len(words) != 2 {
			return nil, fmt.Errorf("wrong config format in annotation: %s", s)
		}

		if len(words[0]) == 0 {
			return nil, fmt.Errorf("empty key in annotation: %s", s)
		}

		result[words[0]] = words[1]
	}

	return result, nil
}

func ParseModifyHeadersFromAnnotationValue(s string) (map[string]string, error) {
	if len(s) == 0 {
		return nil, nil
	}

	result := make(map[string]string)

	elements := strings.Split(s, ",")
	for _, element := range elements {
		words := strings.Split(element, "=")
		if len(words) != 2 {
			return nil, fmt.Errorf("wrong config format in annotation: %s", s)
		}

		if len(words[0]) == 0 {
			return nil, fmt.Errorf("empty key in annotation: %s", s)
		}

		if _, has := result[words[0]]; has {
			result[words[0]] += "," + words[1]
		} else {
			result[words[0]] = words[1]
		}
	}

	return result, nil
}

func GetIngressGroupAnnotation(g *IngressGroup, annotation string) (string, error) {
	result := ""

	for _, item := range g.Items {
		curr := item.Annotations[annotation]
		if result == "" {
			result = curr
		}

		if curr != "" && curr != result {
			return "", fmt.Errorf("%s annotation has different values in single ingressgroup %s, values: %s, %s", annotation, g.Tag, curr, result)
		}
	}

	return result, nil
}
