package file

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSonicImageFile(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
		reason   string
	}{
		// SONIC OS images (should return true)
		{"sonic-vs.4.0.0.img", true, "SONIC OS image with .img extension"},
		{"sonic-broadcom.3.5.2.iso", true, "SONIC OS image with .iso extension"},
		{"sonic-mellanox.qcow2", true, "SONIC OS image with .qcow2 extension"},
		{"SONIC-innovium.4.1.0.vmdk", true, "SONIC OS image with .vmdk extension (case insensitive)"},
		{"sonic-system-image.img", true, "File containing 'sonic' in name"},

		// Device firmware files (should return false - these are what we want to avoid confusion with)
		{"device-driver.bin", false, "Device firmware with .bin extension (no sonic in name)"},
		{"bootloader.fw", false, "Firmware file with .fw extension"},
		{"network-adapter.bin", false, "Network adapter firmware"},
		{"switch-firmware.bin", false, "Switch firmware file"},
		{"driver1.bin", false, "Generic driver firmware"},

		// Other system files (should return false)
		{"README.txt", false, "Documentation file"},
		{"config.json", false, "Configuration file"},
		{"log.txt", false, "Log file"},
		{"system.tar.gz", false, "Archive file"},

		// Edge cases
		{"", false, "Empty filename"},
		{"sonic", true, "Just 'sonic' in filename"},
		{"SONIC.IMG", true, "All caps with valid extension"},
		{"generic-system.img", false, ".img extension but no 'sonic' in name"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := IsSonicImageFile(tc.filename)
			assert.Equal(t, tc.expected, result, "Failed for %s: %s", tc.filename, tc.reason)
		})
	}
}
