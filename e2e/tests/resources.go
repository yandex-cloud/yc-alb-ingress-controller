//go:build e2e
// +build e2e

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/yandex-cloud/alb-ingress/pkg/yc"
)

var setupLog logr.Logger

func buildSDK(keyFile string) (*ycsdk.SDK, error) {
	var creds ycsdk.Credentials
	if len(keyFile) != 0 {
		key, err := iamkey.ReadFromJSONFile(keyFile)
		if err != nil {
			return nil, err
		}
		creds, err = ycsdk.ServiceAccountKey(key)
		if err != nil {
			return nil, err
		}
	} else if token := os.Getenv("INGRESS_TOKEN"); token != "" {
		creds = ycsdk.NewIAMTokenCredentials(token)
	} else {
		return nil, fmt.Errorf("neither --keyfile flag nor INGRESS_TOKEN var has been provided")
	}
	return ycsdk.Build(context.Background(), ycsdk.Config{
		//Credentials: ycsdk.OAuthToken(token),
		Credentials: creds,
	})
}

// BalancerMessages is an intermediate struct for JSON serialization of objects containing proto.Message fields
// Generic solution is not needed at this moment
type BalancerMessages struct {
	Balancer      json.RawMessage   `json:"balancer,omitempty"`
	Router        json.RawMessage   `json:"router,omitempty"`
	TlsRouter     json.RawMessage   `json:"tls_router,omitempty"`
	BackendGroups []json.RawMessage `json:"backend_groups,omitempty"`
	TargetGroup   json.RawMessage   `json:"target_group,omitempty"`
}

type protoJSONHelper struct {
	err error
}

func (p *protoJSONHelper) marshal(message proto.Message) (msg json.RawMessage) {
	if p.err == nil {
		msg, p.err = protojson.Marshal(message)
	}
	return
}

func (p *protoJSONHelper) unmarshal(jmessage json.RawMessage, pmessage proto.Message) {
	if p.err == nil {
		p.err = protojson.Unmarshal(jmessage, pmessage)
	}
}

func FromBalancerResources(b *yc.BalancerResources) (*BalancerMessages, error) {
	pjh := &protoJSONHelper{}
	ret := &BalancerMessages{
		Balancer:  pjh.marshal(b.Balancer),
		Router:    pjh.marshal(b.Router),
		TlsRouter: pjh.marshal(b.TLSRouter),
	}
	if pjh.err != nil {
		return nil, pjh.err
	}
	return ret, nil
}

func ToBalancerResources(b *BalancerMessages) (*yc.BalancerResources, error) {
	pjh := &protoJSONHelper{}
	ret := &yc.BalancerResources{
		Balancer:  &apploadbalancer.LoadBalancer{},
		Router:    &apploadbalancer.HttpRouter{},
		TLSRouter: &apploadbalancer.HttpRouter{},
	}
	pjh.unmarshal(b.Balancer, ret.Balancer)
	pjh.unmarshal(b.Router, ret.Router)
	pjh.unmarshal(b.TlsRouter, ret.TLSRouter)
	if pjh.err != nil {
		return nil, pjh.err
	}
	return ret, nil
}
