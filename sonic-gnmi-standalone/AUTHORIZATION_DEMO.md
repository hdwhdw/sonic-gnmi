# Simple Authorization Implementation for SONiC gNMI Standalone

This document demonstrates the new simple authorization system implemented for the sonic-gnmi-standalone server.

## Overview

The authorization system provides clean separation between authentication and business logic using:

1. **Simple Interface**: `Authorizer.CheckAccess(ctx, service, writeAccess)`
2. **Middleware Pattern**: gRPC interceptors handle authorization automatically  
3. **Clean Handlers**: No authorization code needed in RPC implementations
4. **Certificate-Based**: Integrates with SONiC ConfigDB for certificate-to-role mapping

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   RPC Request   │───▶│   Middleware     │───▶│  RPC Handler    │
│  (with cert)    │    │  - Extract CN    │    │ (business logic │
│                 │    │  - Check roles   │    │    only)        │
│                 │    │  - Allow/Deny    │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Components

### 1. Authorizer Interface (`pkg/server/auth/authorizer.go`)
```go
type Authorizer interface {
    CheckAccess(ctx context.Context, service string, writeAccess bool) error
}
```

### 2. Certificate Authorizer Implementation
- Extracts CN from TLS client certificate
- Looks up roles from SONiC ConfigDB (`GNMI_CLIENT_CERT` table)
- Checks if roles grant access to requested service

### 3. Authorization Middleware (`pkg/server/auth/middleware.go`)
- gRPC Unary and Stream interceptors
- Automatic service detection from method name
- Write/read access determination

### 4. Integration with Server Builder
- Automatically creates authorization when mTLS is enabled
- Seamlessly integrates with existing certificate management

## Role-Based Access Control

### Role Format
Roles follow the pattern: `{service}_{access_level}`

**Supported Services:**
- `gnoi` - gNOI System operations
- `gnmi` - gNMI operations (future)

**Access Levels:**
- `readwrite` - Full read/write access
- `readonly` - Read-only access  
- `noaccess` - Explicitly denied access

### Example Roles

**ConfigDB Setup:**
```redis
# Full admin access
HSET GNMI_CLIENT_CERT|admin.client.sonic role@ "gnoi_readwrite,gnmi_readwrite"

# Read-only monitoring
HSET GNMI_CLIENT_CERT|monitor.client role@ "gnoi_readonly,gnmi_readonly"

# gNOI operations only
HSET GNMI_CLIENT_CERT|system.client role@ "gnoi_readwrite"

# Explicitly denied
HSET GNMI_CLIENT_CERT|blocked.client role@ "gnoi_noaccess"
```

## SetPackage RPC Example

The `gnoi.system.SetPackage` RPC demonstrates the authorization system:

### Before (No Authorization)
```go
func (s *Server) SetPackage(stream system.System_SetPackageServer) error {
    // Direct business logic - no auth checks
    glog.Info("SetPackage RPC called")
    // ... package installation logic
}
```

### After (With Authorization)
```go
func (s *Server) SetPackage(stream system.System_SetPackageServer) error {
    // Authorization handled automatically by middleware
    // - Extracts CN from client certificate
    // - Checks if CN has "gnoi_readwrite" or "gnoi_readonly" roles
    // - Denies access if only "gnoi_readonly" (SetPackage is write operation)
    // - Allows access if "gnoi_readwrite"
    
    glog.Info("SetPackage RPC called") // Only executes if authorized
    // ... package installation logic (unchanged)
}
```

## Authorization Flow

### 1. Client Request
Client makes gRPC call with client certificate:
```bash
grpcurl -cert client.crt -key client.key -cacert ca.crt \
  -plaintext localhost:50055 \
  gnoi.system.System/SetPackage
```

### 2. Middleware Intercepts
```go
// middleware.go - parseMethod()
"/gnoi.system.System/SetPackage" -> service="gnoi", writeAccess=true
```

### 3. Authorization Check
```go
// authorizer.go - CheckAccess()
1. Extract CN from certificate: "admin.client.sonic"
2. Get roles from ConfigDB: ["gnoi_readwrite", "gnmi_readonly"]  
3. Check role "gnoi_readwrite" vs service "gnoi" + write=true
4. Result: ALLOW (readwrite grants write access)
```

### 4. Business Logic Execution
If authorized, the actual SetPackage implementation runs without any auth code.

## Method-to-Service Mapping

The middleware automatically maps gRPC methods to services:

### gNOI System Methods
- **Write Operations** (require `gnoi_readwrite`):
  - `Reboot`, `KillProcess`, `SetPackage`, `SwitchControlProcessor`, `CancelReboot`
  
- **Read Operations** (allow `gnoi_readonly`):
  - `Time`, `Ping`, `Traceroute`, other status methods

### Future gNMI Methods  
- **Write Operations**: `Set`
- **Read Operations**: `Get`, `Subscribe`, `Capabilities`

## Testing

### Build and Run
```bash
# Build the server
go build ./cmd/server

# Run with mTLS and authorization
./server --mtls --tls-ca-cert=ca.crt --redis-addr=localhost:6379
```

### Setup ConfigDB
```bash
# Add authorized client
redis-cli -n 4 HSET GNMI_CLIENT_CERT|test.client.sonic role@ "gnoi_readwrite"

# Test authorization
grpcurl -cert client.crt -key client.key -cacert ca.crt \
  localhost:50055 gnoi.system.System/Time  # Should work (read)

grpcurl -cert client.crt -key client.key -cacert ca.crt \
  localhost:50055 gnoi.system.System/SetPackage  # Should work (write)
```

## Benefits

1. **Clean Separation**: Authorization logic separate from business logic
2. **Zero Boilerplate**: RPC handlers have no auth code
3. **Consistent**: Same auth logic applied to all RPCs
4. **Testable**: Easy to mock `Authorizer` interface
5. **Maintainable**: Change auth logic in one place
6. **SONiC Compatible**: Uses same role format as main sonic-gnmi server

## Comparison with Main sonic-gnmi

| Feature | Main sonic-gnmi | Standalone (New) |
|---------|----------------|------------------|
| **Auth Logic** | Mixed in each RPC handler | Centralized middleware |
| **Code per RPC** | ~10 lines auth code | 0 lines auth code |
| **Testing** | Hard to mock auth | Easy to mock interface |
| **Consistency** | Potential inconsistencies | Guaranteed consistency |
| **Maintenance** | Scattered updates | Single point of change |

This simple authorization system provides the same security guarantees as the main sonic-gnmi server while being much cleaner and more maintainable.