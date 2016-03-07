package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	pb "github.com/creiht/formic/proto"
	"github.com/pandemicsyn/ftls"

	"net"
)

var (
	usetls              = flag.Bool("tls", true, "Connection uses TLS if true, else plain TCP")
	certFile            = flag.String("cert_file", "/etc/oort/server.crt", "The TLS cert file")
	keyFile             = flag.String("key_file", "/etc/oort/server.key", "The TLS key file")
	port                = flag.Int("port", 9443, "The server port")
	oortValueHost       = flag.String("oortvaluehost", "127.0.0.1:6379", "host:port to use when connecting to oort value")
	oortGroupHost       = flag.String("oortgrouphost", "127.0.0.1:6380", "host:port to use when connecting to oort group")
	insecureSkipVerify  = flag.Bool("skipverify", false, "don't verify cert")
	oortClientMutualTLS = flag.Bool("mutualtls", false, "whether or not the server expects mutual tls auth")
	oortClientCert      = flag.String("oort-client-cert", "/etc/oort/client.crt", "cert file to use")
	oortClientKey       = flag.String("oort-client-key", "/etc/oort/client.key", "key file to use")
	oortClientCA        = flag.String("oort-client-ca", "/etc/oort/ca.pem", "ca file to use")
)

// FatalIf is just a lazy log/panic on error func
func FatalIf(err error, msg string) {
	if err != nil {
		grpclog.Fatalf("%s: %v", msg, err)
	}
}

func main() {
	flag.Parse()

	envtls := os.Getenv("FORMICD_TLS")
	if envtls == "true" {
		*usetls = true
	}

	envoortvhost := os.Getenv("FORMICD_OORT_VALUE_HOST")
	if envoortvhost != "" {
		*oortValueHost = envoortvhost
	}

	envoortghost := os.Getenv("FORMICD_OORT_GROUP_HOST")
	if envoortghost != "" {
		*oortGroupHost = envoortghost
	}

	envport := os.Getenv("FORMICD_PORT")
	if envport != "" {
		p, err := strconv.Atoi(envport)
		if err != nil {
			log.Println("Did not send valid port from env:", err)
		} else {
			*port = p
		}
	}

	envcert := os.Getenv("FORMICD_CERT_FILE")
	if envcert != "" {
		*certFile = envcert
	}

	envkey := os.Getenv("FORMICD_KEY_FILE")
	if envkey != "" {
		*keyFile = envkey
	}
	envSkipVerify := os.Getenv("FORMICD_INSECURE_SKIP_VERIFY")
	if envSkipVerify == "true" {
		*insecureSkipVerify = true
	}
	envMutualTLS := os.Getenv("FORMICD_MUTUAL_TLS")
	if envMutualTLS == "true" {
		*oortClientMutualTLS = true
	}
	envClientCA := os.Getenv("FORMICD_CLIENT_CA_FILE")
	if envClientCA != "" {
		*oortClientCA = envClientCA
	}
	envClientCert := os.Getenv("FORMICD_CLIENT_CERT_FILE")
	if envClientCert != "" {
		*oortClientCert = envClientCert
	}
	envClientKey := os.Getenv("FORMICD_CLIENT_KEY_FILE")
	if envClientKey != "" {
		*oortClientKey = envClientKey
	}

	var opts []grpc.ServerOption
	if *usetls {
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		FatalIf(err, "Couldn't load cert from file")
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	s := grpc.NewServer(opts...)
	copt, err := ftls.NewGRPCClientDialOpt(&ftls.Config{
		MutualTLS:          *oortClientMutualTLS,
		InsecureSkipVerify: *insecureSkipVerify,
		CertFile:           *oortClientCert,
		KeyFile:            *oortClientKey,
		CAFile:             *oortClientCA,
	})
	if err != nil {
		grpclog.Fatalln("Cannot setup tls config:", err)
	}
	fs, err := NewOortFS(*oortValueHost, *oortGroupHost, copt)
	if err != nil {
		grpclog.Fatalln(err)
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	FatalIf(err, "Failed to bind to port")
	pb.RegisterApiServer(s, NewApiServer(fs))
	grpclog.Printf("Starting up on %d...\n", *port)
	s.Serve(l)
}
