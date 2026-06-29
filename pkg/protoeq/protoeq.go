package protoeq

import (
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

// Equal is like proto.Equal but ignores unknown fields, so new computed
// fields in ALB API don't cause phantom diffs and infinite update loops.
func Equal(x, y proto.Message) bool {
	return cmp.Equal(x, y, protocmp.Transform(), protocmp.IgnoreUnknown())
}
