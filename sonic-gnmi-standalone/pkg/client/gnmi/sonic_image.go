package gnmi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// SonicImageFileInfo represents information about a SONIC image file.
type SonicImageFileInfo struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	IsDirectory bool      `json:"is_directory"`
	Permissions string    `json:"permissions"`
}

// SonicImageFilesResponse represents the response for SONIC image files listing.
type SonicImageFilesResponse struct {
	Directory string               `json:"directory"`
	FileCount int                  `json:"file_count"`
	Files     []SonicImageFileInfo `json:"files"`
}

// GetSonicImageFiles retrieves a list of SONIC image files in the specified directory.
// This is a convenience method that constructs the appropriate gNMI path and
// handles the response parsing.
func (c *Client) GetSonicImageFiles(ctx context.Context, directory string) (*SonicImageFilesResponse, error) {
	if directory == "" {
		return nil, fmt.Errorf("SONIC image directory is required")
	}

	// Construct the gNMI path: /sonic/system/sonic-image[directory=<directory>]/files
	path := &gnmi.Path{
		Elem: []*gnmi.PathElem{
			{Name: "sonic"},
			{Name: "system"},
			{
				Name: "sonic-image",
				Key:  map[string]string{"directory": directory},
			},
			{Name: "files"},
		},
	}

	glog.V(2).Infof("Requesting SONIC image files for directory: %s", directory)

	// Make the gNMI Get request
	resp, err := c.Get(ctx, []*gnmi.Path{path}, gnmi.Encoding_JSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get SONIC image files for directory %s: %w", directory, err)
	}

	// Parse the response
	if len(resp.Notification) == 0 || len(resp.Notification[0].Update) == 0 {
		return nil, fmt.Errorf("no data received for directory %s", directory)
	}

	update := resp.Notification[0].Update[0]
	jsonVal := update.Val.GetJsonVal()
	if jsonVal == nil {
		return nil, fmt.Errorf("expected JSON response, got %T", update.Val.Value)
	}

	// Unmarshal the JSON response
	var sonicImageResp SonicImageFilesResponse
	if err := json.Unmarshal(jsonVal, &sonicImageResp); err != nil {
		return nil, fmt.Errorf("failed to parse SONIC image files response: %w", err)
	}

	glog.V(2).Infof("Retrieved %d SONIC image files from directory %s",
		sonicImageResp.FileCount, directory)

	return &sonicImageResp, nil
}

// GetSonicImageFileCount retrieves only the count of SONIC image files in the specified directory.
func (c *Client) GetSonicImageFileCount(ctx context.Context, directory string) (int, error) {
	if directory == "" {
		return 0, fmt.Errorf("SONIC image directory is required")
	}

	// Construct the gNMI path: /sonic/system/sonic-image[directory=<directory>]/files/count
	path := &gnmi.Path{
		Elem: []*gnmi.PathElem{
			{Name: "sonic"},
			{Name: "system"},
			{
				Name: "sonic-image",
				Key:  map[string]string{"directory": directory},
			},
			{Name: "files"},
			{Name: "count"},
		},
	}

	glog.V(2).Infof("Requesting SONIC image file count for directory: %s", directory)

	// Make the gNMI Get request
	resp, err := c.Get(ctx, []*gnmi.Path{path}, gnmi.Encoding_JSON)
	if err != nil {
		return 0, fmt.Errorf("failed to get SONIC image file count for directory %s: %w", directory, err)
	}

	// Parse the response
	if len(resp.Notification) == 0 || len(resp.Notification[0].Update) == 0 {
		return 0, fmt.Errorf("no data received for directory %s", directory)
	}

	update := resp.Notification[0].Update[0]
	jsonVal := update.Val.GetJsonVal()
	if jsonVal == nil {
		return 0, fmt.Errorf("expected JSON response, got %T", update.Val.Value)
	}

	// Unmarshal the JSON response (should be a simple number)
	var count int
	if err := json.Unmarshal(jsonVal, &count); err != nil {
		return 0, fmt.Errorf("failed to parse SONIC image file count response: %w", err)
	}

	return count, nil
}

// SonicImageFileResponse represents the response for a specific SONIC image file.
type SonicImageFileResponse struct {
	Directory string             `json:"directory"`
	File      SonicImageFileInfo `json:"file"`
}

// GetSonicImageFileInfo retrieves information about a specific SONIC image file.
func (c *Client) GetSonicImageFileInfo(ctx context.Context, directory string, filename string) (*SonicImageFileResponse, error) {
	if directory == "" {
		return nil, fmt.Errorf("SONIC image directory is required")
	}
	if filename == "" {
		return nil, fmt.Errorf("SONIC image filename is required")
	}

	// Construct the gNMI path: /sonic/system/sonic-image[directory=<directory>]/files/<filename>
	path := &gnmi.Path{
		Elem: []*gnmi.PathElem{
			{Name: "sonic"},
			{Name: "system"},
			{
				Name: "sonic-image",
				Key:  map[string]string{"directory": directory},
			},
			{Name: "files"},
			{Name: filename},
		},
	}

	glog.V(2).Infof("Requesting SONIC image file info for %s in directory: %s", filename, directory)

	// Make the gNMI Get request
	resp, err := c.Get(ctx, []*gnmi.Path{path}, gnmi.Encoding_JSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get SONIC image file info for %s in directory %s: %w", filename, directory, err)
	}

	// Parse the response
	if len(resp.Notification) == 0 || len(resp.Notification[0].Update) == 0 {
		return nil, fmt.Errorf("no data received for file %s in directory %s", filename, directory)
	}

	update := resp.Notification[0].Update[0]
	jsonVal := update.Val.GetJsonVal()
	if jsonVal == nil {
		return nil, fmt.Errorf("expected JSON response, got %T", update.Val.Value)
	}

	// Unmarshal the JSON response
	var fileResp SonicImageFileResponse
	if err := json.Unmarshal(jsonVal, &fileResp); err != nil {
		return nil, fmt.Errorf("failed to parse SONIC image file info response: %w", err)
	}

	glog.V(2).Infof("Retrieved info for SONIC image file %s: %d bytes, modified %s",
		filename, fileResp.File.Size, fileResp.File.ModTime)

	return &fileResp, nil
}
