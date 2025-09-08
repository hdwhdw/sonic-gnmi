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

// parseMethod extracts service name and write access requirement from gRPC method name.
func parseMethod(fullMethod string) (service string, writeAccess bool) {
	glog.V(3).Infof("Parsing method: %s", fullMethod)

	// Skip authorization for gRPC reflection and health check services
	if strings.Contains(fullMethod, "grpc.reflection") ||
		strings.Contains(fullMethod, "grpc.health") {
		glog.V(3).Infof("Skipping authorization for system service: %s", fullMethod)
		return "", false // Empty service means skip authorization
	}

	switch {
	case strings.Contains(fullMethod, "/gnoi.system.System/"):
		// Extract the method name from "/gnoi.system.System/MethodName"
		parts := strings.Split(fullMethod, "/")
		if len(parts) >= 3 {
			methodName := parts[len(parts)-1]

			// Define which gNOI System methods require write access
			writeOps := []string{
				"Reboot", "KillProcess", "SetPackage",
				"SwitchControlProcessor", "CancelReboot",
			}

			for _, op := range writeOps {
				if methodName == op {
					glog.V(3).Infof("Method %s requires write access", methodName)
					return "gnoi", true
				}
			}

			glog.V(3).Infof("Method %s requires read access", methodName)
			return "gnoi", false
		}
		return "gnoi", false

	case strings.Contains(fullMethod, "/gnmi.gNMI/"):
		// Extract method name for gNMI service
		parts := strings.Split(fullMethod, "/")
		if len(parts) >= 3 {
			methodName := parts[len(parts)-1]

			// Only "Set" requires write access in gNMI
			if methodName == "Set" {
				glog.V(3).Infof("gNMI Set method requires write access")
				return "gnmi", true
			}

			glog.V(3).Infof("gNMI %s method requires read access", methodName)
			return "gnmi", false
		}
		return "gnmi", false

	default:
		glog.V(2).Infof("Unknown service for method %s, defaulting to unknown/read", fullMethod)
		return "unknown", false
	}
}
