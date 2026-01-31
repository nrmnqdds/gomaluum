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

	// Get gRPC service URLs from environment
	gasServiceURL := os.Getenv("GAS_SERVICE_URL") // Auth service
	geiServiceURL := os.Getenv("GEI_SERVICE_URL") // Schedule indexer service

	// Validate required services
	if gasServiceURL == "" {
		log.Fatal("GAS_SERVICE_URL environment variable is required")
	}
	if geiServiceURL == "" {
		log.Fatal("GEI_SERVICE_URL environment variable is required")
	}

	// Configure all gRPC services
	grpcConfigs := []server.GRPCServiceConfig{
		{
			Name: "GAS",
			URL:  gasServiceURL,
		},
		{
			Name: "GEI",
			URL:  geiServiceURL,
		},
	}

	// Initialize all gRPC client connections
	grpcClients, err := server.NewGRPCClients(grpcConfigs)
	if err != nil {
		log.Fatalf("failed to connect to gRPC services: %v", err)
	}
	defer grpcClients.Close()

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("failed to convert PORT to int: %v", err)
	}

	server.DocsPath = DocsPath
	httpServer := server.NewServer(port, grpcClients)

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
	fmt.Println(gchalk.Green("Connected to gRPC services:"))
	fmt.Println(gchalk.Green(fmt.Sprintf("  • GAS (Auth Service): %s", gasServiceURL)))
	fmt.Println(gchalk.Green(fmt.Sprintf("  • GEI (Schedule Indexer): %s", geiServiceURL)))

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
