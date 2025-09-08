package auth

import (
	"fmt"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/sonic-net/sonic-gnmi/sonic-gnmi-standalone/pkg/cert"
)

// NewServerWithAuth creates a new gRPC server with authorization middleware.
func NewServerWithAuth(addr string, certMgr cert.CertificateManager, authorizer Authorizer) (*grpc.Server, error) {
	glog.V(1).Info("Creating gRPC server with authorization middleware")

	// Build server options
	var opts []grpc.ServerOption

	// Add TLS if certificate manager is provided
	if certMgr != nil {
		tlsConfig, err := certMgr.GetTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get TLS config: %w", err)
		}

		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.Creds(creds))
		glog.V(1).Info("Added TLS credentials to server")
	} else {
		glog.V(1).Info("Creating insecure server (no certificate manager)")
	}

	// Add authorization interceptors
	if authorizer != nil {
		opts = append(opts,
			grpc.UnaryInterceptor(AuthMiddleware(authorizer)),
			grpc.StreamInterceptor(StreamAuthMiddleware(authorizer)),
		)
		glog.V(1).Info("Added authorization interceptors")
	} else {
		glog.V(1).Info("No authorization - creating server without auth middleware")
	}

	// Create the server
	server := grpc.NewServer(opts...)

	// Register reflection service for development tools
	reflection.Register(server)
	glog.V(2).Info("Registered gRPC reflection service")

	return server, nil
}

// NewInsecureServerWithAuth creates an insecure server with authorization.
func NewInsecureServerWithAuth(authorizer Authorizer) *grpc.Server {
	glog.V(1).Info("Creating insecure gRPC server with authorization")

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(AuthMiddleware(authorizer)),
		grpc.StreamInterceptor(StreamAuthMiddleware(authorizer)),
	}

	// nosemgrep: go.grpc.security.grpc-server-insecure-connection.grpc-server-insecure-connection
	server := grpc.NewServer(opts...)
	reflection.Register(server)

	return server
}
