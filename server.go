package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/availproject/cdk-avail-da-server/da"
	"github.com/availproject/cdk-avail-da-server/rpc"
	"github.com/joho/godotenv"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Error loading .env file: %v", err)
		os.Exit(1)
	}

	availBackend, s3Backend, err := intializeServer()
	if err != nil {
		log.Printf("Failed to initialize server: %v", err)
		os.Exit(1)
	}

	// Set up the HTTP server with the RPC handler
	log.Println("Setting up HTTP server...")
	mux := http.NewServeMux()
	mux.Handle("/rpc", rpc.NewHandler(availBackend, s3Backend))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("Starting RPC server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}
	log.Println("Server stopped")
}

func intializeServer() (*da.AvailBackend, *da.S3Backend, error) {
	log.Println("Initializing server...")

	isBridgeEnabled, err := strconv.ParseBool(os.Getenv("IS_BRIDGE_ENABLED"))
	if err != nil {
		log.Printf("Invalid boolean value for IS_BRIDGE_ENABLED: %v", err)
		return nil, nil, err
	}

	var a *da.AvailBackend
	if isBridgeEnabled {
		attestorAddr := os.Getenv("ATTESTATION_CONTRACT_ADDRESS")
		if attestorAddr == "" {
			log.Printf("ATTESTATION_CONTRACT_ADDRESS is not set")
			return nil, nil, errors.New("ATTESTATION_CONTRACT_ADDRESS is not set")
		}

		l1_rpc_url := os.Getenv("L1_RPC_URL")
		if attestorAddr == "" {
			log.Printf("L1_RPC_URL is not set")
			return nil, nil, errors.New("L1_RPC_URL is not set")
		}

		avail_rpc_url := os.Getenv("AVAIL_RPC_URL")
		if avail_rpc_url == "" {
			log.Printf("AVAIL_RPC_URL is not set")
			return nil, nil, errors.New("AVAIL_RPC_URL is not set")
		}

		a, err = da.NewAvailBackend(true, attestorAddr, l1_rpc_url, avail_rpc_url)
		if err != nil {
			log.Printf("Failed to initialize Avail backend: %v", err)
			return nil, nil, err
		}
	} else {
		a, err = da.NewAvailBackend(false, "", "", "")
		if err != nil {
			log.Printf("Failed to initialize Avail backend: %v", err)
			return nil, nil, err
		}
		log.Println("Avail Bridge is not enabled, using default AvailBackend")
	}

	bucket := os.Getenv("S3_BUCKET")
	region := os.Getenv("S3_REGION")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	objectPrefix := os.Getenv("S3_OBJECT_PREFIX")

	if bucket == "" || region == "" || accessKey == "" || secretKey == "" {
		log.Printf("Missing required S3 configuration")
		return nil, nil, errors.New("missing required S3 configuration")
	}

	s, err := da.NewS3Backend(bucket, region, accessKey, secretKey, objectPrefix)
	if err != nil {
		log.Printf("Failed to initialize S3 backend: %v", err)
		return nil, nil, err
	}

	log.Println("Server initialized successfully")

	return a, s, nil
}
