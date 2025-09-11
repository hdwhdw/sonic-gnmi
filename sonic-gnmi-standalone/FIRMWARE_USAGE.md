# Firmware File Listing gNMI RPC

This document describes how to use the firmware file listing functionality that integrates with the gNOI File service.

## Overview

The firmware file listing functionality allows you to query firmware files in specific directories using gNMI Get requests. This feature requires the gNOI File service to be enabled and provides:
- Discovering available firmware files
- Getting file metadata (size, modification time, permissions)
- Monitoring firmware directory contents

## Prerequisites

**Important**: The firmware file listing functionality requires the gNOI File service to be enabled using the `--enable-gnoi-file` flag when starting the server.

```bash
# Start server with gNOI File service enabled
./sonic-gnmi-standalone --enable-gnoi-file
```

Without this flag, firmware path requests will return an error.

## Supported Paths

The following gNMI paths are supported:

### List all files in a firmware directory
```
/sonic/system/firmware[directory=<path>]/files
```

### Get count of files in a firmware directory
```
/sonic/system/firmware[directory=<path>]/files/count
```

### Get information about a specific file
```
/sonic/system/firmware[directory=<path>]/files/<filename>
```

## Common Firmware Directories

Typical firmware directories in SONiC/Linux systems include:
- `/lib/firmware` - System firmware files
- `/usr/lib/firmware` - Additional firmware files
- `/opt/firmware` - Custom firmware files
- `/etc/firmware` - Configuration-specific firmware

## Example Usage

### Using gnmic client

**Note**: Make sure the server is started with `--enable-gnoi-file` flag for these examples to work.

1. **List all firmware files in /lib/firmware:**
```bash
# Server must be started with: ./sonic-gnmi-standalone --enable-gnoi-file
gnmic -a <server>:50051 get --path "/sonic/system/firmware[directory=/lib/firmware]/files"
```

2. **Get count of files in /usr/lib/firmware:**
```bash
gnmic -a <server>:50051 get --path "/sonic/system/firmware[directory=/usr/lib/firmware]/files/count"
```

3. **Get information about a specific firmware file:**
```bash
gnmic -a <server>:50051 get --path "/sonic/system/firmware[directory=/lib/firmware]/files/example.bin"
```

### Example Response

When listing all files, you'll get a response like:
```json
{
  "directory": "/lib/firmware",
  "file_count": 3,
  "files": [
    {
      "name": "firmware1.bin",
      "size": 1048576,
      "mod_time": "2024-01-15T10:30:00Z",
      "is_directory": false,
      "permissions": "-rw-r--r--"
    },
    {
      "name": "subdir/firmware2.bin",
      "size": 2097152,
      "mod_time": "2024-01-15T11:30:00Z",
      "is_directory": false,
      "permissions": "-rw-r--r--"
    },
    {
      "name": "subdir",
      "size": 4096,
      "mod_time": "2024-01-15T09:30:00Z",
      "is_directory": true,
      "permissions": "drwxr-xr-x"
    }
  ]
}
```

## Error Handling

The implementation handles several error conditions:

- **gNOI File service disabled**: Returns FailedPrecondition if `--enable-gnoi-file` flag is not used
- **Directory not found**: Returns Internal error if the specified directory doesn't exist
- **Permission denied**: Returns Internal error if the server can't access the directory
- **Invalid path format**: Returns InvalidArgument if the gNMI path is malformed
- **File not found**: Returns Internal error if a specific file is requested but doesn't exist

### Example Error Response

If you try to access firmware paths without enabling the gNOI File service:

```bash
# This will fail if --enable-gnoi-file is not used
gnmic -a localhost:50051 get --path "/sonic/system/firmware[directory=/lib/firmware]/files"

# Error response:
# rpc error: code = FailedPrecondition desc = firmware file listing requires gNOI File service to be enabled (use --enable-gnoi-file flag)
```

## Security Considerations

- The server resolves paths relative to its configured rootFS for security
- Only files within the specified directory are accessible
- Directory traversal attacks are prevented by path resolution logic
- File permissions are respected by the underlying filesystem

## Implementation Details

The firmware file listing is implemented as an integration between gNMI and gNOI File services:

- **gNMI Interface**: Provides the query interface using familiar gNMI paths
- **gNOI File Service**: Handles the actual file operations and directory scanning
- **Conditional Activation**: Only enabled when `--enable-gnoi-file` flag is used
- **Service Integration**: gNMI requests delegate to gNOI File service methods
- **Unified Logging**: Both services use consistent logging patterns

### Architecture

```
gNMI Get Request → Path Detection → EnableGNOIFile Check → gNOI File Service → Response
```

This design ensures that:
- File operations are properly isolated in the gNOI File service
- gNMI provides a familiar query interface
- The functionality is opt-in via configuration flag
- Future gNOI File features can be easily added

## Capabilities

The server advertises the following capabilities for firmware functionality:

```json
{
  "supported_models": [
    {
      "name": "sonic-firmware",
      "organization": "SONiC",
      "version": "1.0.0"
    }
  ],
  "supported_paths": [
    "/sonic/system/firmware[directory=*]/files",
    "/sonic/system/firmware[directory=*]/files/count",
    "/sonic/system/firmware[directory=*]/files/*"
  ]
}
```
