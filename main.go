package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nrmnqdds/gomaluum/internal/server"

	"github.com/jwalton/gchalk"
	_ "golang.org/x/crypto/x509roots/fallback"
)

//go:embed docs/*
var DocsPath embed.FS

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

// @title           Gomaluum API Server
// @version         2.0
// @description     This is the API server for Gomaluum, an API that serves i-Ma'luum data for ease of developer.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Quddus
// @contact.url    http://www.swagger.io/support
// @contact.email  ceo@nrmnqdds.com

// @license.name  Bantown Public License
// @license.url   https://github.com/nrmnqdds/gomaluum-api/blob/main/LICENSE.md
func main() {
	// Define mode flags
	var dev bool
	var prod bool
	flag.BoolVar(&dev, "d", false, "run in development mode")
	flag.BoolVar(&prod, "p", false, "run in production mode")
	flag.Parse()

	// Check if exactly one mode is selected
	if !dev && !prod {
		log.Fatal("Error: Must specify either development (-d) or production (-p) mode")
	}
	if dev && prod {
		log.Fatal("Error: Cannot specify both development (-d) and production (-p) mode")
	}

	if dev {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error reading .env file!", err)
		}
		log.Println(".env file loaded")
		log.Println("Running in development mode")
	} else {
		log.Println("Running in production mode")
	}

	// Get gRPC service URL from environment
	grpcServiceURL := os.Getenv("GRPC_SERVICE_URL")
	if grpcServiceURL == "" {
		log.Fatal("GRPC_SERVICE_URL environment variable is required")
	}

	// Initialize gRPC client connection to external service
	grpcClient, err := server.NewGRPCClient(grpcServiceURL)
	if err != nil {
		log.Fatalf("failed to connect to gRPC service: %v", err)
	}
	defer grpcClient.Close()

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("failed to convert PORT to int: %v", err)
	}

	server.DocsPath = DocsPath
	httpServer := server.NewServer(port, grpcClient)

	// Create channels to track server status
	done := make(chan bool, 1)
	httpReady := make(chan bool, 1)

	// myFigure := figure.NewFigure("GoMaluum Rest API", "", true)
	// myFigure.Print()
	fmt.Println(gchalk.Blue(`

 ██████╗  ██████╗ ███╗   ███╗ █████╗ ██╗     ██╗   ██╗██╗   ██╗███╗   ███╗
██╔════╝ ██╔═══██╗████╗ ████║██╔══██╗██║     ██║   ██║██║   ██║████╗ ████║
██║  ███╗██║   ██║██╔████╔██║███████║██║     ██║   ██║██║   ██║██╔████╔██║
██║   ██║██║   ██║██║╚██╔╝██║██╔══██║██║     ██║   ██║██║   ██║██║╚██╔╝██║
╚██████╔╝╚██████╔╝██║ ╚═╝ ██║██║  ██║███████╗╚██████╔╝╚██████╔╝██║ ╚═╝ ██║
 ╚═════╝  ╚═════╝ ╚═╝     ╚═╝╚═╝  ╚═╝╚══════╝ ╚═════╝  ╚═════╝ ╚═╝     ╚═╝

		`))

	fmt.Println(gchalk.Yellow("====================================================="))
	fmt.Println(gchalk.Green(fmt.Sprintf("Connected to gRPC service at %s", grpcServiceURL)))

	// Start HTTP server
	go func() {
		fmt.Println(gchalk.Blue(fmt.Sprintf("HTTP server listening on :%d", port)))
		httpReady <- true
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("http server error: %s", err))
		}
	}()

	// Wait for HTTP server to be ready and print final separator
	go func() {
		<-httpReady
		fmt.Println(gchalk.Yellow("====================================================="))
	}()

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(httpServer, done)

	// Wait for the graceful shutdown to complete
	<-done
	log.Println("Graceful shutdown complete.")
}
