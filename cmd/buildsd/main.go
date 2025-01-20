// cmd/buildsd/main.go

package main

import (
	buildv1 "builds/api/build"
	"builds/internal/server/api"
	"builds/internal/server/db"
	dbmodels "builds/internal/server/db/models"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

func main() {
	// Load the environment variables from the .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	flag.Parse()

	// Get the DATABASE_URL from the environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect to database
	gormDB, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the schema
	if err := autoMigrate(gormDB); err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}

	// Create server
	database := db.New(gormDB)
	server := api.NewServer(database)

	// Start gRPC server
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	buildv1.RegisterBuildServiceServer(grpcServer, server)

	// Handle shutdown gracefully
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Server listening at %v", listener.Addr())
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func autoMigrate(gormDB *gorm.DB) error {
	return gormDB.AutoMigrate(
		&dbmodels.Build{},
		&dbmodels.Environment{},
		&dbmodels.EnvironmentVariable{},
		&dbmodels.Hardware{},
		&dbmodels.GPU{},
		&dbmodels.Compiler{},
		&dbmodels.CompilerOption{},
		&dbmodels.CompilerOptimization{},
		&dbmodels.CompilerExtension{},
		&dbmodels.Command{},
		&dbmodels.CommandArgument{},
		&dbmodels.Output{},
		&dbmodels.Artifact{},
		&dbmodels.CompilerRemark{},
		&dbmodels.RemarkArg{},
		&dbmodels.ResourceUsage{},
		&dbmodels.Performance{},
		&dbmodels.PerformancePhase{},
	)
}
