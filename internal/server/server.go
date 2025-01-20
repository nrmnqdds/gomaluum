package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	auth_proto "github.com/nrmnqdds/gomaluum/internal/proto"
	"github.com/nrmnqdds/gomaluum/pkg/logger"
	"github.com/nrmnqdds/gomaluum/pkg/paseto"
)

type GRPCServer struct {
	auth_proto.UnimplementedAuthServer
}

func NewGRPCServer() *GRPCServer {
	return &GRPCServer{}
}

type Server struct {
	log    *logger.AppLogger
	paseto *paseto.AppPaseto
	grpc   *GRPCServer
	port   int
}

func NewServer(port int, grpc *GRPCServer) *http.Server {
	paseto, err := paseto.New()
	if err != nil {
		log.Fatalf("Failed to create paseto: %v", err)
		return nil
	}

	NewServer := &Server{
		port:   port,
		log:    logger.New(),
		paseto: paseto,
		grpc:   grpc,
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
