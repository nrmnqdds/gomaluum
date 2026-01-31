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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/lib/pq"
)

type Handlers interface {
	LoginHandler(w http.ResponseWriter, r *http.Request)
	LogoutHandler(w http.ResponseWriter, r *http.Request)
	ProfileHandler(w http.ResponseWriter, r *http.Request)
	ScheduleHandler(w http.ResponseWriter, r *http.Request)
	ResultHandler(w http.ResponseWriter, r *http.Request)
}

// GRPCClients holds connections to multiple gRPC services
type GRPCClients struct {
	// GAS - Gomaluum Auth Service
	GASConn   *grpc.ClientConn
	GASClient auth_proto.AuthClient

	// GEI - Gomaluum Event Indexer (schedule indexer)
	// Add the proto client type when you have it, for example:
	// GEIConn   *grpc.ClientConn
	// GEIClient schedule_proto.ScheduleClient
	GEIConn *grpc.ClientConn
	// GEIClient will be added when you have the proto definition
}

// GRPCServiceConfig holds configuration for a single gRPC service
type GRPCServiceConfig struct {
	Name string
	URL  string
}

// NewGRPCClients creates and initializes all gRPC client connections
func NewGRPCClients(configs []GRPCServiceConfig) (*GRPCClients, error) {
	clients := &GRPCClients{}

	for _, config := range configs {
		conn, err := grpc.Dial(config.URL, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			// Close any previously opened connections before returning error
			clients.Close()
			return nil, fmt.Errorf("failed to connect to %s gRPC service at %s: %w", config.Name, config.URL, err)
		}

		// Route connection to appropriate service
		switch config.Name {
		case "GAS":
			clients.GASConn = conn
			clients.GASClient = auth_proto.NewAuthClient(conn)
			log.Printf("Connected to GAS (Auth Service) at %s", config.URL)
		case "GEI":
			clients.GEIConn = conn
			// When you have the proto definition, uncomment:
			// clients.GEIClient = schedule_proto.NewScheduleClient(conn)
			log.Printf("Connected to GEI (Schedule Indexer) at %s", config.URL)
		default:
			conn.Close()
			return nil, fmt.Errorf("unknown gRPC service: %s", config.Name)
		}
	}

	return clients, nil
}

// Close closes all gRPC connections
func (c *GRPCClients) Close() error {
	var lastErr error

	if c.GASConn != nil {
		if err := c.GASConn.Close(); err != nil {
			log.Printf("Error closing GAS connection: %v", err)
			lastErr = err
		}
	}

	if c.GEIConn != nil {
		if err := c.GEIConn.Close(); err != nil {
			log.Printf("Error closing GEI connection: %v", err)
			lastErr = err
		}
	}

	return lastErr
}

type Server struct {
	log          *logger.AppLogger
	paseto       *paseto.AppPaseto
	grpcClients  *GRPCClients
	httpClient   *http.Client
	port         int
	tokenManager *sf.TokenManager
	db           *sql.DB
}

func NewServer(port int, grpcClients *GRPCClients) *http.Server {
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

	var db *sql.DB
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		var err error
		db, err = sql.Open("postgres", databaseURL)
		if err != nil {
			log.Printf("Failed to create database connection: %v", err)
		} else {
			schema := []string{
				`CREATE TABLE IF NOT EXISTS analytics (
					matric_no VARCHAR(10) NOT NULL PRIMARY KEY,
					batch INTEGER GENERATED ALWAYS AS (CAST(SUBSTRING(matric_no, 1, 2) AS INTEGER) + 2000) STORED,
					level VARCHAR(10) GENERATED ALWAYS AS (
						CASE LENGTH(matric_no)
							WHEN 7 THEN 'DEGREE'
							WHEN 6 THEN 'CFS'
						END
					) STORED,
					timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
				)`,
				`CREATE INDEX IF NOT EXISTS idx_batch ON analytics(batch)`,
				`CREATE INDEX IF NOT EXISTS idx_level ON analytics(level)`,
				`CREATE INDEX IF NOT EXISTS idx_batch_level ON analytics(batch, level)`,
			}

			for _, stmt := range schema {
				if _, err := db.Exec(stmt); err != nil {
					log.Printf("Failed to create database schema: %v", err)
					db.Close()
					db = nil
					break
				}
			}
		}
	}

	tm := sf.NewTokenManager()

	NewServer := &Server{
		port:         port,
		log:          logger.New(),
		paseto:       paseto,
		grpcClients:  grpcClients,
		httpClient:   httpClient,
		tokenManager: tm,
		db:           db,
	}

	// Add cleanup for graceful shutdown
	if db != nil {
		// You can add a cleanup function or defer close if needed
		// For now, the connection will be cleaned up when the process exits
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

// Close closes the database connection and gRPC client connections if they exist
func (s *Server) Close() error {
	var dbErr, grpcErr error

	if s.db != nil {
		dbErr = s.db.Close()
	}

	if s.grpcClients != nil {
		grpcErr = s.grpcClients.Close()
	}

	if dbErr != nil {
		return dbErr
	}
	return grpcErr
}
