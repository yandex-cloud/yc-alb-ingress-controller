package builders

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/vpc/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/builders/mocks"
)

func TestAddressResolver(t *testing.T) {
	testData := []struct {
		desc     string
		addrs    []AddressData
		expected []*apploadbalancer.Address
		wantErr  bool
	}{
		{
			desc:     "OK",
			addrs:    []AddressData{{}, {InternalIPv4: "168.0.2.16", SubnetID: "rftghxxxx"}, {}},
			expected: []*apploadbalancer.Address{{Address: &apploadbalancer.Address_InternalIpv4Address{InternalIpv4Address: &apploadbalancer.InternalIpv4Address{Address: "168.0.2.16", SubnetId: "rftghxxxx"}}}},
		},
		{
			// internal address and subnet must be defined in the same ingress
			desc:    "subnet without internal address",
			addrs:   []AddressData{{SubnetID: "rftghxxxx"}, {InternalIPv4: "168.0.2.16"}, {}},
			wantErr: true,
		},
		{
			desc:     "OK with default subnet",
			addrs:    []AddressData{{}, {InternalIPv4: "168.0.2.16"}, {}},
			expected: []*apploadbalancer.Address{{Address: &apploadbalancer.Address_InternalIpv4Address{InternalIpv4Address: &apploadbalancer.InternalIpv4Address{Address: "168.0.2.16", SubnetId: "abcdxxxxdefault"}}}},
		},
		{
			desc:     "same address defined twice",
			addrs:    []AddressData{{ExternalIPv4: "168.0.2.16"}, {ExternalIPv4: "168.0.2.16"}, {}},
			expected: []*apploadbalancer.Address{{Address: &apploadbalancer.Address_ExternalIpv4Address{ExternalIpv4Address: &apploadbalancer.ExternalIpv4Address{Address: "168.0.2.16"}}}},
		},
		{
			desc:     "same internal address defined twice",
			addrs:    []AddressData{{}, {InternalIPv4: "168.0.2.16", SubnetID: "rftghxxxx"}, {InternalIPv4: "168.0.2.16", SubnetID: "rftghxxxx"}, {}},
			expected: []*apploadbalancer.Address{{Address: &apploadbalancer.Address_InternalIpv4Address{InternalIpv4Address: &apploadbalancer.InternalIpv4Address{Address: "168.0.2.16", SubnetId: "rftghxxxx"}}}},
		},
		{
			desc:     "same internal address defined twice with the same explicit and default subnet",
			addrs:    []AddressData{{}, {InternalIPv4: "168.0.2.16"}, {InternalIPv4: "168.0.2.16", SubnetID: "abcdxxxxdefault"}, {}},
			expected: []*apploadbalancer.Address{{Address: &apploadbalancer.Address_InternalIpv4Address{InternalIpv4Address: &apploadbalancer.InternalIpv4Address{Address: "168.0.2.16", SubnetId: "abcdxxxxdefault"}}}},
		},
		{
			desc:    "same type of address defined twice",
			addrs:   []AddressData{{ExternalIPv4: "168.0.2.16"}, {ExternalIPv4: "168.0.2.17"}, {}},
			wantErr: true,
		},
		{
			desc:  "different types of address defined",
			addrs: []AddressData{{ExternalIPv4: "168.0.2.16"}, {ExternalIPv6: "2001:0db8:0001:0000:0000:0ab9:C0A8:0102"}, {}},
			expected: []*apploadbalancer.Address{
				{
					Address: &apploadbalancer.Address_ExternalIpv4Address{ExternalIpv4Address: &apploadbalancer.ExternalIpv4Address{Address: "168.0.2.16"}},
				},
				{
					Address: &apploadbalancer.Address_ExternalIpv6Address{ExternalIpv6Address: &apploadbalancer.ExternalIpv6Address{Address: "2001:0db8:0001:0000:0000:0ab9:C0A8:0102"}},
				},
			},
		},
		{
			// no validation performed on our side at the moment
			desc:     "invalid address",
			addrs:    []AddressData{{ExternalIPv4: "168.0.2.a6"}, {}},
			expected: []*apploadbalancer.Address{{Address: &apploadbalancer.Address_ExternalIpv4Address{ExternalIpv4Address: &apploadbalancer.ExternalIpv4Address{Address: "168.0.2.a6"}}}},
			wantErr:  false,
		},
		{
			// should fail, we enforce "auto" instead of empty strings
			desc:    "empty address strings",
			addrs:   []AddressData{{}, {}},
			wantErr: true,
		},
		{
			desc:     "auto IPv4 address string",
			addrs:    []AddressData{{}, {ExternalIPv4: "auto"}},
			expected: []*apploadbalancer.Address{{Address: &apploadbalancer.Address_ExternalIpv4Address{}}},
		},
		{
			desc:     "auto IPv6 address string",
			addrs:    []AddressData{{}, {ExternalIPv6: "auto"}},
			expected: []*apploadbalancer.Address{{Address: &apploadbalancer.Address_ExternalIpv6Address{}}},
		},
	}
	resolvers := &Resolvers{}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			r := resolvers.Addresses(AddressParams{"abcdxxxxdefault"})
			var err error
			for i := 0; i < len(tc.addrs); i++ {
				r.Resolve(tc.addrs[i])
			}

			res, err := r.Result()
			require.True(t, (err != nil) == tc.wantErr, "Result() error = %v)", err)
			if tc.wantErr {
				return
			}

			for i := range res {
				assert.Condition(t, func() bool { return proto.Equal(tc.expected[i], res[i]) }, "exp %v\ngot %v", tc.expected, res)
			}
		})
	}
}

func TestLocations(t *testing.T) {
	subnets := map[string]*vpc.Subnet{
		"idXXX1": {
			Id:        "idXXX1",
			Name:      "subnet1",
			NetworkId: "idXXXXDefault",
			ZoneId:    "zone-A",
		},
		"idXXX2": {
			Id:        "idXXX2",
			Name:      "subnet2",
			NetworkId: "idXXXXDefault",
			ZoneId:    "zone-B",
		},
		"idXXX3": {
			Id:        "idXXX3",
			Name:      "subnet3",
			NetworkId: "idXXXXDefault",
			ZoneId:    "zone-C",
		},
		"idXXX4": {
			Id:        "idXXX4",
			Name:      "subnet4",
			NetworkId: "idXXXXDefault",
			ZoneId:    "zone-A",
		},
		"idXXX5": {
			Id:        "idXXX5",
			Name:      "subnet5",
			NetworkId: "idXXXXOther",
			ZoneId:    "zone-A",
		},
	}

	testData := []struct {
		desc              string
		subnetIDStrs      []string
		expectedLocations []*apploadbalancer.Location
		expectedNetwork   string
		wantResolveErr    bool
		wantResultErr     bool
		mockSubnetCalls   []string
	}{
		{
			desc:         "OK",
			subnetIDStrs: []string{"idXXX3,idXXX1", "idXXX2", "idXXX2,idXXX3"},
			expectedLocations: []*apploadbalancer.Location{
				{
					ZoneId:   "zone-A",
					SubnetId: "idXXX1",
				},
				{
					ZoneId:   "zone-B",
					SubnetId: "idXXX2",
				},
				{
					ZoneId:   "zone-C",
					SubnetId: "idXXX3",
				},
			},
			expectedNetwork: "idXXXXDefault",
		},
		{
			desc:           "two subnet in one zone",
			subnetIDStrs:   []string{"idXXX3,idXXX1", "idXXX2", "idXXX2,idXXX4"},
			wantResolveErr: true,
		},
		{
			desc:           "subnet in different networks",
			subnetIDStrs:   []string{"idXXX3,idXXX1", "idXXX3,idXXX5"},
			wantResolveErr: true,
		},
		{
			desc:           "subnet not found",
			subnetIDStrs:   []string{"idXXX3,idXXX1", "idXXX3,idXXX1,idXXX6"},
			wantResolveErr: true,
		},
		{
			desc:          "no subnets",
			subnetIDStrs:  []string{"", ",,,"},
			wantResultErr: true,
		},
	}

	ctrl := gomock.NewController(t)
	repo := mocks.NewMockSubnetRepository(ctrl)
	for subnetID := range subnets {
		repo.EXPECT().FindSubnetByID(gomock.Any(), subnetID).Return(subnets[subnetID], nil).AnyTimes()
	}
	repo.EXPECT().FindSubnetByID(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			r := NewResolvers(repo).Location()
			var err error
			for _, subnetIDstr := range tc.subnetIDStrs {
				if err = r.Resolve(subnetIDstr); err != nil {
					break
				}
			}
			require.True(t, (err != nil) == tc.wantResolveErr, "Resolve() error = %v)", err)
			if tc.wantResolveErr {
				return
			}
			network, locations, err := r.Result()
			require.True(t, (err != nil) == tc.wantResultErr, "Result() error = %v)", err)
			if tc.wantResolveErr {
				return
			}
			require.Equal(t, len(tc.expectedLocations), len(locations))
			assert.Equal(t, tc.expectedNetwork, network)
			for i := range tc.expectedLocations {
				comp := func() bool { return proto.Equal(tc.expectedLocations[i], locations[i]) }
				assert.Condition(t, comp, "expected %v, got %v", tc.expectedLocations[i], locations[i])
			}
		})
	}
}

func TestSecurityGroupIDs(t *testing.T) {
	testData := []struct {
		desc   string
		idStrs []string
		exp    []string
	}{
		{
			desc:   "all different",
			idStrs: []string{"idXXX3,idXXX1", "idXXX2,idXXX4"},
			exp:    []string{"idXXX3", "idXXX1", "idXXX2", "idXXX4"},
		},
		{
			desc:   "duplicates",
			idStrs: []string{"idXXX3,idXXX3", "idXXX3,idXXX4"},
			exp:    []string{"idXXX3", "idXXX4"},
		},
		{
			desc:   "no groups provided",
			idStrs: []string{""},
			exp:    nil,
		},
	}
	resolvers := &Resolvers{}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			r := resolvers.SecurityGroups()
			for _, idStr := range tc.idStrs {
				r.Resolve(idStr)
			}
			ret := r.Result()
			assert.Equal(t, tc.exp, ret)
		})
	}
}

func TestAutoScalePolicy(t *testing.T) {
	testData := []struct {
		desc         string
		minZoneSizes []string
		maxSizes     []string
		exp          *apploadbalancer.AutoScalePolicy
		expErr       bool
	}{
		{
			desc:         "OK",
			minZoneSizes: []string{"1", ""},
			maxSizes:     []string{"3", ""},
			exp: &apploadbalancer.AutoScalePolicy{
				MinZoneSize: 1,
				MaxSize:     3,
			},
		},
		{
			desc:         "multiple OK",
			minZoneSizes: []string{"1", "1", ""},
			maxSizes:     []string{"", "", ""},
			exp: &apploadbalancer.AutoScalePolicy{
				MinZoneSize: 1,
			},
		},
		{
			desc:         "multiple conflict",
			minZoneSizes: []string{"1", "2", ""},
			maxSizes:     []string{"", "4", ""},
			exp:          nil,
			expErr:       true,
		},
		{
			desc:         "OK empty",
			minZoneSizes: []string{"", ""},
			maxSizes:     []string{"", ""},
			exp:          nil,
		},
	}
	resolvers := &Resolvers{}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			r := resolvers.AutoScalePolicy()
			assert.Equal(t, len(tc.minZoneSizes), len(tc.maxSizes))
			var err error
			for i := range tc.minZoneSizes {
				err = r.Resolve(tc.minZoneSizes[i], tc.maxSizes[i])
				if err != nil {
					break
				}
			}
			if tc.expErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			ret := r.Result()
			assert.Equal(t, tc.exp, ret)
		})
	}
}

func TestVirtualHostOptsResolver_Resolve(t *testing.T) {
	testData := []struct {
		desc              string
		removeRespHeader  string
		replaceRespHeader string
		appendRespHeader  string
		renameRespHeader  string

		removeReqHeader  string
		replaceReqHeader string
		appendReqHeader  string
		renameReqHeader  string

		securityProfile string
		exp             VirtualHostResolveOpts
		wantErr         bool
	}{
		{
			desc:              "OK",
			removeRespHeader:  "toRemove=true,notToRemove=false",
			replaceRespHeader: "toReplace=replace,toReplaceTwo=replace_two",
			renameRespHeader:  "toRename=rename",
			appendRespHeader:  "toAppend=append",
			removeReqHeader:   "toRemove=true,notToRemove=false",
			replaceReqHeader:  "toReplace=replace,toReplaceTwo=replace_two",
			renameReqHeader:   "toRename=rename",
			appendReqHeader:   "toAppend=append",
			securityProfile:   "security-profile-id",
			exp: VirtualHostResolveOpts{
				SecurityProfileID: "security-profile-id",
				ModifyResponse: ModifyHeaderOpts{
					Append: map[string]string{
						"toAppend": "append",
					},
					Replace: map[string]string{
						"toReplace":    "replace",
						"toReplaceTwo": "replace_two",
					},
					Rename: map[string]string{
						"toRename": "rename",
					},
					Remove: map[string]bool{
						"toRemove":    true,
						"notToRemove": false,
					},
				},
				ModifyRequest: ModifyHeaderOpts{
					Append: map[string]string{
						"toAppend": "append",
					},
					Replace: map[string]string{
						"toReplace":    "replace",
						"toReplaceTwo": "replace_two",
					},
					Rename: map[string]string{
						"toRename": "rename",
					},
					Remove: map[string]bool{
						"toRemove":    true,
						"notToRemove": false,
					},
				},
			},
			wantErr: false,
		},
		{
			desc:    "empty",
			exp:     VirtualHostResolveOpts{},
			wantErr: false,
		},
		{
			desc:             "bad remove format",
			removeRespHeader: "toRemove=fals",
			exp:              VirtualHostResolveOpts{},
			wantErr:          true,
		},
	}
	resolvers := NewResolvers(nil)
	for _, tc := range testData {
		r := resolvers.VirtualHostOpts()
		t.Run(tc.desc, func(t *testing.T) {
			ret, err := r.Resolve(
				tc.removeRespHeader, tc.renameRespHeader, tc.appendRespHeader, tc.replaceRespHeader,
				tc.removeReqHeader, tc.renameReqHeader, tc.appendReqHeader, tc.replaceReqHeader,
				tc.securityProfile,
			)
			require.True(t, (err != nil) == tc.wantErr, "Result() error = %v)", err)
			if !tc.wantErr {
				assert.Equal(t, tc.exp, ret)
			}
		})
	}
}

func TestRouteOptsResolver(t *testing.T) {
	testData := []struct {
		desc           string
		timeout        string
		idleTimeout    string
		prefixRewrite  string
		upgradeTypes   string
		proto          string
		useRegex       string
		allowedMethods string
		exp            RouteResolveOpts
		wantErr        bool
	}{
		{
			desc:           "OK",
			timeout:        "10s",
			idleTimeout:    "1m",
			prefixRewrite:  "/apis/v1",
			upgradeTypes:   "websocket,other",
			proto:          "http2",
			useRegex:       "true",
			allowedMethods: "GET,POST",
			exp: RouteResolveOpts{
				Timeout:        &durationpb.Duration{Seconds: 10},
				IdleTimeout:    &durationpb.Duration{Seconds: 60},
				PrefixRewrite:  "/apis/v1",
				UpgradeTypes:   []string{"websocket", "other"},
				BackendType:    HTTP2,
				UseRegex:       true,
				AllowedMethods: []string{"GET", "POST"},
			},
			wantErr: false,
		},
		{
			desc:        "bad time format",
			idleTimeout: "1 minute",
			wantErr:     true,
		},
		{
			desc:    "empty",
			exp:     RouteResolveOpts{},
			wantErr: false,
		},
		{
			desc:    "unsupported protocol",
			proto:   "http76",
			exp:     RouteResolveOpts{},
			wantErr: true,
		},
		{
			desc:     "bad useRegex format",
			useRegex: "fals",
			exp:      RouteResolveOpts{},
			wantErr:  true,
		},
	}
	resolvers := NewResolvers(nil)
	for _, tc := range testData {
		r := resolvers.RouteOpts()
		t.Run(tc.desc, func(t *testing.T) {
			ret, err := r.Resolve(tc.timeout, tc.idleTimeout, tc.prefixRewrite, tc.upgradeTypes, tc.proto, tc.useRegex, tc.allowedMethods)
			require.True(t, (err != nil) == tc.wantErr, "Result() error = %v)", err)
			if !tc.wantErr {
				assert.Equal(t, tc.exp, ret)
			}
		})
	}
}

func TestBackendOptsResolver(t *testing.T) {
	hc1 := defaultHealthChecks

	hc2 := healthCheckTemplate()
	hc2.HealthcheckPort = 30102

	hc3 := &apploadbalancer.HealthCheck{
		Timeout:            &durationpb.Duration{Seconds: 10},
		Interval:           &durationpb.Duration{Seconds: 20},
		HealthyThreshold:   3,
		UnhealthyThreshold: 2,
		HealthcheckPort:    30103,
		Healthcheck: &apploadbalancer.HealthCheck_Http{
			Http: &apploadbalancer.HealthCheck_HttpHealthCheck{
				Path: "/health-1",
			},
		},
		TransportSettings: &apploadbalancer.HealthCheck_Plaintext{
			Plaintext: &apploadbalancer.PlaintextTransportSettings{},
		},
	}
	hc4 := healthCheckTemplate()
	hc4.HealthcheckPort = 30104
	hc4.Healthcheck = &apploadbalancer.HealthCheck_Grpc{
		Grpc: &apploadbalancer.HealthCheck_GrpcHealthCheck{
			ServiceName: "healthchecker",
		},
	}

	testData := []struct {
		desc                          string
		protocol                      string
		balancingMode                 string
		balancingPanicThreshold       string
		balancingLocalityAwareRouting string
		transportSecurity             string
		affinityHeader                string
		affinityCookie                string
		affinityConnection            string
		healthChecks                  string
		exp                           BackendResolveOpts
		wantErr                       bool
	}{
		{
			desc:                          "OK, case 1",
			protocol:                      "http",
			balancingMode:                 "mode-1",
			balancingPanicThreshold:       "10",
			balancingLocalityAwareRouting: "50",
			affinityHeader:                "name=name-1",
			exp: BackendResolveOpts{
				BackendType: HTTP,
				Secure:      false,
				LoadBalancingConfig: LoadBalancingConfig{
					Mode:                 "mode-1",
					PanicThreshold:       10,
					LocalityAwareRouting: 50,
				},
				healthChecks: hc1,
				affinityOpts: SessionAffinityOpts{
					header: &apploadbalancer.HeaderSessionAffinity{
						HeaderName: "name-1",
					},
				},
			},
			wantErr: false,
		},
		{
			desc:                    "OK, case 2",
			protocol:                "grpc",
			balancingMode:           "mode-1",
			balancingPanicThreshold: "12",
			transportSecurity:       "tls",
			affinityCookie:          "name=name-1,ttl=10s",
			healthChecks:            "port=30102",
			exp: BackendResolveOpts{
				BackendType: GRPC,
				Secure:      true,
				LoadBalancingConfig: LoadBalancingConfig{
					Mode:           "mode-1",
					PanicThreshold: 12,
				},
				healthChecks: []*apploadbalancer.HealthCheck{hc2},
				affinityOpts: SessionAffinityOpts{
					cookie: &apploadbalancer.CookieSessionAffinity{
						Name: "name-1",
						Ttl:  &durationpb.Duration{Seconds: 10},
					},
				},
			},
			wantErr: false,
		},
		{
			desc:               "OK, case 3",
			protocol:           "http2",
			balancingMode:      "mode-1",
			affinityConnection: "source-ip=true",
			healthChecks:       "port=30103,http-path=/health-1,timeout=10s,interval=20s,healthy-threshold=3,unhealthy-threshold=2",
			exp: BackendResolveOpts{
				BackendType: HTTP2,
				Secure:      false,
				LoadBalancingConfig: LoadBalancingConfig{
					Mode:           "mode-1",
					PanicThreshold: 0,
				},
				healthChecks: []*apploadbalancer.HealthCheck{hc3},
				affinityOpts: SessionAffinityOpts{
					connection: &apploadbalancer.ConnectionSessionAffinity{
						SourceIp: true,
					},
				},
			},
			wantErr: false,
		},
		{
			desc:               "OK, case 4",
			protocol:           "http2",
			balancingMode:      "mode-1",
			affinityConnection: "source-ip=true",
			healthChecks:       "port=30104,grpc-service-name=healthchecker",
			exp: BackendResolveOpts{
				BackendType: HTTP2,
				Secure:      false,
				LoadBalancingConfig: LoadBalancingConfig{
					Mode: "mode-1",
				},
				healthChecks: []*apploadbalancer.HealthCheck{hc4},
				affinityOpts: SessionAffinityOpts{
					connection: &apploadbalancer.ConnectionSessionAffinity{
						SourceIp: true,
					},
				},
			},
			wantErr: false,
		},
		{
			desc:               "Wrong, wrong affinity connection",
			protocol:           "http2",
			balancingMode:      "mode-1",
			affinityConnection: "source-ip=wrong",
			exp:                BackendResolveOpts{},
			wantErr:            true,
		},
		{
			desc:          "Wrong, wrong health checks timeout",
			protocol:      "http2",
			balancingMode: "mode-1",
			healthChecks:  "port=30100,timeout=wrong",
			exp:           BackendResolveOpts{},
			wantErr:       true,
		},
		{
			desc:          "Wrong, no port in health checks",
			protocol:      "http76",
			balancingMode: "mode-1",
			healthChecks:  "timeout=10s",
			exp:           BackendResolveOpts{},
			wantErr:       true,
		},
		{
			desc:                    "Wrong, panic threshold is not a number",
			protocol:                "http76",
			balancingMode:           "mode-1",
			balancingPanicThreshold: "wrong",
			exp:                     BackendResolveOpts{},
			wantErr:                 true,
		},
	}

	resolvers := NewResolvers(nil)
	for _, tc := range testData {
		r := resolvers.BackendOpts()
		t.Run(tc.desc, func(t *testing.T) {
			ret, err := r.Resolve(
				tc.protocol, tc.balancingMode, tc.balancingPanicThreshold, tc.balancingLocalityAwareRouting,
				tc.transportSecurity, tc.affinityHeader, tc.affinityCookie, tc.affinityConnection, tc.healthChecks,
			)
			require.True(t, (err != nil) == tc.wantErr, "Result() error = %v)", err)
			if !tc.wantErr {
				assert.Equal(t, tc.exp, ret)
			}
		})
	}
}
