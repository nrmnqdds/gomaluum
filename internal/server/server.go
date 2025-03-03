package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"time"

	auth_proto "github.com/nrmnqdds/gomaluum/internal/proto"
	"github.com/nrmnqdds/gomaluum/pkg/logger"
	"github.com/nrmnqdds/gomaluum/pkg/paseto"
)

type Handlers interface {
	LoginHandler(w http.ResponseWriter, r *http.Request)
	LogoutHandler(w http.ResponseWriter, r *http.Request)
	ProfileHandler(w http.ResponseWriter, r *http.Request)
	ScheduleHandler(w http.ResponseWriter, r *http.Request)
	ResultHandler(w http.ResponseWriter, r *http.Request)
}

type GRPCServer struct {
	auth_proto.UnimplementedAuthServer
	httpClient *http.Client
}

func NewGRPCServer() *GRPCServer {
	// Create the HTTP client with proper certificate handling
	httpClient, err := CreateHTTPClient()
	if err != nil {
		log.Fatalf("Failed to create HTTP client: %v", err)
		return nil
	}

	return &GRPCServer{
		httpClient: httpClient,
	}
}

type Server struct {
	log        *logger.AppLogger
	paseto     *paseto.AppPaseto
	grpc       *GRPCServer
	httpClient *http.Client
	port       int
}

func NewServer(port int, grpc *GRPCServer) *http.Server {
	paseto, err := paseto.New()
	if err != nil {
		log.Fatalf("Failed to create paseto: %v", err)
		return nil
	}

	// Create the HTTP client with proper certificate handling
	httpClient, err := CreateHTTPClient()
	if err != nil {
		log.Fatalf("Failed to create HTTP client: %v", err)
		return nil
	}

	NewServer := &Server{
		port:       port,
		log:        logger.New(),
		paseto:     paseto,
		grpc:       grpc,
		httpClient: httpClient,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}

// CreateHTTPClient returns an HTTP client configured with system and custom certificates
func CreateHTTPClient() (*http.Client, error) {
	// Get system certificate pool
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load system cert pool: %w", err)
	}

	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// Optionally load custom certificates if needed
	// Uncomment if you have specific certificates to add
	/*
		customCert, err := ioutil.ReadFile("/etc/ssl/custom-cert.pem")
		if err != nil {
			return nil, fmt.Errorf("failed to read custom cert: %w", err)
		}

		if ok := rootCAs.AppendCertsFromPEM(customCert); !ok {
			return nil, fmt.Errorf("failed to append custom cert")
		}
	*/

	// Create a custom transport with the enhanced certificate pool
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: rootCAs,
		},
	}

	// Return a client with the custom transport
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}, nil
}
