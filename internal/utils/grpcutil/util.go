// internal/utils/grpcutil/util.go

package grpcutil

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func CreateGRPCConnection(addr string, useTLS bool) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	// Handle ngrok-specific configuration
	if strings.Contains(addr, "ngrok-free.app") {
		// Ensure proper formatting for ngrok URLs
		if !strings.HasPrefix(addr, "https://") {
			addr = "https://" + addr
		}

		// Parse the URL to get the host
		u, err := url.Parse(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid ngrok URL: %v", err)
		}

		// Enhanced TLS config for ngrok with proper ALPN settings
		config := &tls.Config{
			ServerName:         u.Hostname(),
			InsecureSkipVerify: true,
			NextProtos:         []string{"h2", "http/1.1"},
			MinVersion:         tls.VersionTLS12,
		}

		opts = append(opts,
			grpc.WithTransportCredentials(credentials.NewTLS(config)),
			grpc.WithAuthority(u.Hostname()),
			grpc.WithDisableRetry(),
			grpc.WithBlock(),
			grpc.WithUserAgent("grpc-go/1.0"),
		)

		dialAddr := u.Hostname() + ":443"
		return grpc.DialContext(context.Background(), dialAddr, opts...)
	}

	// Handle HTTP/HTTPS URLs
	if strings.HasPrefix(addr, "http://") {
		addr = strings.TrimPrefix(addr, "http://")
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else if strings.HasPrefix(addr, "https://") {
		addr = strings.TrimPrefix(addr, "https://")
		host := addr
		if strings.Contains(addr, ":") {
			host, _, _ = net.SplitHostPort(addr)
		}
		config := &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: true,
			NextProtos:         []string{"h2", "http/1.1"},
			MinVersion:         tls.VersionTLS12,
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(config)))
	} else {
		// Plain TCP connection
		if useTLS {
			host := addr
			if strings.Contains(addr, ":") {
				host, _, _ = net.SplitHostPort(addr)
			}
			config := &tls.Config{
				ServerName:         host,
				InsecureSkipVerify: true,
				NextProtos:         []string{"h2", "http/1.1"},
				MinVersion:         tls.VersionTLS12,
			}
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(config)))
		} else {
			opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}
	}

	// General options for all non-ngrok connections
	opts = append(opts,
		grpc.WithBlock(),
		grpc.WithDisableRetry(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return grpc.DialContext(ctx, addr, opts...)
}
