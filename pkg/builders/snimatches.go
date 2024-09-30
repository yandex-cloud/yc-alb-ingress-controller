package builders

import (
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"

	"github.com/yandex-cloud/yc-alb-ingress-controller/pkg/metadata"
)

type hostAndCert struct {
	host, cert string
}
type HandlerBuilder struct {
	names *metadata.Names
	tag   string

	// keep the order in which hosts are fed to this builder so that map randomization shouldn't cause updates
	hostsOrder []string
	// collect certificates per host and ensure no duplicated hosts
	certs map[string][]string
	// ensure no duplicated certificates for the same hosts
	hostAndCerts map[hostAndCert]struct{}
}

func (b *HandlerBuilder) AddCertificate(hosts []string, certID string) {
	for _, host := range hosts {
		certsForHost, ok := b.certs[host]
		if !ok {
			b.hostsOrder = append(b.hostsOrder, host)
		}
		hc := hostAndCert{host: host, cert: certID}
		if _, ok := b.hostAndCerts[hc]; !ok {
			b.certs[host] = append(certsForHost, certID)
			b.hostAndCerts[hc] = struct{}{}
		}
	}
}

func (b *HandlerBuilder) Build() []*apploadbalancer.SniMatch {
	var ret []*apploadbalancer.SniMatch

	for _, host := range b.hostsOrder {
		certificateIDs := b.certs[host]
		sniName := b.names.SNIMatchForHost(b.tag, host)
		sniHandler := &apploadbalancer.TlsHandler{
			Handler: &apploadbalancer.TlsHandler_HttpHandler{
				HttpHandler: &apploadbalancer.HttpHandler{},
			},
			CertificateIds: certificateIDs,
		}
		ret = append(ret, &apploadbalancer.SniMatch{
			Name:        sniName,
			ServerNames: []string{host},
			Handler:     sniHandler,
		})
	}
	return ret
}
