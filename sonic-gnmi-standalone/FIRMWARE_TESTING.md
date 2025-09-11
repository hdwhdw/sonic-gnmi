# Firmware File Listing Test Documentation

This document describes the comprehensive test suite for the firmware file listing gNMI RPC functionality.

## Test Files Created

### 1. **Client Convenience Methods** (`pkg/client/gnmi/firmware.go`)

**Purpose**: Provides easy-to-use client methods for firmware operations.

**Key Methods**:
- `GetFirmwareFiles(ctx, directory)` - Lists all firmware files in a directory
- `GetFirmwareFileCount(ctx, directory)` - Gets count of files in a directory
- `GetFirmwareFileInfo(ctx, directory, filename)` - Gets info about a specific file

**Data Structures**:
```go
type FirmwareFileInfo struct {
    Name        string    `json:"name"`
    Size        int64     `json:"size"`
    ModTime     time.Time `json:"mod_time"`
    IsDirectory bool      `json:"is_directory"`
    Permissions string    `json:"permissions"`
}

type FirmwareFilesResponse struct {
    Directory string             `json:"directory"`
    FileCount int                `json:"file_count"`
    Files     []FirmwareFileInfo `json:"files"`
}
```

### 2. **Comprehensive Test Suite** (`tests/loopback/gnmi_firmware_test.go`)

**Purpose**: End-to-end integration tests for firmware functionality.

**Test Categories**:

#### **A. Basic Functionality Tests**
- ✅ **capabilities_include_firmware**: Verifies firmware model is advertised
- ✅ **list_all_firmware_files**: Tests listing all files with metadata
- ✅ **get_firmware_file_count**: Tests file count retrieval
- ✅ **get_specific_firmware_file_info**: Tests specific file information
- ✅ **get_nested_file_info**: Tests files in subdirectories
- ✅ **get_directory_info**: Tests directory metadata

#### **B. Error Handling Tests**
- ✅ **error_nonexistent_directory**: Tests missing directory handling
- ✅ **error_nonexistent_file**: Tests missing file handling
- ✅ **error_empty_directory_path**: Tests validation of empty directory
- ✅ **error_empty_filename**: Tests validation of empty filename

#### **C. Service Dependency Tests**
- ✅ **firmware_fails_without_gnoi_file_service**: Tests that firmware functionality requires gNOI File service

### 3. **Test Infrastructure Updates**

#### **Updated Test Helpers** (`tests/loopback/test_helpers.go`)
- Added support for `gnoi.file` service in test setup
- Automatically enables `EnableGNOIFile` config when gnoi.file service is requested
- Maintains backward compatibility with existing tests

#### **Updated Existing Tests** (`tests/loopback/gnmi_diskspace_test.go`)
- Updated capabilities test to expect 2 models (sonic-system v1.1.0 + sonic-firmware v1.0.0)
- Maintains compatibility with enhanced server capabilities

## Test Structure

### **Test Setup**
```go
// Enable both gNMI and gNOI File services
testServer := SetupInsecureTestServer(t, tempDir, []string{"gnmi", "gnoi.file"})

// Create test firmware directory structure
firmwareDir := filepath.Join(tempDir, "test-firmware")
// ... create test files and subdirectories
```

### **Sample Test Data**
The tests create a realistic firmware directory structure:
```
test-firmware/
├── device1.bin           (firmware content for device 1)
├── device2.fw            (firmware content for device 2) 
├── bootloader.img        (bootloader firmware image)
└── drivers/
    ├── driver1.bin       (driver firmware 1)
    └── driver2.bin       (driver firmware 2)
```

### **Verification Points**
Each test verifies:
- ✅ **Response Structure**: Proper gNMI response format
- ✅ **Data Accuracy**: File count, names, sizes, and metadata
- ✅ **Recursive Scanning**: Files in subdirectories are found
- ✅ **File vs Directory**: Proper identification of files vs directories
- ✅ **Error Handling**: Appropriate error responses for invalid requests

## Running the Tests

### **Run Firmware Tests Only**
```bash
cd sonic-gnmi-standalone
go test ./tests/loopback -v -run TestGNMIFirmware
```

### **Run All Integration Tests**
```bash
cd sonic-gnmi-standalone
go test ./tests/loopback -v
```

### **Test with Coverage**
```bash
cd sonic-gnmi-standalone
go test ./tests/loopback -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Expected Test Results

### **Successful Test Run**
```
=== RUN   TestGNMIFirmwareLoopback
=== RUN   TestGNMIFirmwareLoopback/capabilities_include_firmware
=== RUN   TestGNMIFirmwareLoopback/list_all_firmware_files
=== RUN   TestGNMIFirmwareLoopback/get_firmware_file_count
=== RUN   TestGNMIFirmwareLoopback/get_specific_firmware_file_info
=== RUN   TestGNMIFirmwareLoopback/get_nested_file_info
=== RUN   TestGNMIFirmwareLoopback/get_directory_info
=== RUN   TestGNMIFirmwareLoopback/error_nonexistent_directory
=== RUN   TestGNMIFirmwareLoopback/error_nonexistent_file
=== RUN   TestGNMIFirmwareLoopback/error_empty_directory_path
=== RUN   TestGNMIFirmwareLoopback/error_empty_filename
--- PASS: TestGNMIFirmwareLoopback (2.34s)
=== RUN   TestGNMIFirmwareWithoutGNOIFile
=== RUN   TestGNMIFirmwareWithoutGNOIFile/firmware_fails_without_gnoi_file_service
--- PASS: TestGNMIFirmwareWithoutGNOIFile (1.02s)
PASS
```

## Integration with CI/CD

The tests are designed to:
- ✅ **Be Deterministic**: Use temporary directories and controlled test data
- ✅ **Be Fast**: Complete in under 5 seconds typically
- ✅ **Be Isolated**: Each test creates its own temporary environment
- ✅ **Be Comprehensive**: Cover both success and failure scenarios
- ✅ **Be Maintainable**: Clear test names and good error messages

## Test Coverage

The test suite provides comprehensive coverage of:

| Component | Coverage |
|-----------|----------|
| **gNMI Path Parsing** | ✅ All firmware paths tested |
| **gNOI File Service Integration** | ✅ Service dependency verified |
| **File System Operations** | ✅ Files, directories, nested structures |
| **Error Handling** | ✅ All error conditions tested |
| **Response Formatting** | ✅ JSON structure and content verified |
| **Client Convenience Methods** | ✅ All methods tested |
| **Service Configuration** | ✅ EnableGNOIFile dependency tested |

This comprehensive test suite ensures the firmware file listing functionality is robust, reliable, and ready for production use.
