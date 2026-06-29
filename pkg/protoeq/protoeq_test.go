package protoeq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// withUnknownField returns a clone of m carrying an unknown field, emulating a
// new computed field added to the ALB API that this proto version doesn't know.
func withUnknownField(m proto.Message) proto.Message {
	clone := proto.Clone(m)
	r := clone.ProtoReflect()
	unknown := protowire.AppendTag(nil, 99999, protowire.VarintType)
	unknown = protowire.AppendVarint(unknown, 1)
	r.SetUnknown(append(r.GetUnknown(), unknown...))
	return clone
}

func TestEqual_IgnoresUnknownFields(t *testing.T) {
	desired := &apploadbalancer.Target{
		AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "10.0.0.1"},
		SubnetId:    "subnet-1",
	}
	actual := withUnknownField(desired)

	// Sanity: plain proto.Equal regresses on unknown fields (the bug).
	assert.False(t, proto.Equal(desired, actual), "precondition: proto.Equal should differ on unknown field")
	// Our helper must treat them as equal.
	assert.True(t, Equal(desired, actual), "Equal must ignore unknown fields")
}

func TestEqual_DetectsRealDiff(t *testing.T) {
	a := &apploadbalancer.Target{
		AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "10.0.0.1"},
		SubnetId:    "subnet-1",
	}
	b := &apploadbalancer.Target{
		AddressType: &apploadbalancer.Target_IpAddress{IpAddress: "10.0.0.2"},
		SubnetId:    "subnet-1",
	}
	assert.False(t, Equal(a, b), "Equal must still detect real differences in known fields")
}
