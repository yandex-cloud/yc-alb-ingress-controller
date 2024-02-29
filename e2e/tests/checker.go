//go:build e2e
// +build e2e

package tests

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
)

type TestCase struct {
	Name        string
	Description string
	Files       []string
	GoldenFile  string
	Checks      []Check
}

type Check struct {
	Ingress types.NamespacedName
	Host    string
	Proto   string
	Paths   []Path
}

type CheckBodyFunc func([]byte) error

type Path struct {
	Path string
	Code int

	CheckBody CheckBodyFunc
}

type Validator struct {
	Check
	k8scli *kubernetes.Clientset
}

func (v *Validator) Run() error {
	ing, err := v.k8scli.NetworkingV1().Ingresses(v.Ingress.Namespace).Get(context.Background(), v.Ingress.Name, v1.GetOptions{})
	if err != nil {
		return err
	}
	if len(ing.Status.LoadBalancer.Ingress) != 1 {
		return fmt.Errorf("invalid or unset Ingress %s/%s Status field", v.Ingress.Namespace, v.Ingress.Name)
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2: true,
			MaxIdleConns:      100,
			IdleConnTimeout:   90 * time.Second,
			TLSClientConfig: &tls.Config{
				ServerName:         v.Host,
				InsecureSkipVerify: true,
			},
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: simpleCheckRedirect,
	}
	for _, p := range v.Paths {
		url := fmt.Sprintf("%s://%s%s", v.Proto, ing.Status.LoadBalancer.Ingress[0].IP, p.Path)
		if k8s.IsIPv6(ing.Status.LoadBalancer.Ingress[0].IP) {
			url = fmt.Sprintf("%s://[%s]%s", v.Proto, ing.Status.LoadBalancer.Ingress[0].IP, p.Path)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		req.Host = v.Host
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != p.Code {
			return fmt.Errorf("expected status code %d for %s://%s%s, got %d", p.Code, v.Proto, v.Host, p.Path, resp.StatusCode)
		}

		if p.CheckBody != nil {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			err = p.CheckBody(body)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// simpleCheckRedirect is a highly primitive redirect policy, insofar as at the moment the only redirect we have and
// test is an http handler's redirecting to https if SNI match is present. It's only goal is to replace a fake host
// in the redirect URL with an actual balancer IP, so we shouldn't need to fake its DNS resolution
func simpleCheckRedirect(req *http.Request, via []*http.Request) error {
	host, _, err := net.SplitHostPort(req.URL.Host)
	if via[0].Host == host {
		req.URL.Host = via[0].URL.Host
		req.Host = host
	}
	return err
}
