package loopback

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGNMISonicImageLoopback tests the complete client-server loopback
// for the gNMI SONIC image file listing functionality.
func TestGNMISonicImageLoopback(t *testing.T) {
	// Setup test infrastructure
	tempDir := t.TempDir()

	// SONIC image functionality now uses internal file operations, no gNOI File service needed
	testServer := SetupInsecureTestServer(t, tempDir, []string{"gnmi"})
	defer testServer.Stop()

	client := SetupGNMIClient(t, testServer.Addr, 10*time.Second)
	defer client.Close()

	// Create test SONIC image directory structure
	sonicImageDir := filepath.Join(tempDir, "test-sonic-images")
	err := os.MkdirAll(sonicImageDir, 0755)
	require.NoError(t, err, "Failed to create SONIC image directory")

	// Create test SONIC OS image files (complete system images, not firmware)
	testFiles := map[string]string{
		"sonic-vs.4.0.0.img":       "SONIC Virtual Switch OS image v4.0.0",
		"sonic-broadcom.4.1.0.img": "SONIC Broadcom ASIC OS image v4.1.0",
		"sonic-mellanox.3.5.2.iso": "SONIC Mellanox ASIC OS image v3.5.2",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(sonicImageDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err, "Failed to create test file %s", filename)
	}

	// Create subdirectory with archived/older SONIC images
	archiveDir := filepath.Join(sonicImageDir, "archive")
	err = os.MkdirAll(archiveDir, 0755)
	require.NoError(t, err, "Failed to create archive subdirectory")

	err = os.WriteFile(filepath.Join(archiveDir, "sonic-vs.3.0.0.img"), []byte("Archived SONIC Virtual Switch OS image v3.0.0"), 0644)
	require.NoError(t, err, "Failed to create archived image file")

	err = os.WriteFile(filepath.Join(archiveDir, "sonic-innovium.3.5.0.img"), []byte("Archived SONIC Innovium ASIC OS image v3.5.0"), 0644)
	require.NoError(t, err, "Failed to create archived image file")

	ctx := context.Background()

	// Run test cases
	t.Run("capabilities_check", func(t *testing.T) {
		// Get capabilities
		resp, err := client.Capabilities(ctx)
		require.NoError(t, err, "Capabilities RPC failed")

		// Check that no YANG models are registered (as per current implementation)
		assert.Empty(t, resp.SupportedModels, "No YANG models should be registered without proper schema definitions")

		// Check supported encodings
		assert.Contains(t, resp.SupportedEncodings, gnmi.Encoding_JSON, "JSON encoding should be supported")
		assert.Contains(t, resp.SupportedEncodings, gnmi.Encoding_JSON_IETF, "JSON_IETF encoding should be supported")
	})

	t.Run("list_all_sonic_image_files", func(t *testing.T) {
		// Test listing all SONIC image files
		sonicImageResp, err := client.GetSonicImageFiles(ctx, sonicImageDir)
		require.NoError(t, err, "GetSonicImageFiles RPC failed")
		require.NotNil(t, sonicImageResp)

		assert.Equal(t, sonicImageDir, sonicImageResp.Directory, "Directory mismatch")
		assert.Equal(t, 6, sonicImageResp.FileCount, "Expected 6 items (3 OS images + 1 archive dir + 2 archived images)")
		assert.Len(t, sonicImageResp.Files, 6, "Files slice length mismatch")

		// Check that we have the expected file names
		fileNames := make([]string, len(sonicImageResp.Files))
		for i, file := range sonicImageResp.Files {
			fileNames[i] = file.Name
		}

		expectedFiles := []string{
			"sonic-vs.4.0.0.img",
			"sonic-broadcom.4.1.0.img",
			"sonic-mellanox.3.5.2.iso",
			"archive",
			"archive/sonic-vs.3.0.0.img",
			"archive/sonic-innovium.3.5.0.img",
		}

		for _, expectedFile := range expectedFiles {
			assert.Contains(t, fileNames, expectedFile, "Expected file %s not found", expectedFile)
		}

		// Verify file properties
		for _, file := range sonicImageResp.Files {
			assert.NotEmpty(t, file.Name, "File name should not be empty")
			assert.NotZero(t, file.ModTime, "ModTime should not be zero")
			assert.NotEmpty(t, file.Permissions, "Permissions should not be empty")

			if !file.IsDirectory {
				assert.Greater(t, file.Size, int64(0), "File size should be greater than 0")
			}
		}
	})

	t.Run("get_sonic_image_file_count", func(t *testing.T) {
		// Test getting SONIC image file count
		count, err := client.GetSonicImageFileCount(ctx, sonicImageDir)
		require.NoError(t, err, "GetSonicImageFileCount RPC failed")

		assert.Equal(t, 6, count, "Expected 6 items in SONIC image directory (3 OS images + 1 archive dir + 2 archived images)")
	})

	t.Run("get_specific_sonic_image_file_info", func(t *testing.T) {
		// Test getting info for a specific SONIC OS image file
		fileResp, err := client.GetSonicImageFileInfo(ctx, sonicImageDir, "sonic-vs.4.0.0.img")
		require.NoError(t, err, "GetSonicImageFileInfo RPC failed")
		require.NotNil(t, fileResp)

		assert.Equal(t, sonicImageDir, fileResp.Directory, "Directory mismatch")
		assert.Equal(t, "sonic-vs.4.0.0.img", fileResp.File.Name, "File name mismatch")
		assert.False(t, fileResp.File.IsDirectory, "File should not be a directory")
		assert.Equal(t, int64(len("SONIC Virtual Switch OS image v4.0.0")), fileResp.File.Size, "File size mismatch")
		assert.NotZero(t, fileResp.File.ModTime, "ModTime should not be zero")
		assert.NotEmpty(t, fileResp.File.Permissions, "Permissions should not be empty")

		// Test archived image file access
		fileResp, err = client.GetSonicImageFileInfo(ctx, sonicImageDir, "archive/sonic-vs.3.0.0.img")
		require.NoError(t, err, "GetSonicImageFileInfo RPC failed for archived image")
		require.NotNil(t, fileResp)

		assert.Equal(t, sonicImageDir, fileResp.Directory, "Directory mismatch")
		assert.Equal(t, "archive/sonic-vs.3.0.0.img", fileResp.File.Name, "Archived image name mismatch")
		assert.False(t, fileResp.File.IsDirectory, "Archived image should not be a directory")
		assert.Equal(t, int64(len("Archived SONIC Virtual Switch OS image v3.0.0")), fileResp.File.Size, "Archived image size mismatch")
	})

	t.Run("get_directory_info", func(t *testing.T) {
		// Test getting info for the archive directory
		fileResp, err := client.GetSonicImageFileInfo(ctx, sonicImageDir, "archive")
		require.NoError(t, err, "GetSonicImageFileInfo RPC failed for archive directory")
		require.NotNil(t, fileResp)

		assert.Equal(t, sonicImageDir, fileResp.Directory, "Directory mismatch")
		assert.Equal(t, "archive", fileResp.File.Name, "Archive directory name mismatch")
		assert.True(t, fileResp.File.IsDirectory, "Archive should be a directory")
		assert.NotZero(t, fileResp.File.ModTime, "Directory ModTime should not be zero")
		assert.NotEmpty(t, fileResp.File.Permissions, "Directory permissions should not be empty")
	})

	// Error handling tests
	t.Run("error_cases", func(t *testing.T) {
		// Test non-existent directory
		nonExistentDir := "/non/existent/sonic-images"
		_, err := client.GetSonicImageFiles(ctx, nonExistentDir)
		require.Error(t, err, "Should fail for non-existent directory")
		assert.Contains(t, err.Error(), "does not exist", "Error should mention directory does not exist")

		// Test non-existent file
		_, err = client.GetSonicImageFileInfo(ctx, sonicImageDir, "nonexistent.bin")
		require.Error(t, err, "Should fail for non-existent file")

		// Test empty directory parameter
		_, err = client.GetSonicImageFiles(ctx, "")
		require.Error(t, err, "Should fail for empty directory")
		assert.Contains(t, err.Error(), "SONIC image directory is required", "Error should mention directory required")

		// Test empty filename parameter
		_, err = client.GetSonicImageFileInfo(ctx, sonicImageDir, "")
		require.Error(t, err, "Should fail for empty filename")
		assert.Contains(t, err.Error(), "SONIC image filename is required", "Error should mention filename required")
	})
}

// TestGNMISonicImageWithNonExistentDirectory tests error handling
// when trying to access non-existent directories.
func TestGNMISonicImageWithNonExistentDirectory(t *testing.T) {
	// Setup test infrastructure
	tempDir := t.TempDir()
	testServer := SetupInsecureTestServer(t, tempDir, []string{"gnmi"})
	defer testServer.Stop()

	client := SetupGNMIClient(t, testServer.Addr, 10*time.Second)
	defer client.Close()

	ctx := context.Background()

	t.Run("sonic_image_fails_with_nonexistent_directory", func(t *testing.T) {
		// Should fail because directory doesn't exist
		_, err := client.GetSonicImageFiles(ctx, "/non/existent/path")
		require.Error(t, err, "Should fail when directory doesn't exist")
		assert.Contains(t, err.Error(), "does not exist", "Error should mention directory does not exist")
	})
}
