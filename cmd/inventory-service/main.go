package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	servicesv1 "github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/gen/services/v1"
	"github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/internal/inventory"
	"google.golang.org/grpc"
)

const shutdownTimeout = 5 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	restAddress := envOrDefault("INVENTORY_REST_ADDR", ":8081")
	grpcAddress := envOrDefault("INVENTORY_GRPC_ADDR", ":50051")

	delay, err := durationFromEnv("INVENTORY_DELAY", 0)
	if err != nil {
		return err
	}

	checker := inventory.WithDelay(inventory.NewDefaultStore(), delay)
	restServer := &http.Server{
		Addr:              restAddress,
		Handler:           inventory.NewRESTHandler(checker),
		ReadHeaderTimeout: 5 * time.Second,
	}
	grpcServer := grpc.NewServer()
	servicesv1.RegisterInventoryServiceServer(grpcServer, inventory.NewGRPCServer(checker))

	restListener, err := net.Listen("tcp", restAddress)
	if err != nil {
		return fmt.Errorf("listen for REST on %s: %w", restAddress, err)
	}
	grpcListener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		_ = restListener.Close()
		return fmt.Errorf("listen for gRPC on %s: %w", grpcAddress, err)
	}

	log.Printf("inventory REST server listening on %s", restAddress)
	log.Printf("inventory gRPC server listening on %s", grpcAddress)
	if delay > 0 {
		log.Printf("inventory artificial delay enabled: %s", delay)
	}

	serverErrors := make(chan error, 2)
	go func() {
		if err := restServer.Serve(restListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("serve REST: %w", err)
		}
	}()
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			serverErrors <- fmt.Errorf("serve gRPC: %w", err)
		}
	}()

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var serveErr error
	select {
	case <-signalContext.Done():
		log.Print("shutting down inventory service")
	case serveErr = <-serverErrors:
	}

	grpcServer.GracefulStop()
	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := restServer.Shutdown(shutdownContext); err != nil && serveErr == nil {
		serveErr = fmt.Errorf("shut down REST server: %w", err)
	}

	return serveErr
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func durationFromEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration < 0 {
		return 0, fmt.Errorf("%s must not be negative", name)
	}
	return duration, nil
}
