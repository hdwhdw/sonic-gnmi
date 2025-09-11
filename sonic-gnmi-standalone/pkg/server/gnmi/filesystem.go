package gnmi

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sonic-net/sonic-gnmi/sonic-gnmi-standalone/internal/diskspace"
	"github.com/sonic-net/sonic-gnmi/sonic-gnmi-standalone/internal/file"
)

// handleFilesystemPath processes filesystem-related gNMI path requests.
// It supports disk space queries for any filesystem path.
func (s *Server) handleFilesystemPath(path *gnmi.Path) (*gnmi.Update, error) {
	// Extract the filesystem path from the gNMI path
	fsPath, err := extractFilesystemPath(path)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid filesystem path: %v", err)
	}

	// Check if this is a disk space request
	if isDiskSpacePath(path) {
		return s.handleDiskSpaceRequest(path, fsPath)
	}

	// For now, only disk space is supported
	return nil, status.Errorf(codes.NotFound, "unsupported filesystem metric: %s", pathToString(path))
}

// handleSonicImagePath processes SONIC image-related gNMI path requests.
// It supports listing SONIC image files in specified directories when gNOI File service is enabled.
func (s *Server) handleSonicImagePath(path *gnmi.Path) (*gnmi.Update, error) {
	// Extract the SONIC image directory from the gNMI path
	sonicImageDir, err := extractSonicImageDirectory(path)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid SONIC image path: %v", err)
	}

	// Check if this is a SONIC image files request
	if isSonicImageFilesPath(path) {
		return s.handleSonicImageFilesRequest(path, sonicImageDir)
	}

	// For now, only files listing is supported
	return nil, status.Errorf(codes.NotFound, "unsupported SONIC image metric: %s", pathToString(path))
}

// handleDiskSpaceRequest processes disk space queries for a specific filesystem path.
func (s *Server) handleDiskSpaceRequest(path *gnmi.Path, fsPath string) (*gnmi.Update, error) {
	// Validate the disk space path
	if err := validateDiskSpacePath(path); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid disk space path: %v", err)
	}

	// Resolve the filesystem path with rootFS
	resolvedPath := s.resolveFilesystemPath(fsPath)

	glog.V(2).Infof("Getting disk space for filesystem path: %s (resolved: %s)", fsPath, resolvedPath)

	// Get disk space information
	info, err := diskspace.GetDiskSpace(resolvedPath)
	if err != nil {
		glog.Errorf("Failed to get disk space for %s: %v", resolvedPath, err)
		return nil, status.Errorf(codes.Internal, "failed to retrieve disk space for path %s: %v", fsPath, err)
	}

	// Create the response value
	value := map[string]interface{}{
		"path":         fsPath,
		"total-mb":     info.TotalMB,
		"available-mb": info.AvailableMB,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal response: %v", err)
	}

	return &gnmi.Update{
		Path: path,
		Val: &gnmi.TypedValue{
			Value: &gnmi.TypedValue_JsonVal{
				JsonVal: jsonBytes,
			},
		},
	}, nil
}

// handleSonicImageFilesRequest processes SONIC image files queries for a specific directory.
// This method delegates to the gNOI File service for the actual file operations.
func (s *Server) handleSonicImageFilesRequest(path *gnmi.Path, sonicImageDir string) (*gnmi.Update, error) {
	// Determine which field is being requested
	field, err := getSonicImageFileField(path)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid SONIC image files path: %v", err)
	}

	glog.V(2).Infof("Listing SONIC image files in directory: %s using internal file library", sonicImageDir)

	// Use internal file library to get SONIC image files information
	var value interface{}

	switch field {
	case "count":
		count, err := file.GetSonicImageFileCount(sonicImageDir, s.rootFS)
		if err != nil {
			glog.Errorf("Failed to get SONIC image file count in %s: %v", sonicImageDir, err)
			return nil, status.Errorf(codes.Internal, "failed to get SONIC image file count in directory %s: %v", sonicImageDir, err)
		}
		value = count

	case "list":
		files, err := file.ListSonicImageFiles(sonicImageDir, s.rootFS)
		if err != nil {
			glog.Errorf("Failed to list SONIC image files in %s: %v", sonicImageDir, err)
			return nil, status.Errorf(codes.Internal, "failed to list SONIC image files in directory %s: %v", sonicImageDir, err)
		}
		value = map[string]interface{}{
			"directory":  sonicImageDir,
			"file_count": len(files),
			"files":      files,
		}

	default:
		// Look for a specific file
		fileInfo, err := file.GetSonicImageFileInfo(sonicImageDir, field, s.rootFS)
		if err != nil {
			glog.Errorf("Failed to get SONIC image file info for %s in %s: %v", field, sonicImageDir, err)
			return nil, status.Errorf(codes.Internal,
				"failed to get SONIC image file info for %s in directory %s: %v", field, sonicImageDir, err)
		}
		value = map[string]interface{}{
			"directory": sonicImageDir,
			"file":      fileInfo,
		}
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal response: %v", err)
	}

	return &gnmi.Update{
		Path: path,
		Val: &gnmi.TypedValue{
			Value: &gnmi.TypedValue_JsonVal{
				JsonVal: jsonBytes,
			},
		},
	}, nil
}

// resolveFilesystemPath resolves a filesystem path with the server's rootFS.
// This handles the difference between containerized and bare-metal deployments.
func (s *Server) resolveFilesystemPath(fsPath string) string {
	// If no rootFS is configured or it's root, use the path as-is
	if s.rootFS == "" || s.rootFS == "/" {
		return fsPath
	}

	// If the path is already absolute and starts with rootFS, use as-is
	if strings.HasPrefix(fsPath, s.rootFS) {
		return fsPath
	}

	// For containerized deployments, resolve the path within rootFS
	if strings.HasPrefix(fsPath, "/") {
		// Absolute path - join with rootFS
		return filepath.Join(s.rootFS, fsPath)
	}

	// Relative path - use as-is (though this is unusual for filesystem queries)
	return fsPath
}
