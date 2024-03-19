package tests

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	helloworld "go.opencensus.io/examples/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
)

const (
	grpcTLSPort       = "443"
	grpcPlaintextPort = "80"

	grpcRequestName = "egorushka"
)

type TestCase struct {
	Name        string
	Description string
	Files       []string
	GoldenFile  string
	Checks      []Check
}

type Check interface {
	Run(cli *kubernetes.Clientset) error
	Desc() string
}

type Validator struct {
	Check
	k8scli *kubernetes.Clientset
}

func (v *Validator) Run() error {
	return v.Check.Run(v.k8scli)
}

type GRPCCheck struct {
	Ingress types.NamespacedName
	Host    string
	UseTLS  bool
}

type HTTPCheck struct {
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

func (c *HTTPCheck) Desc() string {
	return fmt.Sprintf("url accessibility for host %s://%s of ingress %s/%s", c.Proto, c.Host, c.Ingress.Namespace, c.Ingress.Namespace)
}

func (c *HTTPCheck) Run(cli *kubernetes.Clientset) error {
	ing, err := cli.NetworkingV1().Ingresses(c.Ingress.Namespace).Get(context.Background(), c.Ingress.Name, v1.GetOptions{})
	if err != nil {
		return err
	}
	if len(ing.Status.LoadBalancer.Ingress) != 1 {
		return fmt.Errorf("invalid or unset Ingress %s/%s Status field", c.Ingress.Namespace, c.Ingress.Name)
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
				ServerName:         c.Host,
				InsecureSkipVerify: true,
			},
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: simpleCheckRedirect,
	}
	for _, p := range c.Paths {
		url := fmt.Sprintf("%s://%s%s", c.Proto, ing.Status.LoadBalancer.Ingress[0].IP, p.Path)
		if k8s.IsIPv6(ing.Status.LoadBalancer.Ingress[0].IP) {
			url = fmt.Sprintf("%s://[%s]%s", c.Proto, ing.Status.LoadBalancer.Ingress[0].IP, p.Path)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		req.Host = c.Host
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != p.Code {
			return fmt.Errorf("expected status code %d for %s://%s%s, got %d", p.Code, c.Proto, c.Host, p.Path, resp.StatusCode)
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

func (c *GRPCCheck) Desc() string {
	return fmt.Sprintf("grpc accessibility for host %s", c.Host)
}

func (c *GRPCCheck) Run(cli *kubernetes.Clientset) error {
	ing, err := cli.NetworkingV1().Ingresses(c.Ingress.Namespace).Get(context.Background(), c.Ingress.Name, v1.GetOptions{})
	if err != nil {
		return err
	}
	if len(ing.Status.LoadBalancer.Ingress) != 1 {
		return fmt.Errorf("invalid or unset Ingress %s/%s Status field", c.Ingress.Namespace, c.Ingress.Name)
	}

	var conn *grpc.ClientConn
	if c.UseTLS {
		creds := credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
		target := fmt.Sprintf("%s:%s", c.Host, grpcTLSPort)
		conn, err = grpc.Dial(target, grpc.WithTransportCredentials(creds))
		if err != nil {
			return fmt.Errorf("error dialing grpc %w", err)
		}
	} else {
		target := fmt.Sprintf("%s:%s", c.Host, grpcPlaintextPort)
		conn, err = grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	client := helloworld.NewGreeterClient(conn)
	msg, err := client.SayHello(context.Background(), &helloworld.HelloRequest{
		Name: grpcRequestName,
	})
	if err != nil {
		return err
	}

	exp := fmt.Sprintf("Hello %s", grpcRequestName)
	if msg.Message != exp {
		return fmt.Errorf("expected result is %s, but got %s", exp, msg.Message)
	}

	return nil
}
