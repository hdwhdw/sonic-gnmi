# Manual Testing Guide

This guide explains how to set up and test the SONiC upgrade service manually.

## Prerequisites

- Docker installed
- Go 1.21+ (for building binaries)
- Access to a SONiC device or local Linux host

## Server Setup

You can run the upgrade service server in two ways:

### Option 1: Local Linux Host (Recommended for Testing)

For testing without SONiC dependencies, you can run the server on any Linux host:

```bash
# Build and deploy the server locally
./docker/build_deploy.sh -t

# This will:
# 1. Build the Docker image with latest code
# 2. Stop any existing container
# 3. Start a new container with port 50051 exposed
# 4. Mount /tmp as /host for file operations
```

The server will be available at `localhost:50051` and ready for client connections.

### Option 2: Remote SONiC Host (Production-like Testing)

For testing on actual SONiC hardware or VMs:

```bash
# Example with SONiC KVM switch
ssh admin@vlab-01

# Transfer and run the deployment script
./docker/build_deploy.sh -t

# The server will be available at vlab-01:50051
```

## Verify Server is Running

Check that the server is running and accessible:

```bash
# Check container status
docker ps | grep opsd

# Check server logs
docker logs opsd

# Test connectivity (should see gRPC error, which is expected)
curl -v localhost:50051
```

## Client Setup

Build the client binary:

```bash
# Build the client
make build

# The client binary will be available at:
# ./bin/upgrade-agent
```

## Basic Client Testing

### 1. Test Server Connectivity

```bash
# Test basic server connectivity with disk space check
./bin/upgrade-agent disk-space --server localhost:50051 --no-tls

# Expected output: filesystem usage information
```

### 2. Test Download Functionality

```bash
# Test with a small HTTP file for quick validation
./bin/upgrade-agent download \
  --url http://httpbin.org/bytes/51200 \
  --output-path /tmp/test-download.bin \
  --server localhost:50051 \
  --no-tls

# Expected output: Progress bar showing download progress
# [===========================] 100% 50.0 KB/50.0 KB @ X.X KB/s
```

### 3. Test Configuration-based Download

Create a test configuration file:

```bash
cat > /tmp/test-config.yaml << 'EOF'
apiVersion: upgrade/v1
kind: UpgradeConfig
metadata:
  name: test-download
spec:
  firmware:
    desiredVersion: "test-version"
    downloadUrl: "http://httpbin.org/bytes/102400"
    savePath: "/tmp/sonic-images/"
  download:
    connectTimeoutSeconds: 30
    totalTimeoutSeconds: 300
  server:
    address: "localhost:50051"
    tlsEnabled: false
EOF

# Test with configuration file
./bin/upgrade-agent apply \
  --config /tmp/test-config.yaml \
  --server localhost:50051 \
  --no-tls
```

## Advanced Testing

### Testing Different Scenarios

> **Note**: httpbin.org/bytes has a ~100KB size limit, so we use smaller files for testing.

1. **Large File Download** (test progress tracking):
   ```bash
   ./bin/upgrade-agent download \
     --url http://httpbin.org/bytes/102400 \
     --output-path /tmp/large-test.bin \
     --server localhost:50051 \
     --no-tls
   ```

2. **Error Handling** (test 404 response):
   ```bash
   ./bin/upgrade-agent download \
     --url http://httpbin.org/status/404 \
     --output-path /tmp/error-test.bin \
     --server localhost:50051 \
     --no-tls
   ```

3. **Timeout Testing** (test connection timeout):
   ```bash
   ./bin/upgrade-agent download \
     --url http://httpbin.org/delay/10 \
     --output-path /tmp/timeout-test.bin \
     --connect-timeout 5 \
     --server localhost:50051 \
     --no-tls
   ```

### Monitor Download Progress

In a separate terminal, you can monitor download status:

```bash
# Get the session ID from the download command output, then:
./bin/upgrade-agent status \
  --session-id download-1234567890 \
  --server localhost:50051 \
  --no-tls
```

## Troubleshooting

### Server Not Starting

```bash
# Check container logs
docker logs opsd

# Check if port is in use
netstat -ln | grep 50051

# Restart the server
docker restart opsd
```

### Client Connection Issues

```bash
# Verify server address and port
./bin/upgrade-agent disk-space --server localhost:50051 --no-tls --verbose

# Check network connectivity
telnet localhost 50051
```

### Download Issues

```bash
# Test with verbose logging
./bin/upgrade-agent download \
  --url http://httpbin.org/get \
  --output-path /tmp/debug.bin \
  --server localhost:50051 \
  --no-tls \
  --verbose
```

## Expected Behavior

### Successful Download
- Progress bar should update smoothly during download
- Shows percentage, downloaded/total bytes, and speed
- Completes with success message and file information

### Failed Download
- Clear error message indicating the failure reason
- Proper error categorization (network, HTTP, filesystem, etc.)
- Session remains queryable for error details

## Cleanup

```bash
# Stop the server container
docker stop opsd

# Remove test files
rm -f /tmp/test-*.bin /tmp/test-config.yaml

# Remove downloaded images (if any)
rm -rf /tmp/sonic-images/
```
