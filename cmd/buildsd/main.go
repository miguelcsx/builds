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
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	host = flag.String("host", "", "The server host (default: all interfaces)")
	port = flag.Int("port", 50051, "The server port")
)

func getNetworkInterfaces() []string {
	var addresses []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return addresses
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.IsLinkLocalUnicast() {
					continue
				}
				if ipnet.IP.To4() != nil {
					addresses = append(addresses, ipnet.IP.String())
				}
			}
		}
	}
	return addresses
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	gormDB, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := autoMigrate(gormDB); err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}

	database := db.New(gormDB)
	srv := api.NewServer(database)

	grpcServer := grpc.NewServer()
	buildv1.RegisterBuildServiceServer(grpcServer, srv)

	addr := fmt.Sprintf("%s:%d", *host, *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a multiplexed handler that can handle both gRPC and HTTP/2
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && r.Header.Get("Content-Type") == "application/grpc" {
			grpcServer.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Builds Server - Use gRPC client to connect")
		}
	})

	h2sServer := &http.Server{
		Handler: h2c.NewHandler(httpHandler, &http2.Server{}),
	}

	// Print server addresses
	ips := getNetworkInterfaces()
	if len(ips) > 0 {
		log.Println("Server is available at:")
		for _, ip := range ips {
			log.Printf("  %s:%d\n", ip, *port)
		}
	} else {
		log.Printf("Server listening at %v\n", listener.Addr())
	}

	// Handle shutdown gracefully
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("\nShutting down server...")
		grpcServer.GracefulStop()
		h2sServer.Close()
	}()

	if err := h2sServer.Serve(listener); err != nil && err != http.ErrServerClosed {
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
		&dbmodels.KernelInfo{},
		&dbmodels.MemoryAccess{},
		&dbmodels.ResourceUsage{},
		&dbmodels.Performance{},
		&dbmodels.PerformancePhase{},
	)
}
