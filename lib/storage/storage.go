package storage

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

// DiskInfo represents information about a storage device.
type DiskInfo struct {
	Name         string `json:"name,omitempty"`
	Model        string `json:"model,omitempty"` // Add Model
	Manufacturer string `json:"manufacturer,omitempty"`
	ProductName  string `json:"product_name,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	Size         string `json:"size,omitempty"` // Store size as a string cuz of megacli, e.g., "1.090 TB"
	Slot         string `json:"slot,omitempty"`
}

// GetStorageInfo fetches information about the storage devices, handling both simple disks and RAID setups.
func GetStorageInfo() ([]DiskInfo, error) {
	// Check if RAID is present by checking for the existence of MegaCli.
	if _, err := os.Stat("/opt/MegaRAID/MegaCli/MegaCli64"); err == nil {
		return getRAIDDiskInfo()
	}

	// If MegaCli is not present, gather simple disk information.
	return getSimpleDiskInfo()
}

func getSimpleDiskInfo() ([]DiskInfo, error) {
	disks := []DiskInfo{}
	sysBlock := "/sys/block"
	devices, err := os.ReadDir(sysBlock)
	if err != nil {
		return nil, fmt.Errorf("error reading /sys/block: %v", err)
	}

	for _, link := range devices {
		fullpath := path.Join(sysBlock, link.Name())
		dev, err := os.Readlink(fullpath)
		if err != nil {
			continue
		}

		if strings.HasPrefix(dev, "../devices/virtual/") {
			continue
		}

		// Filter out floppy and CD/DVD devices.
		if strings.HasPrefix(dev, "../devices/platform/floppy") || slurpFile(path.Join(fullpath, "device", "type")) == "5" {
			continue
		}

		// Extract model information
		modelFull := slurpFile(path.Join(fullpath, "device", "model"))
		var manufacturer, model string
		if modelFull != "" {
			parts := strings.Fields(modelFull)
			if len(parts) >= 2 {
				manufacturer = parts[0]             // First part is the manufacturer
				model = strings.Join(parts[1:], "") // Remaining parts make up the model
			} else if len(parts) == 1 {
				manufacturer = parts[0] // Use the entire field as manufacturer if it's single-worded
				model = "Unknown"       // Model is unknown
			}
		}

		device := DiskInfo{
			Name:         link.Name(),
			Manufacturer: capitalizeManufacturer(manufacturer), // Extracted manufacturer
			Model:        model,                                // Extracted model
			SerialNumber: getSerial(link.Name(), fullpath),
		}

		// Extract vendor if available, does not start with 0x, and is not ATA.
		if vendor := slurpFile(path.Join(fullpath, "device", "vendor")); !strings.HasPrefix(vendor, "0x") && vendor != "ATA" {
			device.Manufacturer = capitalizeManufacturer(vendor)
		}

		// Capture size as string from the size file and convert it to human-readable format.
		sizeBytesStr := slurpFile(path.Join(fullpath, "size"))
		sizeBytes, _ := strconv.ParseUint(sizeBytesStr, 10, 64)
		if sizeBytes > 0 {
			sizeGB := float64(sizeBytes) / 1953125.0 // Convert to GiB
			device.Size = fmt.Sprintf("%.3f GB", sizeGB)
		} else {
			device.Size = "0 GB"
		}

		disks = append(disks, device)
	}
	return disks, nil
}

// getRAIDDiskInfo gathers information from MegaCli for RAID setups.
func getRAIDDiskInfo() ([]DiskInfo, error) {
	// Run MegaCli to gather disk information.
	cmd := exec.Command("/opt/MegaRAID/MegaCli/MegaCli64", "-PDList", "-aALL")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running MegaCli: %v", err)
	}

	return parseMegaCliOutput(output), nil
}

// parseMegaCliOutput parses the output from MegaCli to extract disk information.
func parseMegaCliOutput(output []byte) []DiskInfo {
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var disks []DiskInfo
	var disk DiskInfo

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// If we encounter a Slot Number, start a new disk entry
		if strings.HasPrefix(line, "Slot Number:") {
			if disk.Slot != "" {
				// Append the previous disk info before starting a new one
				disks = append(disks, disk)
			}
			disk = DiskInfo{
				Slot: strings.TrimSpace(strings.TrimPrefix(line, "Slot Number:")),
			}
		}

		// Extract Raw Size as a string
		if strings.HasPrefix(line, "Raw Size:") {
			sizeStr := strings.Split(strings.TrimPrefix(line, "Raw Size:"), "[")[0]
			disk.Size = strings.TrimSpace(sizeStr) // Store size directly as a string
		}

		// Extract Inquiry Data
		if strings.HasPrefix(line, "Inquiry Data:") {
			parts := strings.Fields(strings.TrimPrefix(line, "Inquiry Data:"))
			if len(parts) >= 3 {
				disk.Manufacturer = capitalizeManufacturer(parts[0]) // Vendor
				disk.Model = parts[1]                                // Model
				disk.SerialNumber = parts[2]                         // Serial Number
			}
		}
	}

	// Append the last disk info if it wasn't added
	if disk.Slot != "" {
		disks = append(disks, disk)
	}

	return disks
}

// Read one-liner text files, strip newline.
func slurpFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// If a deferred function has any return values, they are discarded when the function completes.
func Check(f func() error) {
	if err := f(); err != nil {
		fmt.Println("Received error:", err)
	}
}

// getSerial extracts the serial number from udev for a given disk.
func getSerial(name, fullpath string) (serial string) {
	var f *os.File
	var err error

	// Try modern location/format of the udev database.
	if dev := slurpFile(path.Join(fullpath, "dev")); dev != "" {
		f, err = os.Open(path.Join("/run/udev/data", "b"+dev))
		if err == nil {
			defer Check(f.Close)
			s := bufio.NewScanner(f)
			for s.Scan() {
				if sl := strings.Split(s.Text(), "="); len(sl) == 2 {
					if sl[0] == "E:ID_SERIAL_SHORT" {
						serial = sl[1]
						break
					}
				}
			}
			return
		}
	}

	// Try legacy location/format of the udev database.
	f, err = os.Open(path.Join("/dev/.udev/db", "block:"+name))
	if err == nil {
		defer Check(f.Close)
		s := bufio.NewScanner(f)
		for s.Scan() {
			if sl := strings.Split(s.Text(), "="); len(sl) == 2 {
				if sl[0] == "E:ID_SERIAL_SHORT" {
					serial = sl[1]
					break
				}
			}
		}
	}

	// No serial if both locations failed.
	return
}

// Helper function to capitalize the first letter of a string.
func capitalizeManufacturer(manufacturer string) string {
	if len(manufacturer) == 0 {
		return manufacturer
	}
	return strings.ToUpper(manufacturer[:1]) + strings.ToLower(manufacturer[1:])
}
