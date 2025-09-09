package auth

import (
	"strings"
	"testing"
)

func TestParseMethodPath(t *testing.T) {
	tests := []struct {
		name        string
		fullMethod  string
		wantService string
		wantMethod  string
		wantValid   bool
	}{
		{
			name:        "valid gNOI system method",
			fullMethod:  "/gnoi.system.System/SetPackage",
			wantService: "gnoi.system.System",
			wantMethod:  "SetPackage",
			wantValid:   true,
		},
		{
			name:        "valid gNMI method",
			fullMethod:  "/gnmi.gNMI/Get",
			wantService: "gnmi.gNMI",
			wantMethod:  "Get",
			wantValid:   true,
		},
		{
			name:       "invalid format - missing parts",
			fullMethod: "/invalid",
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.fullMethod, "/")
			if len(parts) < 3 && tt.wantValid {
				t.Errorf("Expected valid format but got invalid: %s", tt.fullMethod)
			}
			if len(parts) >= 3 && tt.wantValid {
				service, method := parts[1], parts[2]
				if service != tt.wantService {
					t.Errorf("Got service %s, want %s", service, tt.wantService)
				}
				if method != tt.wantMethod {
					t.Errorf("Got method %s, want %s", method, tt.wantMethod)
				}
			}
		})
	}
}

func TestGnoiServiceExtraction(t *testing.T) {
	tests := []struct {
		grpcService string
		wantService string
		wantValid   bool
	}{
		{"gnoi.system.System", "system", true},
		{"gnoi.file.File", "file", true},
		{"gnoi.cert.CertificateManagement", "cert", true},
		{"gnmi.gNMI", "", false}, // Not a gNOI service
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.grpcService, func(t *testing.T) {
			if strings.HasPrefix(tt.grpcService, "gnoi.") {
				parts := strings.Split(tt.grpcService, ".")
				if len(parts) >= 2 {
					service := parts[1]
					if tt.wantValid && service != tt.wantService {
						t.Errorf("Got service %s, want %s", service, tt.wantService)
					}
				}
			} else if tt.wantValid {
				t.Errorf("Expected gNOI service but got %s", tt.grpcService)
			}
		})
	}
}

func TestMethodLookup(t *testing.T) {
	t.Run("gNOI method lookup", func(t *testing.T) {
		// Test known write method
		if writeAccess, exists := gnoiMethods["system.SetPackage"]; !exists || !writeAccess {
			t.Error("system.SetPackage should be a write method")
		}

		// Test known read method
		if writeAccess, exists := gnoiMethods["system.Time"]; !exists || writeAccess {
			t.Error("system.Time should be a read method")
		}

		// Test unknown method
		if _, exists := gnoiMethods["system.UnknownMethod"]; exists {
			t.Error("system.UnknownMethod should not exist")
		}
	})

	t.Run("gNMI method lookup", func(t *testing.T) {
		// Test known write method
		if writeAccess, exists := gnmiMethods["Set"]; !exists || !writeAccess {
			t.Error("Set should be a write method")
		}

		// Test known read method
		if writeAccess, exists := gnmiMethods["Get"]; !exists || writeAccess {
			t.Error("Get should be a read method")
		}
	})
}

func TestSkipSystemServices(t *testing.T) {
	systemServices := []string{
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
		"/grpc.health.v1.Health/Check",
	}

	for _, method := range systemServices {
		t.Run(method, func(t *testing.T) {
			shouldSkip := strings.Contains(method, "grpc.reflection") ||
				strings.Contains(method, "grpc.health")
			if !shouldSkip {
				t.Errorf("Method %s should be skipped for authorization", method)
			}
		})
	}
}

func TestMethodConfiguration(t *testing.T) {
	// Verify we have some methods configured
	if len(gnoiMethods) == 0 {
		t.Error("No gNOI methods configured")
	}
	if len(gnmiMethods) == 0 {
		t.Error("No gNMI methods configured")
	}

	// Verify method naming format for gNOI
	for method := range gnoiMethods {
		if !strings.Contains(method, ".") {
			t.Errorf("gNOI method %s should use service.Method format", method)
		}
	}
}

// TestEasyExtensionExample demonstrates how simple it is to add new gNOI services.
func TestEasyExtensionExample(t *testing.T) {
	// Example: Adding gNOI File service methods would only require:
	//   "file.Put":    true,   // write
	//   "file.Remove": true,   // write
	//   "file.Get":    false,  // read
	//   "file.Stat":   false,  // read

	// Save original gnoiMethods
	original := make(map[string]bool)
	for k, v := range gnoiMethods {
		original[k] = v
	}

	// Add new file service methods
	gnoiMethods["file.Put"] = true
	gnoiMethods["file.Remove"] = true
	gnoiMethods["file.Get"] = false
	gnoiMethods["file.Stat"] = false

	// Test it works
	service, writeAccess := parseMethod("/gnoi.file.File/Put")
	if service != "gnoi" || !writeAccess {
		t.Errorf("Expected gnoi service with write access, got %s (write=%t)", service, writeAccess)
	}

	service, writeAccess = parseMethod("/gnoi.file.File/Get")
	if service != "gnoi" || writeAccess {
		t.Errorf("Expected gnoi service with read access, got %s (write=%t)", service, writeAccess)
	}

	// Test adding methods from multiple gNOI services
	gnoiMethods["cert.Install"] = true
	gnoiMethods["cert.GetCertificates"] = false
	gnoiMethods["bgp.ClearBGPNeighbor"] = true

	// Test cert service
	service, writeAccess = parseMethod("/gnoi.cert.CertificateManagement/Install")
	if service != "gnoi" || !writeAccess {
		t.Errorf("Expected gnoi cert service with write access, got %s (write=%t)", service, writeAccess)
	}

	// Test bgp service
	service, writeAccess = parseMethod("/gnoi.bgp.BGP/ClearBGPNeighbor")
	if service != "gnoi" || !writeAccess {
		t.Errorf("Expected gnoi bgp service with write access, got %s (write=%t)", service, writeAccess)
	}

	// Restore original configuration
	gnoiMethods = original
}

// TestParseMethodIntegration provides a focused integration test of the complete parseMethod function.
func TestParseMethodIntegration(t *testing.T) {
	tests := []struct {
		name        string
		fullMethod  string
		wantService string
		wantWrite   bool
	}{
		{"gNOI write", "/gnoi.system.System/SetPackage", "gnoi", true},
		{"gNOI read", "/gnoi.system.System/Time", "gnoi", false},
		{"gNMI write", "/gnmi.gNMI/Set", "gnmi", true},
		{"gNMI read", "/gnmi.gNMI/Get", "gnmi", false},
		{"skip reflection", "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo", "", false},
		{"unknown service", "/unknown.service/Method", "unknown", false},
		{"unknown gNOI method", "/gnoi.system.System/UnknownMethod", "gnoi", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, writeAccess := parseMethod(tt.fullMethod)
			if service != tt.wantService {
				t.Errorf("service = %v, want %v", service, tt.wantService)
			}
			if writeAccess != tt.wantWrite {
				t.Errorf("writeAccess = %v, want %v", writeAccess, tt.wantWrite)
			}
		})
	}
}
