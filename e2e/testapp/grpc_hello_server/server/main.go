package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	helloworld "grpchello/hello"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

type MyServer struct {
	helloworld.UnimplementedGreeterServer
}

func (s *MyServer) SayHello(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: fmt.Sprintf("Hello %s", req.Name)}, nil
}

var _ helloworld.GreeterServer = &MyServer{}

const (
	certFile = "/etc/self_cert/hc_cert.pem"
	keyFile  = "/etc/self_cert/hc_key.pem"
)

func main() {
	check := func(err error, format string, args ...interface{}) {
		if err != nil {
			log.Fatalf(format+": %w", append(args, err)...)
		}
	}
	certPem, err := ioutil.ReadFile(certFile)
	check(err, "failed to open %s", certFile)
	keyPem, err := ioutil.ReadFile(keyFile)
	check(err, "failed to open %s", keyFile)
	certificate, err := tls.X509KeyPair(certPem, keyPem)
	check(err, "failed to create cert")
	creds := grpc.Creds(credentials.NewServerTLSFromCert(&certificate))
	log.Printf("tls configured")

	var wg sync.WaitGroup
	runServer := func(portName, srvDesc string, opts ...grpc.ServerOption) {
		port, err := strconv.Atoi(os.Getenv(portName))
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		check(err, "%s server failed to listen", srvDesc)
		s := grpc.NewServer(opts...)
		helloworld.RegisterGreeterServer(s, &MyServer{})
		reflection.Register(s)
		log.Printf("%s server listening at %v", lis.Addr(), srvDesc)
		check(s.Serve(lis), "%s server failed to serve", srvDesc)
		wg.Done()
	}
	wg.Add(2)
	go runServer("PORT", "plaintext")
	go runServer("SECURE_PORT", "secure", creds)
	wg.Wait()
}
