# Variables
PROTO_DIR := "proto"
API_DIR := "api"

# Compile everything
all: clean proto build

# Clean generated files
clean:
	rm -rf {{API_DIR}}
	rm -rf bin/
	find . -type f -name '*.pb.go' -delete

# Generate proto buffers
proto:
	mkdir -p {{API_DIR}}
	protoc -I={{PROTO_DIR}} \
		--go_out={{API_DIR}} --go_opt=paths=source_relative \
		--go-grpc_out={{API_DIR}} --go-grpc_opt=paths=source_relative \
		{{PROTO_DIR}}/build/*.proto

# Build binaries
build:
	go build -o bin/builds ./cmd/builds
	go build -o bin/buildsd ./cmd/buildsd
	go build -o bin/buildsctl ./cmd/buildsctl

# Build Docker images
docker-build:
	docker-compose build

# Start services
docker-up:
	@just docker-build
	docker-compose up

# Stop services
docker-down:
	docker-compose down
