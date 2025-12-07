package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/di"
	pb "github.com/Code0716/stock-price-repository/pb"
)

func main() {
	ctx := context.Background()

	// Load configuration
	config.LoadEnvConfig()

	// Initialize dependencies
	components, cleanup, err := di.InitializeStockServiceServer(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize gRPC server: %v", err)
	}
	defer cleanup()

	stockServiceServer := components.Server
	logger := components.Logger
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Failed to sync logger: %v", err)
		}
	}()

	// Create gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterStockServiceServer(grpcServer, stockServiceServer)

	// Enable reflection for development
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "local" || appEnv == "dev" {
		reflection.Register(grpcServer)
		logger.Info("gRPC reflection enabled")
	}

	// Start server
	port := getPort()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on port %s: %v", port, err))
		return
	}

	logger.Info(fmt.Sprintf("gRPC server listening on port %s", port))

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("Shutting down gRPC server...")
	case err := <-errCh:
		logger.Error(fmt.Sprintf("Failed to serve: %v", err))
	}

	grpcServer.GracefulStop()
	logger.Info("gRPC server stopped")
}

func getPort() string {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}
	return port
}
