package file

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
)

// SonicImageFileInfo represents information about a SONIC image file.
type SonicImageFileInfo struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	IsDirectory bool      `json:"is_directory"`
	Permissions string    `json:"permissions"`
}

// ListSonicImageFiles lists all SONIC image files in the specified directory.
// This function walks through the directory and returns information about all files,
// with special focus on SONIC image files (typically .bin files).
func ListSonicImageFiles(directory string, rootFS string) ([]SonicImageFileInfo, error) {
	glog.V(2).Infof("Listing SONIC image files in directory: %s", directory)

	// Resolve the SONIC image directory path with rootFS
	resolvedPath := resolveFilesystemPath(directory, rootFS)

	// Check if directory exists
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", directory)
	} else if err != nil {
		return nil, fmt.Errorf("failed to access directory %s: %v", directory, err)
	}

	var files []SonicImageFileInfo

	// Walk through the directory and collect file information
	err := filepath.WalkDir(resolvedPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			glog.Warningf("Error accessing path %s: %v", path, err)
			return nil // Continue processing other files
		}

		// Skip the root directory itself
		if path == resolvedPath {
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			glog.Warningf("Error getting info for %s: %v", path, err)
			return nil // Continue processing other files
		}

		// Get relative path from the SONIC image directory
		relPath, err := filepath.Rel(resolvedPath, path)
		if err != nil {
			glog.Warningf("Error getting relative path for %s: %v", path, err)
			return nil // Continue processing other files
		}

		// Create file info entry
		fileInfo := SonicImageFileInfo{
			Name:        relPath,
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			IsDirectory: info.IsDir(),
			Permissions: info.Mode().String(),
		}

		files = append(files, fileInfo)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %v", directory, err)
	}

	glog.V(2).Infof("Found %d files in SONIC image directory %s", len(files), directory)
	return files, nil
}

// GetSonicImageFileCount returns the count of files in a SONIC image directory.
func GetSonicImageFileCount(directory string, rootFS string) (int, error) {
	files, err := ListSonicImageFiles(directory, rootFS)
	if err != nil {
		return 0, err
	}
	return len(files), nil
}

// GetSonicImageFileInfo returns information about a specific SONIC image file.
func GetSonicImageFileInfo(directory string, filename string, rootFS string) (*SonicImageFileInfo, error) {
	files, err := ListSonicImageFiles(directory, rootFS)
	if err != nil {
		return nil, err
	}

	// Look for the specific file
	for _, file := range files {
		if file.Name == filename {
			return &file, nil
		}
	}

	return nil, fmt.Errorf("file not found: %s", filename)
}

// IsSonicImageFile checks if a file is likely a SONIC OS image file based on its extension and name.
// SONIC OS images are complete system images (like .img, .iso) not individual firmware files.
func IsSonicImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	lowername := strings.ToLower(filename)

	// Files containing "sonic" in the name are likely SONIC OS images
	containsSonic := strings.Contains(lowername, "sonic")

	if containsSonic {
		return true
	}

	// For files without "sonic" in name, they're only considered SONIC images
	// if they have typical OS image extensions AND are in a SONIC context
	// (This is more restrictive to avoid false positives)
	sonicImageExts := []string{".img", ".iso", ".qcow2", ".vmdk"}
	for _, validExt := range sonicImageExts {
		if ext == validExt {
			// Only return true for these extensions if "sonic" is in the name
			// This prevents generic .img files from being misidentified
			return false // Changed to be more restrictive
		}
	}

	return false
}

// resolveFilesystemPath resolves a filesystem path with the server's rootFS.
// This handles the difference between containerized and bare-metal deployments.
func resolveFilesystemPath(fsPath string, rootFS string) string {
	// If no rootFS is configured or it's root, use the path as-is
	if rootFS == "" || rootFS == "/" {
		return fsPath
	}

	// If the path is already absolute and starts with rootFS, use as-is
	if strings.HasPrefix(fsPath, rootFS) {
		return fsPath
	}

	// For containerized deployments, resolve the path within rootFS
	if strings.HasPrefix(fsPath, "/") {
		// Absolute path - join with rootFS
		return filepath.Join(rootFS, fsPath)
	}

	// Relative path - use as-is (though this is unusual for filesystem queries)
	return fsPath
}
