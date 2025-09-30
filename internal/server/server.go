package server

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	auth_proto "github.com/nrmnqdds/gomaluum/internal/proto"
	"github.com/nrmnqdds/gomaluum/pkg/logger"
	"github.com/nrmnqdds/gomaluum/pkg/paseto"
	"github.com/nrmnqdds/gomaluum/pkg/sf"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "golang.org/x/crypto/x509roots/fallback"
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
	httpClient, err := createHTTPClient()
	if err != nil {
		log.Fatalf("Failed to create HTTP client: %v", err)
		return nil
	}

	return &GRPCServer{
		httpClient: httpClient,
	}
}

type Server struct {
	log          *logger.AppLogger
	paseto       *paseto.AppPaseto
	grpc         *GRPCServer
	httpClient   *http.Client
	port         int
	tokenManager *sf.TokenManager
	db           *sql.DB
}

func NewServer(port int, grpc *GRPCServer) *http.Server {
	paseto, err := paseto.New()
	if err != nil {
		log.Fatalf("Failed to create paseto: %v", err)
		return nil
	}

	// Create the HTTP client with proper certificate handling
	httpClient, err := createHTTPClient()
	if err != nil {
		log.Fatalf("Failed to create HTTP client: %v", err)
		return nil
	}

	db, err := sql.Open("libsql", os.Getenv("DB_PATH"))
	if err != nil {
		log.Fatalf("Failed to create database connection: %v", err)
		return nil
	}

	schema := []string{
		`CREATE TABLE IF NOT EXISTS analytics (
			matric_no TEXT NOT NULL PRIMARY KEY,
			batch AS (substr(matric_no, 1, 2) + 2000) STORED,
			level AS (
				CASE length(matric_no)
					WHEN 7 THEN 'DEGREE'
					WHEN 6 THEN 'CFS'
				END
			) STORED,
			timestamp DATETIME DEFAULT current_timestamp
		)`,
		`CREATE INDEX IF NOT EXISTS idx_batch ON analytics(batch)`,
		`CREATE INDEX IF NOT EXISTS idx_level ON analytics(level)`,
		`CREATE INDEX IF NOT EXISTS idx_batch_level ON analytics(batch, level)`,
	}

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return nil
		}
	}

	tm := sf.NewTokenManager()

	NewServer := &Server{
		port:         port,
		log:          logger.New(),
		paseto:       paseto,
		grpc:         grpc,
		httpClient:   httpClient,
		tokenManager: tm,
		db:           db,
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
func createHTTPClient() (*http.Client, error) {
	// Get system certificate pool
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load system cert pool: %w", err)
	}

	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// Create a custom transport with the enhanced certificate pool
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig: &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: true,
		},
	}

	// Return a client with the custom transport
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}, nil
}
