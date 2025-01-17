package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	pb "github.com/nrmnqdds/gomaluum/internal/proto"
	"github.com/nrmnqdds/gomaluum/pkg/cloudflare"
	"github.com/nrmnqdds/gomaluum/pkg/logger"
	"github.com/nrmnqdds/gomaluum/pkg/paseto"
)

type Proto struct {
	pb.UnimplementedAuthServer
}

type Server struct {
	log    *logger.AppLogger
	cf     *cloudflare.AppCloudflare
	paseto *paseto.AppPaseto
	pb     *Proto
	port   int
}

func NewServer(port int) *http.Server {
	paseto, err := paseto.New()
	if err != nil {
		log.Fatalf("Failed to create paseto: %v", err)
		return nil
	}

	NewServer := &Server{
		port:   port,
		log:    logger.New(),
		cf:     cloudflare.New(),
		paseto: paseto,
		pb:     &Proto{},
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
