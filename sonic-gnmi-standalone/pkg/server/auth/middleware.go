package auth

import (
	"context"
	"strings"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

// AuthMiddleware creates a gRPC unary interceptor for authorization.
func AuthMiddleware(auth Authorizer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		service, writeAccess := parseMethod(info.FullMethod)

		// Skip authorization for system services (empty service name)
		if service == "" {
			glog.V(3).Infof("Skipping authorization for method %s", info.FullMethod)
			return handler(ctx, req)
		}

		glog.V(2).Infof("Authorization check for method %s -> service=%s, writeAccess=%t",
			info.FullMethod, service, writeAccess)

		if err := auth.CheckAccess(ctx, service, writeAccess); err != nil {
			glog.V(1).Infof("Authorization denied for method %s: %v", info.FullMethod, err)
			return nil, err
		}

		glog.V(2).Infof("Authorization granted for method %s", info.FullMethod)
		return handler(ctx, req)
	}
}

// StreamAuthMiddleware creates a gRPC stream interceptor for authorization.
func StreamAuthMiddleware(auth Authorizer) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
		handler grpc.StreamHandler) error {
		service, writeAccess := parseMethod(info.FullMethod)

		// Skip authorization for system services (empty service name)
		if service == "" {
			glog.V(3).Infof("Skipping stream authorization for method %s", info.FullMethod)
			return handler(srv, ss)
		}

		glog.V(2).Infof("Stream authorization check for method %s -> service=%s, writeAccess=%t",
			info.FullMethod, service, writeAccess)

		if err := auth.CheckAccess(ss.Context(), service, writeAccess); err != nil {
			glog.V(1).Infof("Stream authorization denied for method %s: %v", info.FullMethod, err)
			return err
		}

		glog.V(2).Infof("Stream authorization granted for method %s", info.FullMethod)
		return handler(srv, ss)
	}
}

// gnoiMethods maps gNOI service.method to write access requirement.
// Key format: "service.Method" (e.g., "system.SetPackage", "file.Remove").
var gnoiMethods = map[string]bool{
	// System service methods
	"system.SetPackage":             true,  // write
	"system.Reboot":                 true,  // write
	"system.KillProcess":            true,  // write
	"system.SwitchControlProcessor": true,  // write
	"system.CancelReboot":           true,  // write
	"system.Ping":                   false, // read
	"system.Traceroute":             false, // read
	"system.Time":                   false, // read
	"system.RebootStatus":           false, // read
}

// gnmiMethods maps gNMI methods to write access requirement.
// Since gNMI has only one service, we use just the method name.
var gnmiMethods = map[string]bool{
	"Set":       true,  // write
	"Get":       false, // read
	"Subscribe": false, // read
}

// parseMethod extracts service name and write access requirement from gRPC method name.
func parseMethod(fullMethod string) (service string, writeAccess bool) {
	glog.V(3).Infof("Parsing method: %s", fullMethod)

	// Skip authorization for gRPC reflection and health check services
	if strings.Contains(fullMethod, "grpc.reflection") ||
		strings.Contains(fullMethod, "grpc.health") {
		glog.V(3).Infof("Skipping authorization for system service: %s", fullMethod)
		return "", false // Empty service means skip authorization
	}

	// Extract service and method from fullMethod
	// Format: /package.ServiceName/MethodName
	parts := strings.Split(fullMethod, "/")
	if len(parts) < 3 {
		glog.V(2).Infof("Invalid method format: %s", fullMethod)
		return "unknown", false
	}

	grpcServiceName := parts[1] // e.g., "gnoi.system.System" or "gnmi.gNMI"
	methodName := parts[2]      // e.g., "SetPackage" or "Get"

	// Handle gNOI services (all start with "gnoi.")
	if strings.HasPrefix(grpcServiceName, "gnoi.") {
		// Extract service name from grpcServiceName (e.g., "gnoi.system.System" -> "system")
		serviceParts := strings.Split(grpcServiceName, ".")
		if len(serviceParts) >= 2 {
			serviceName := serviceParts[1]                  // "system", "file", "cert", etc.
			serviceMethod := serviceName + "." + methodName // "system.SetPackage"

			if writeAccess, exists := gnoiMethods[serviceMethod]; exists {
				glog.V(3).Infof("gNOI method %s mapped to write=%t", serviceMethod, writeAccess)
				return "gnoi", writeAccess
			}

			// Unknown gNOI method - default to read access
			glog.V(2).Infof("Unknown gNOI method %s, defaulting to read", serviceMethod)
			return "gnoi", false
		}

		glog.V(2).Infof("Invalid gNOI service name format: %s", grpcServiceName)
		return "gnoi", false
	}

	// Handle gNMI service
	if grpcServiceName == "gnmi.gNMI" {
		if writeAccess, exists := gnmiMethods[methodName]; exists {
			glog.V(3).Infof("gNMI method %s mapped to write=%t", methodName, writeAccess)
			return "gnmi", writeAccess
		}

		// Unknown gNMI method - default to read access
		glog.V(2).Infof("Unknown gNMI method %s, defaulting to read", methodName)
		return "gnmi", false
	}

	// Unknown service
	glog.V(2).Infof("Unknown service for method %s", fullMethod)
	return "unknown", false
}
