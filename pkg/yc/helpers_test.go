package yc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestLocationsNeedUpdate(t *testing.T) {
	var (
		locationA = &apploadbalancer.Location{
			ZoneId:         "ru-central1-a",
			SubnetId:       "idXXXX01",
			DisableTraffic: false,
		}
		locationB = &apploadbalancer.Location{
			ZoneId:         "ru-central1-b",
			SubnetId:       "idXXXX02",
			DisableTraffic: false,
		}
		locationC = &apploadbalancer.Location{
			ZoneId:         "ru-central1-c",
			SubnetId:       "idXXXX03",
			DisableTraffic: false,
		}
		locationBDisabled = &apploadbalancer.Location{
			ZoneId:         "ru-central1-b",
			SubnetId:       "idXXXX02",
			DisableTraffic: true,
		}
	)
	testData := []struct {
		desc   string
		l1, l2 []*apploadbalancer.Location
		exp    bool
	}{
		{
			desc: "equal, different order",
			l1:   []*apploadbalancer.Location{locationA, locationB},
			l2:   []*apploadbalancer.Location{locationB, locationA},
			exp:  false,
		},
		{
			desc: "different subnets",
			l1:   []*apploadbalancer.Location{locationA, locationB},
			l2:   []*apploadbalancer.Location{locationA, locationC},
			exp:  true,
		},
		{
			desc: "different DisableTraffic value",
			l1:   []*apploadbalancer.Location{locationA, locationB},
			l2:   []*apploadbalancer.Location{locationA, locationBDisabled},
			exp:  true,
		},
		{
			desc: "expected less locations than the current state",
			l1:   []*apploadbalancer.Location{locationA, locationB},
			l2:   []*apploadbalancer.Location{locationA},
			exp:  true,
		},
		{
			desc: "expected more locations than the current state",
			l1:   []*apploadbalancer.Location{locationA},
			l2:   []*apploadbalancer.Location{locationA, locationB},
			exp:  true,
		},
		{
			desc: "expected locations empty", // cannot really happen
			l1:   []*apploadbalancer.Location{locationA, locationBDisabled},
			l2:   []*apploadbalancer.Location{},
			exp:  true,
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			res := locationsNeedUpdate(tc.l1, tc.l2)
			assert.Equal(t, tc.exp, res)
		})
	}
}

func TestBackendGroupNeedsUpdate(t *testing.T) {
	var (
		bg1 = &apploadbalancer.BackendGroup{
			Id:          "ID_1",
			Name:        "BackendGroup_1",
			Description: "any",
			FolderId:    "",
			Labels:      nil,
			Backend: &apploadbalancer.BackendGroup_Http{
				Http: &apploadbalancer.HttpBackendGroup{
					Backends: []*apploadbalancer.HttpBackend{},
				},
			},
			CreatedAt: &timestamppb.Timestamp{
				Seconds: 1111,
				Nanos:   0,
			},
		}
		bg2 = &apploadbalancer.BackendGroup{
			Id:          "ID_2",
			Name:        "BackendGroup_2",
			Description: "any other",
			FolderId:    "",
			Labels:      nil,
			Backend: &apploadbalancer.BackendGroup_Http{
				Http: &apploadbalancer.HttpBackendGroup{
					Backends: []*apploadbalancer.HttpBackend{},
				},
			},
			CreatedAt: &timestamppb.Timestamp{
				Seconds: 2222,
				Nanos:   0,
			},
		}
		bg2_2 = &apploadbalancer.BackendGroup{
			Id:          "ID_2",
			Name:        "BackendGroup_2",
			Description: "any other",
			FolderId:    "",
			Labels:      nil,
			Backend: &apploadbalancer.BackendGroup_Http{
				Http: &apploadbalancer.HttpBackendGroup{
					Backends: []*apploadbalancer.HttpBackend{{
						BackendWeight: &wrapperspb.Int64Value{Value: 30},
					}},
				},
			},
			CreatedAt: &timestamppb.Timestamp{
				Seconds: 2222,
				Nanos:   0,
			},
		}
	)
	testData := []struct {
		desc     string
		bg1, bg2 *apploadbalancer.BackendGroup
		exp      bool
	}{
		{
			desc: "equal, different irrelevant fields time",
			bg1:  bg1,
			bg2:  bg2,
			exp:  false,
		},
		{
			desc: "different",
			bg1:  bg1,
			bg2:  bg2_2,
			exp:  true,
		},
		{
			desc: "non-http", // other types not supported yet
			bg1:  nil,
			bg2:  nil,
			exp:  false,
		},
	}
	p := &UpdatePredicates{}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			res := p.BackendGroupNeedsUpdate(tc.bg1, tc.bg2)
			assert.Equal(t, tc.exp, res)
		})
	}
}

func TestBalancerNeedsUpdate(t *testing.T) {
	var (
		listener = &apploadbalancer.Listener{
			Name: "listener-1",
			Endpoints: []*apploadbalancer.Endpoint{
				{
					Ports: []int64{
						80,
					},
					Addresses: []*apploadbalancer.Address{
						{
							Address: &apploadbalancer.Address_ExternalIpv4Address{
								ExternalIpv4Address: &apploadbalancer.ExternalIpv4Address{
									Address: "8.8.8.8",
								},
							},
						},
					},
				},
			},
			Listener: &apploadbalancer.Listener_Http{
				Http: &apploadbalancer.HttpListener{
					Handler: &apploadbalancer.HttpHandler{
						HttpRouterId: "router-1",
					},
				},
			},
		}
		listenerInternal = &apploadbalancer.Listener{
			Name: "listener-1",
			Endpoints: []*apploadbalancer.Endpoint{
				{
					Ports: []int64{
						80,
					},
					Addresses: []*apploadbalancer.Address{
						{
							Address: &apploadbalancer.Address_InternalIpv4Address{
								InternalIpv4Address: &apploadbalancer.InternalIpv4Address{
									Address:  "8.8.8.8",
									SubnetId: "subnet-id",
								},
							},
						},
					},
				},
			},
			Listener: &apploadbalancer.Listener_Http{
				Http: &apploadbalancer.HttpListener{
					Handler: &apploadbalancer.HttpHandler{
						HttpRouterId: "router-1",
					},
				},
			},
		}
		listenerInternalAuto = &apploadbalancer.Listener{
			Name: "listener-1",
			Endpoints: []*apploadbalancer.Endpoint{
				{
					Ports: []int64{
						80,
					},
					Addresses: []*apploadbalancer.Address{
						{
							Address: &apploadbalancer.Address_InternalIpv4Address{
								InternalIpv4Address: &apploadbalancer.InternalIpv4Address{
									SubnetId: "subnet-id",
								},
							},
						},
					},
				},
			},
			Listener: &apploadbalancer.Listener_Http{
				Http: &apploadbalancer.HttpListener{
					Handler: &apploadbalancer.HttpHandler{
						HttpRouterId: "router-1",
					},
				},
			},
		}
		listenerIPv6 = &apploadbalancer.Listener{
			Name: "listener-1",
			Endpoints: []*apploadbalancer.Endpoint{
				{
					Ports: []int64{
						80,
					},
					Addresses: []*apploadbalancer.Address{
						{
							Address: &apploadbalancer.Address_ExternalIpv6Address{
								ExternalIpv6Address: &apploadbalancer.ExternalIpv6Address{
									Address: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
								},
							},
						},
					},
				},
			},
			Listener: &apploadbalancer.Listener_Http{
				Http: &apploadbalancer.HttpListener{
					Handler: &apploadbalancer.HttpHandler{
						HttpRouterId: "router-1",
					},
				},
			},
		}
		listenerIPv6Auto = &apploadbalancer.Listener{
			Name: "listener-1",
			Endpoints: []*apploadbalancer.Endpoint{
				{
					Ports: []int64{
						80,
					},
					Addresses: []*apploadbalancer.Address{
						{
							Address: &apploadbalancer.Address_ExternalIpv6Address{
								ExternalIpv6Address: &apploadbalancer.ExternalIpv6Address{},
							},
						},
					},
				},
			},
			Listener: &apploadbalancer.Listener_Http{
				Http: &apploadbalancer.HttpListener{
					Handler: &apploadbalancer.HttpHandler{
						HttpRouterId: "router-1",
					},
				},
			},
		}
		listenerDiffListener = &apploadbalancer.Listener{
			Name: "listener-1",
			Endpoints: []*apploadbalancer.Endpoint{
				{
					Ports: []int64{
						80,
					},
					Addresses: []*apploadbalancer.Address{
						{
							Address: &apploadbalancer.Address_ExternalIpv4Address{
								ExternalIpv4Address: &apploadbalancer.ExternalIpv4Address{
									Address: "8.8.8.8",
								},
							},
						},
					},
				},
			},
			Listener: &apploadbalancer.Listener_Http{
				Http: &apploadbalancer.HttpListener{
					Handler: &apploadbalancer.HttpHandler{
						HttpRouterId: "router-2",
					},
				},
			},
		}
		listenerDiffEndpoints = &apploadbalancer.Listener{
			Name: "listener-1",
			Endpoints: []*apploadbalancer.Endpoint{
				{
					Ports: []int64{
						80,
					},
					Addresses: []*apploadbalancer.Address{
						{
							Address: &apploadbalancer.Address_ExternalIpv4Address{
								ExternalIpv4Address: &apploadbalancer.ExternalIpv4Address{
									Address: "9.9.9.9",
								},
							},
						},
					},
				},
			},
			Listener: &apploadbalancer.Listener_Http{
				Http: &apploadbalancer.HttpListener{
					Handler: &apploadbalancer.HttpHandler{
						HttpRouterId: "router-1",
					},
				},
			},
		}

		location1 = &apploadbalancer.Location{
			ZoneId:         "zone-1",
			SubnetId:       "subnet-1",
			DisableTraffic: false,
		}

		location2 = &apploadbalancer.Location{
			ZoneId:         "zone-2",
			SubnetId:       "subnet-2",
			DisableTraffic: false,
		}

		balancer = apploadbalancer.LoadBalancer{
			AllocationPolicy: &apploadbalancer.AllocationPolicy{
				Locations: []*apploadbalancer.Location{
					location1,
				},
			},
			SecurityGroupIds: []string{
				"sg1", "sg2",
			},
			Listeners: []*apploadbalancer.Listener{
				listener,
			},
		}

		balancerIPv6 = apploadbalancer.LoadBalancer{
			AllocationPolicy: &apploadbalancer.AllocationPolicy{
				Locations: []*apploadbalancer.Location{
					location1,
				},
			},
			SecurityGroupIds: []string{
				"sg1", "sg2",
			},
			Listeners: []*apploadbalancer.Listener{
				listenerIPv6,
			},
		}

		balancerIPv6Auto = apploadbalancer.LoadBalancer{
			AllocationPolicy: &apploadbalancer.AllocationPolicy{
				Locations: []*apploadbalancer.Location{
					location1,
				},
			},
			SecurityGroupIds: []string{
				"sg1", "sg2",
			},
			Listeners: []*apploadbalancer.Listener{
				listenerIPv6Auto,
			},
		}

		balancerInternal = apploadbalancer.LoadBalancer{
			AllocationPolicy: &apploadbalancer.AllocationPolicy{
				Locations: []*apploadbalancer.Location{
					location1,
				},
			},
			SecurityGroupIds: []string{
				"sg1", "sg2",
			},
			Listeners: []*apploadbalancer.Listener{
				listenerInternal,
			},
		}

		balancerInternalAuto = apploadbalancer.LoadBalancer{
			AllocationPolicy: &apploadbalancer.AllocationPolicy{
				Locations: []*apploadbalancer.Location{
					location1,
				},
			},
			SecurityGroupIds: []string{
				"sg1", "sg2",
			},
			Listeners: []*apploadbalancer.Listener{
				listenerInternalAuto,
			},
		}

		differentLocation = apploadbalancer.LoadBalancer{
			AllocationPolicy: &apploadbalancer.AllocationPolicy{
				Locations: []*apploadbalancer.Location{
					location2,
				},
			},
			SecurityGroupIds: []string{
				"sg1", "sg2",
			},
			Listeners: []*apploadbalancer.Listener{
				listener,
			},
		}

		differentSG = apploadbalancer.LoadBalancer{
			AllocationPolicy: &apploadbalancer.AllocationPolicy{
				Locations: []*apploadbalancer.Location{
					location1,
				},
			},
			SecurityGroupIds: []string{
				"sg1",
			},
			Listeners: []*apploadbalancer.Listener{
				listener,
			},
		}

		differentListener1 = apploadbalancer.LoadBalancer{
			AllocationPolicy: &apploadbalancer.AllocationPolicy{
				Locations: []*apploadbalancer.Location{
					location1,
				},
			},
			SecurityGroupIds: []string{
				"sg1", "sg2",
			},
			Listeners: []*apploadbalancer.Listener{
				listenerDiffEndpoints,
			},
		}

		differentListener2 = apploadbalancer.LoadBalancer{
			AllocationPolicy: &apploadbalancer.AllocationPolicy{
				Locations: []*apploadbalancer.Location{
					location1,
				},
			},
			SecurityGroupIds: []string{
				"sg1", "sg2",
			},
			Listeners: []*apploadbalancer.Listener{
				listenerDiffListener,
			},
		}

		listenerAutoEndpoints = &apploadbalancer.Listener{
			Name: "listener-1",
			Endpoints: []*apploadbalancer.Endpoint{
				{
					Ports: []int64{
						80,
					},
					Addresses: []*apploadbalancer.Address{
						{
							Address: &apploadbalancer.Address_ExternalIpv4Address{
								ExternalIpv4Address: &apploadbalancer.ExternalIpv4Address{},
							},
						},
					},
				},
			},
			Listener: &apploadbalancer.Listener_Http{
				Http: &apploadbalancer.HttpListener{
					Handler: &apploadbalancer.HttpHandler{
						HttpRouterId: "router-1",
					},
				},
			},
		}

		balancerAutoEndpoints = apploadbalancer.LoadBalancer{
			AllocationPolicy: &apploadbalancer.AllocationPolicy{
				Locations: []*apploadbalancer.Location{
					location1,
				},
			},
			SecurityGroupIds: []string{
				"sg1", "sg2",
			},
			Listeners: []*apploadbalancer.Listener{
				listenerAutoEndpoints,
			},
		}
	)

	for _, tc := range []struct {
		desc     string
		lhs, rhs *apploadbalancer.LoadBalancer
		res      bool
	}{
		{
			desc: "same",
			lhs:  &balancer,
			rhs:  &balancer,
			res:  false,
		},
		{
			desc: "different listener",
			lhs:  &balancer,
			rhs:  &differentListener2,
			res:  true,
		},
		{
			desc: "different endpoints",
			lhs:  &balancer,
			rhs:  &differentListener1,
			res:  true,
		},
		{
			desc: "different location",
			lhs:  &balancer,
			rhs:  &differentLocation,
			res:  true,
		},
		{
			desc: "different sg",
			lhs:  &balancer,
			rhs:  &differentSG,
			res:  true,
		},
		{
			desc: "auto endpoints",
			lhs:  &balancer,
			rhs:  &balancerAutoEndpoints,
			res:  false,
		},
		{
			desc: "auto endpoints internal",
			lhs:  &balancerInternal,
			rhs:  &balancerInternalAuto,
			res:  false,
		},
		{
			desc: "auto endpoints ipv6",
			lhs:  &balancerIPv6,
			rhs:  &balancerIPv6Auto,
			res:  false,
		},
	} {
		predicates := &UpdatePredicates{}

		needsUpdate := predicates.BalancerNeedsUpdate(tc.lhs, tc.rhs)
		require.Equal(t, tc.res, needsUpdate)
	}
}
