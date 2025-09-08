// Package auth provides simple authorization for gRPC services.
package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/sonic-net/sonic-gnmi/sonic-gnmi-standalone/pkg/cert"
)

// Authorizer defines the interface for checking access permissions.
type Authorizer interface {
	CheckAccess(ctx context.Context, service string, writeAccess bool) error
}

// CertAuthorizer implements certificate-based authorization using SONiC ConfigDB.
type CertAuthorizer struct {
	clientAuth *cert.ClientAuthManager
}

// NewCertAuthorizer creates a new certificate-based authorizer.
func NewCertAuthorizer(clientAuth *cert.ClientAuthManager) *CertAuthorizer {
	return &CertAuthorizer{
		clientAuth: clientAuth,
	}
}

// CheckAccess verifies if the client has the required permissions.
func (a *CertAuthorizer) CheckAccess(ctx context.Context, service string, writeAccess bool) error {
	// Get CN from TLS context
	cn, err := a.extractCN(ctx)
	if err != nil {
		glog.V(2).Infof("Failed to extract CN: %v", err)
		return status.Error(codes.Unauthenticated, "no valid certificate")
	}

	// Get roles from CONFIG_DB
	roles, err := a.getRoles(cn)
	if err != nil {
		glog.V(2).Infof("Failed to get roles for CN %s: %v", cn, err)
		return status.Error(codes.Unauthenticated, "unauthorized certificate")
	}

	// Check if any role grants access
	if a.hasAccess(roles, service, writeAccess) {
		glog.V(2).Infof("Access granted to CN %s for service %s (write=%t)", cn, service, writeAccess)
		return nil
	}

	glog.V(1).Infof("Access denied to CN %s for service %s (write=%t)", cn, service, writeAccess)
	return status.Error(codes.PermissionDenied, "insufficient permissions")
}

// extractCN extracts the Common Name from the client certificate.
func (a *CertAuthorizer) extractCN(ctx context.Context) (string, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("no peer context")
	}

	tlsInfo, ok := peer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", fmt.Errorf("no TLS info")
	}

	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return "", fmt.Errorf("no verified certificate chain")
	}

	cert := tlsInfo.State.VerifiedChains[0][0]
	cn := cert.Subject.CommonName
	if cn == "" {
		return "", fmt.Errorf("certificate has no common name")
	}

	return cn, nil
}

// getRoles retrieves roles for a given common name from the client auth manager.
func (a *CertAuthorizer) getRoles(cn string) ([]string, error) {
	// Use the existing client auth manager to get roles
	// We need to modify the ClientAuthManager to expose roles
	roles := a.clientAuth.GetRolesForCN(cn)
	if len(roles) == 0 {
		return nil, fmt.Errorf("no roles found for CN: %s", cn)
	}

	return roles, nil
}

// hasAccess checks if any of the roles grants access to the service.
func (a *CertAuthorizer) hasAccess(roles []string, service string, writeAccess bool) bool {
	for _, role := range roles {
		glog.V(3).Infof("Checking role %s for service %s", role, service)

		if strings.HasPrefix(role, service) {
			// Extract the access level (e.g., "gnoi_readwrite" -> "readwrite")
			suffix := strings.TrimPrefix(role, service+"_")

			switch suffix {
			case "readwrite":
				glog.V(3).Infof("Role %s grants readwrite access", role)
				return true
			case "readonly":
				if !writeAccess {
					glog.V(3).Infof("Role %s grants readonly access", role)
					return true
				}
				glog.V(3).Infof("Role %s denies write access", role)
			case "noaccess":
				glog.V(3).Infof("Role %s explicitly denies access", role)
				return false
			}
		}
	}

	glog.V(3).Infof("No matching roles found for service %s", service)
	return false
}
