package builders

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/proto"
	v1 "k8s.io/api/networking/v1"

	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/builders/mocks"
	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/metadata"
)

func TestSNIMatches(t *testing.T) {
	testData := []struct {
		desc     string
		tlsItems []*v1.IngressTLS
		exp      []*apploadbalancer.SniMatch
	}{
		{
			desc: "OK",
			tlsItems: []*v1.IngressTLS{
				{
					Hosts:      []string{"example1.com", "example2.com"},
					SecretName: "XXX1",
				},
				{
					Hosts:      []string{"example1.com", "example3.com"},
					SecretName: "XXX2",
				},
			},
			exp: []*apploadbalancer.SniMatch{
				{
					Name:        "sni-1954a6fc86a55a010c3c8e48f0603e956a6054ec",
					ServerNames: []string{"example1.com"},
					Handler: &apploadbalancer.TlsHandler{
						Handler: &apploadbalancer.TlsHandler_HttpHandler{
							HttpHandler: &apploadbalancer.HttpHandler{},
						},
						CertificateIds: []string{"XXX1", "XXX2"},
					},
				},
				{
					Name:        "sni-76386ba2c5cb3f2a6f21d0dbedc9519c118a910a",
					ServerNames: []string{"example2.com"},
					Handler: &apploadbalancer.TlsHandler{
						Handler: &apploadbalancer.TlsHandler_HttpHandler{
							HttpHandler: &apploadbalancer.HttpHandler{},
						},
						CertificateIds: []string{"XXX1"},
					},
				},
				{
					Name:        "sni-14c7e5889344f599818724afb051e23de0fc208c",
					ServerNames: []string{"example3.com"},
					Handler: &apploadbalancer.TlsHandler{
						Handler: &apploadbalancer.TlsHandler_HttpHandler{
							HttpHandler: &apploadbalancer.HttpHandler{},
						},
						CertificateIds: []string{"XXX2"},
					},
				},
			},
		},
		{
			desc: "duplicated host+cert pair",
			tlsItems: []*v1.IngressTLS{
				{
					Hosts:      []string{"example1.com", "example2.com"},
					SecretName: "XXX1",
				},
				{
					Hosts:      []string{"example1.com", "example3.com"},
					SecretName: "XXX2",
				},
				{
					Hosts:      []string{"example1.com"},
					SecretName: "XXX1",
				},
			},
			exp: []*apploadbalancer.SniMatch{
				{
					Name:        "sni-1954a6fc86a55a010c3c8e48f0603e956a6054ec",
					ServerNames: []string{"example1.com"},
					Handler: &apploadbalancer.TlsHandler{
						Handler: &apploadbalancer.TlsHandler_HttpHandler{
							HttpHandler: &apploadbalancer.HttpHandler{},
						},
						CertificateIds: []string{"XXX1", "XXX2"},
					},
				},
				{
					Name:        "sni-76386ba2c5cb3f2a6f21d0dbedc9519c118a910a",
					ServerNames: []string{"example2.com"},
					Handler: &apploadbalancer.TlsHandler{
						Handler: &apploadbalancer.TlsHandler_HttpHandler{
							HttpHandler: &apploadbalancer.HttpHandler{},
						},
						CertificateIds: []string{"XXX1"},
					},
				},
				{
					Name:        "sni-14c7e5889344f599818724afb051e23de0fc208c",
					ServerNames: []string{"example3.com"},
					Handler: &apploadbalancer.TlsHandler{
						Handler: &apploadbalancer.TlsHandler_HttpHandler{
							HttpHandler: &apploadbalancer.HttpHandler{},
						},
						CertificateIds: []string{"XXX2"},
					},
				},
			},
		},
	}
	tag := "tag"
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			tgRepo := mocks.NewMockTargetGroupFinder(ctrl)
			f := NewFactory("my-folder", "", &metadata.Names{ClusterID: "my-cluster"}, &metadata.Labels{ClusterID: "my-cluster"}, nil, tgRepo)

			b := f.HandlerBuilder(tag)
			for _, tls := range tc.tlsItems {
				b.AddCertificate(tls.Hosts, tls.SecretName)
			}
			res := b.Build()
			require.Equal(t, len(tc.exp), len(res))
			for i, sni := range tc.exp {
				comp := func() bool { return proto.Equal(sni, res[i]) }
				assert.Condition(t, comp, "SNI mismatch at position %d\nexp %v\ngot %v", i, sni, res[i])
			}
		})
	}
}
