# Use an official Golang image
FROM golang:1.23.3

WORKDIR /app

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the server binary
RUN go build -o /bin/buildsctl ./cmd/buildsctl

CMD ["/bin/buildsctl"]
