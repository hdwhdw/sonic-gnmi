// Package server provides a builder pattern for creating gRPC servers with configurable services.
//
// The ServerBuilder pattern provides:
//   - Clean separation between infrastructure and services
//   - Dynamic service enablement/disablement
//   - Programmatic TLS/mTLS configuration
//   - Configuration-driven service selection
//   - Easy extension for new services
//
// Example usage:
//
//	// Basic server with gNOI System service
//	srv, err := server.NewServerBuilder().
//	    WithAddress(":50055").
//	    WithRootFS("/mnt/host").
//	    EnableGNOISystem().
//	    Build()
//
//	// Server with TLS
//	srv, err := server.NewServerBuilder().
//	    WithAddress(":50055").
//	    WithTLS("server.crt", "server.key").
//	    EnableGNOISystem().
//	    Build()
//
//	// Server with mTLS
//	srv, err := server.NewServerBuilder().
//	    WithAddress(":50055").
//	    WithMTLS("server.crt", "server.key", "ca.crt").
//	    EnableServices([]string{"gnoi.system"}).
//	    Build()
package server

import (
	"github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnoi/file"
	"github.com/openconfig/gnoi/system"

	"github.com/sonic-net/sonic-gnmi/sonic-gnmi-standalone/pkg/server/config"
	gnmiserver "github.com/sonic-net/sonic-gnmi/sonic-gnmi-standalone/pkg/server/gnmi"
	gnoiFile "github.com/sonic-net/sonic-gnmi/sonic-gnmi-standalone/pkg/server/gnoi/file"
	gnoiSystem "github.com/sonic-net/sonic-gnmi/sonic-gnmi-standalone/pkg/server/gnoi/system"
)

// ServerBuilder provides a fluent interface for configuring and building a gRPC server
// with various SONiC services. It follows the builder pattern to allow selective
// enabling/disabling of services based on deployment requirements.
type ServerBuilder struct {
	addr      string
	rootFS    string
	services  map[string]bool
	tlsConfig *tlsConfig
}

// tlsConfig holds TLS configuration for the server builder.
type tlsConfig struct {
	enabled     bool
	mtlsEnabled bool
	certFile    string
	keyFile     string
	caCertFile  string
}

// NewServerBuilder creates a new ServerBuilder instance with default configuration.
// Services are disabled by default and must be explicitly enabled.
func NewServerBuilder() *ServerBuilder {
	return &ServerBuilder{
		services: make(map[string]bool),
	}
}

// WithAddress sets the network address for the server to listen on.
// If not called, the server will use the global configuration address.
func (b *ServerBuilder) WithAddress(addr string) *ServerBuilder {
	b.addr = addr
	return b
}

// WithRootFS sets the root filesystem path for containerized deployments.
// This is typically "/mnt/host" for containers or "/" for bare metal.
func (b *ServerBuilder) WithRootFS(rootFS string) *ServerBuilder {
	b.rootFS = rootFS
	return b
}

// WithTLS enables TLS with the specified certificate and key files.
// This overrides global TLS configuration from command-line flags.
func (b *ServerBuilder) WithTLS(certFile, keyFile string) *ServerBuilder {
	b.tlsConfig = &tlsConfig{
		enabled:  true,
		certFile: certFile,
		keyFile:  keyFile,
	}
	return b
}

// WithMTLS enables mutual TLS with the specified certificate, key, and CA certificate files.
// This overrides global TLS configuration from command-line flags.
func (b *ServerBuilder) WithMTLS(certFile, keyFile, caCertFile string) *ServerBuilder {
	b.tlsConfig = &tlsConfig{
		enabled:     true,
		mtlsEnabled: true,
		certFile:    certFile,
		keyFile:     keyFile,
		caCertFile:  caCertFile,
	}
	return b
}

// WithoutTLS disables TLS for the server.
// This overrides global TLS configuration from command-line flags.
func (b *ServerBuilder) WithoutTLS() *ServerBuilder {
	b.tlsConfig = &tlsConfig{
		enabled: false,
	}
	return b
}

// EnableGNOISystem enables the gNOI System service, which provides system-level
// operations including package management, reboot, and system information.
func (b *ServerBuilder) EnableGNOISystem() *ServerBuilder {
	b.services["gnoi.system"] = true
	return b
}

// EnableGNMI enables the gNMI service, which provides network management
// interface for configuration and operational state retrieval.
func (b *ServerBuilder) EnableGNMI() *ServerBuilder {
	b.services["gnmi"] = true
	return b
}

// EnableGNOIFile enables the gNOI File service, which provides file management
// operations including firmware file listing and file transfer capabilities.
func (b *ServerBuilder) EnableGNOIFile() *ServerBuilder {
	b.services["gnoi.file"] = true
	return b
}

// EnableServices enables multiple services at once based on a slice of service names.
// Valid service names include: "gnoi.system", "gnoi.file", "gnoi.containerz", "gnmi".
func (b *ServerBuilder) EnableServices(services []string) *ServerBuilder {
	for _, service := range services {
		b.services[service] = true
	}
	return b
}

// Build creates and configures the gRPC server with the specified services.
// It registers only the services that have been explicitly enabled through
// the builder methods. Returns an error if server creation fails.
func (b *ServerBuilder) Build() (*Server, error) {
	// Use provided address or fall back to global config
	addr := b.addr
	if addr == "" {
		addr = config.Global.Addr
	}

	// Use provided rootFS or fall back to global config
	rootFS := b.rootFS
	if rootFS == "" {
		rootFS = config.Global.RootFS
	}

	// Determine TLS configuration - use builder config if provided, otherwise global config
	var tlsEnabled, mtlsEnabled bool
	var certFile, keyFile, caCertFile string

	if b.tlsConfig != nil {
		// Use builder-specific TLS configuration
		tlsEnabled = b.tlsConfig.enabled
		mtlsEnabled = b.tlsConfig.mtlsEnabled
		certFile = b.tlsConfig.certFile
		keyFile = b.tlsConfig.keyFile
		caCertFile = b.tlsConfig.caCertFile
	} else {
		// Fall back to global configuration
		tlsEnabled = config.Global.TLSEnabled
		mtlsEnabled = config.Global.MTLSEnabled
		certFile = config.Global.TLSCertFile
		keyFile = config.Global.TLSKeyFile
		caCertFile = config.Global.TLSCACertFile
	}

	// Create the base gRPC server
	srv, err := NewServerWithTLS(
		addr,
		tlsEnabled,
		certFile,
		keyFile,
		mtlsEnabled,
		caCertFile,
	)
	if err != nil {
		return nil, err
	}

	// Register enabled services
	b.registerServices(srv, rootFS)

	return srv, nil
}

// registerServices registers all enabled services with the gRPC server.
// This method handles the service-specific registration logic and logging.
func (b *ServerBuilder) registerServices(srv *Server, rootFS string) {
	serviceCount := 0

	// Register gNOI System service
	if b.services["gnoi.system"] {
		systemServer := gnoiSystem.NewServer(rootFS)
		system.RegisterSystemServer(srv.grpcServer, systemServer)
		glog.Info("Registered gNOI System service")
		serviceCount++
	}

	// Register gNMI service if enabled
	if b.services["gnmi"] {
		gnmiSrv := gnmiserver.NewServer(rootFS)
		gnmi.RegisterGNMIServer(srv.grpcServer, gnmiSrv)
		glog.Info("Registered gNMI service")
		serviceCount++
	}

	// Register gNOI File service if enabled
	if b.services["gnoi.file"] {
		fileServer := gnoiFile.NewServer(rootFS)
		file.RegisterFileServer(srv.grpcServer, fileServer)
		glog.Info("Registered gNOI File service")
		serviceCount++
	}

	// Future services will be implemented:
	// - gNOI Containerz service

	if serviceCount == 0 {
		glog.Info("Server created with gRPC reflection only - no services enabled")
	} else {
		glog.Infof("Registered %d services", serviceCount)
	}
}
