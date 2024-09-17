package dmidecode

import (
	"strings"

	"github.com/yumaojun03/dmidecode"
)

// MemoryDeviceInfo holds the details of a memory device.
type MemoryDeviceInfo struct {
	Size          uint16 `json:"size"`
	FormFactor    string `json:"form_factor"`
	Speed         uint16 `json:"speed"`
	Type          string `json:"type"`
	Manufacturer  string `json:"manufacturer"`
	SerialNumber  string `json:"serial_number"`
	AssetTag      string `json:"asset_tag"`
	PartNumber    string `json:"part_number"`
	DeviceLocator string `json:"device_locator"`
}

// GetMemoryDevices fetches and returns the memory device information as a list.
func GetMemoryDevices() ([]MemoryDeviceInfo, error) {
	dmi, err := dmidecode.New()
	if err != nil {
		return nil, err
	}

	// Fetch memory devices information
	memDevices, err := dmi.MemoryDevice()
	if err != nil {
		return nil, err
	}

	// Convert memory device information to MemoryDeviceInfo structs
	var memoryDevices []MemoryDeviceInfo
	for _, device := range memDevices {
		// Skip empty or unknown memory slots (where size is 0 or unknown)
		if device.Size == 0 || device.Size == 0xFFFF {
			continue
		}

		// Trim spaces from part number
		partNumber := strings.TrimSpace(device.PartNumber)

		// Add the device to the list
		memoryDevices = append(memoryDevices, MemoryDeviceInfo{
			Size:          device.Size,
			FormFactor:    device.FormFactor.String(),
			Speed:         device.Speed,
			Type:          device.Type.String(),
			Manufacturer:  device.Manufacturer,
			SerialNumber:  device.SerialNumber,
			AssetTag:      device.AssetTag,
			PartNumber:    partNumber,
			DeviceLocator: device.DeviceLocator,
		})
	}

	return memoryDevices, nil
}
