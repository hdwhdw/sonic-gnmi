// Package auth provides a simple test example showing authorization in action.
package auth

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/openconfig/gnoi/system"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sonic-net/sonic-gnmi/sonic-gnmi-standalone/pkg/cert"
	gnoiSystem "github.com/sonic-net/sonic-gnmi/sonic-gnmi-standalone/pkg/server/gnoi/system"
)

// DemoAuthorizationExample demonstrates the authorization system.
func DemoAuthorizationExample() {
	fmt.Println("=== Simple Authorization Demo ===")

	// 1. Create a mock client auth manager
	clientAuth := cert.NewClientAuthManager("localhost:6379", 4, "GNMI_CLIENT_CERT")

	// 2. Add some test client certificates with different roles
	clientAuth.AddClientCN("admin.client", "gnoi_readwrite,gnmi_readwrite")  // Full access
	clientAuth.AddClientCN("readonly.client", "gnoi_readonly,gnmi_readonly") // Read-only
	clientAuth.AddClientCN("noaccess.client", "gnoi_noaccess")               // Explicitly denied
	clientAuth.AddClientCN("gnmi-only.client", "gnmi_readwrite")             // Only gNMI access

	// 3. Create authorizer
	authorizer := NewCertAuthorizer(clientAuth)

	// 4. Create a test server with authorization
	grpcServer, err := NewServerWithAuth(":0", nil, authorizer) // Insecure for demo
	if err != nil {
		fmt.Printf("Failed to create server: %v\n", err)
		return
	}

	// 5. Register the gNOI System service
	systemService := gnoiSystem.NewServer("/tmp")
	system.RegisterSystemServer(grpcServer, systemService)

	// 6. Start server
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			glog.Errorf("Server error: %v", err)
		}
	}()

	fmt.Printf("Server started on %s\n", lis.Addr().String())

	// 7. Test different scenarios
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()

	_ = system.NewSystemClient(conn)

	fmt.Println("\n=== Testing Authorization Scenarios ===")

	// Test Case 1: Try to call Time (read operation) - should work with any client
	fmt.Println("\n1. Testing Time RPC (read operation):")
	fmt.Println("   - This would be authorized for: admin.client, readonly.client, gnmi-only.client")
	fmt.Println("   - This would be denied for: noaccess.client")

	// Test Case 2: Try to call SetPackage (write operation) - should only work with readwrite
	fmt.Println("\n2. Testing SetPackage RPC (write operation):")
	fmt.Println("   - This would be authorized for: admin.client")
	fmt.Println("   - This would be denied for: readonly.client, noaccess.client, gnmi-only.client")

	fmt.Println("\n=== Role Authorization Logic ===")
	testRoleLogic(authorizer)

	grpcServer.Stop()
	fmt.Println("\n=== Demo Complete ===")
}

// testRoleLogic demonstrates the authorization logic without network calls.
func testRoleLogic(authorizer Authorizer) {
	// Create mock contexts with different client certificates
	// Note: In real scenario, these would come from actual TLS connections

	testCases := []struct {
		clientCN    string
		service     string
		writeAccess bool
		expectPass  bool
	}{
		{"admin.client", "gnoi", false, true},      // admin can read gNOI
		{"admin.client", "gnoi", true, true},       // admin can write gNOI
		{"readonly.client", "gnoi", false, true},   // readonly can read gNOI
		{"readonly.client", "gnoi", true, false},   // readonly cannot write gNOI
		{"noaccess.client", "gnoi", false, false},  // noaccess denied gNOI read
		{"noaccess.client", "gnoi", true, false},   // noaccess denied gNOI write
		{"gnmi-only.client", "gnoi", false, false}, // gnmi-only has no gNOI access
		{"gnmi-only.client", "gnmi", true, true},   // gnmi-only can write gNMI
	}

	for i, tc := range testCases {
		// Note: This is a simplified test - in real usage, the CN would come from TLS context
		fmt.Printf("Test %d: CN=%s, service=%s, write=%t -> Expected: %t\n",
			i+1, tc.clientCN, tc.service, tc.writeAccess, tc.expectPass)

		// In a real implementation, you would:
		// 1. Create a context with proper TLS peer info
		// 2. Call authorizer.CheckAccess(ctx, tc.service, tc.writeAccess)
		// 3. Check if err == nil matches tc.expectPass
	}
}

// Example of how to use the authorization in main.go.
func ExampleUsage() {
	// 1. Set up client auth manager (connects to Redis ConfigDB)
	clientAuth := cert.NewClientAuthManager("localhost:6379", 4, "GNMI_CLIENT_CERT")
	if err := clientAuth.LoadClientCertConfig(); err != nil {
		glog.Fatalf("Failed to load client config: %v", err)
	}

	// 2. Create authorizer
	authorizer := NewCertAuthorizer(clientAuth)

	// 3. Create server with authorization
	server, err := NewServerWithAuth(":50055", nil, authorizer)
	if err != nil {
		glog.Fatalf("Failed to create server: %v", err)
	}

	// 4. Register services - they automatically get authorization
	systemService := gnoiSystem.NewServer("/mnt/host")
	system.RegisterSystemServer(server, systemService)

	// 5. Start server
	lis, _ := net.Listen("tcp", ":50055")
	server.Serve(lis) // Authorization happens automatically via middleware
}
