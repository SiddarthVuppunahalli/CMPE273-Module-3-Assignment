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
	"github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/internal/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const shutdownTimeout = 5 * time.Second

type config struct {
	restAddress          string
	grpcAddress          string
	inventoryRESTURL     string
	inventoryGRPCAddress string
	inventoryTimeout     time.Duration
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	configuration, err := loadConfig()
	if err != nil {
		return err
	}

	inventoryConnection, err := grpc.NewClient(
		configuration.inventoryGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("create Inventory gRPC connection: %w", err)
	}
	defer inventoryConnection.Close()

	restInventoryClient := order.NewRESTInventoryClient(configuration.inventoryRESTURL, configuration.inventoryTimeout)
	grpcInventoryClient := order.NewGRPCInventoryClient(
		servicesv1.NewInventoryServiceClient(inventoryConnection),
		configuration.inventoryTimeout,
	)

	restServer := &http.Server{
		Addr:              configuration.restAddress,
		Handler:           order.NewRESTHandler(order.NewService(restInventoryClient)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	grpcServer := grpc.NewServer()
	servicesv1.RegisterOrderServiceServer(grpcServer, order.NewGRPCServer(order.NewService(grpcInventoryClient)))

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	restListener, err := net.Listen("tcp", configuration.restAddress)
	if err != nil {
		return fmt.Errorf("listen for REST on %s: %w", configuration.restAddress, err)
	}
	grpcListener, err := net.Listen("tcp", configuration.grpcAddress)
	if err != nil {
		_ = restListener.Close()
		return fmt.Errorf("listen for gRPC on %s: %w", configuration.grpcAddress, err)
	}

	log.Printf("order REST server listening on %s", configuration.restAddress)
	log.Printf("order gRPC server listening on %s", configuration.grpcAddress)
	log.Printf("inventory timeout/deadline: %s", configuration.inventoryTimeout)

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
		log.Print("shutting down order service")
	case serveErr = <-serverErrors:
	}

	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_NOT_SERVING)
	grpcServer.GracefulStop()
	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := restServer.Shutdown(shutdownContext); err != nil && serveErr == nil {
		serveErr = fmt.Errorf("shut down REST server: %w", err)
	}

	return serveErr
}

func loadConfig() (config, error) {
	timeout, err := positiveDurationFromEnv("ORDER_INVENTORY_TIMEOUT", time.Second)
	if err != nil {
		return config{}, err
	}

	return config{
		restAddress:          envOrDefault("ORDER_REST_ADDR", ":8080"),
		grpcAddress:          envOrDefault("ORDER_GRPC_ADDR", ":50050"),
		inventoryRESTURL:     envOrDefault("INVENTORY_REST_URL", "http://localhost:8081"),
		inventoryGRPCAddress: envOrDefault("INVENTORY_GRPC_ADDR", "localhost:50051"),
		inventoryTimeout:     timeout,
	}, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func positiveDurationFromEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}
	return duration, nil
}
