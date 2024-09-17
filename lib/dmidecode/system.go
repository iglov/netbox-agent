package dmidecode

import (
    "strings"

    "github.com/yumaojun03/dmidecode"
)

// SystemInfo holds the details of the system.
type SystemInfo struct {
    Manufacturer      string `json:"manufacturer"`
    ProductName       string `json:"product_name"`
    Version           string `json:"version"`
    SerialNumber      string `json:"serial_number"`
    LocationInChassis string `json:"location_in_chassis"`
}

// GetSystemInfo fetches and returns the system and baseboard information.
func GetSystemInfo() ([]SystemInfo, error) {
    dmi, err := dmidecode.New()
    if err != nil {
        return nil, err
    }

    // Fetch system information
    systemInfo, err := dmi.System()
    if err != nil {
        return nil, err
    }

    // Fetch baseboard information (for LocationInChassis)
    baseboardInfo, err := dmi.BaseBoard()
    if err != nil {
        return nil, err
    }

    // Convert system and baseboard information to SystemInfo structs
    var systemList []SystemInfo
    for i, sys := range systemInfo {
        // Extract only the first part of Manufacturer (before space)
        manufacturerParts := strings.Split(sys.Manufacturer, " ")
        manufacturer := manufacturerParts[0]

        // Get the location_in_chassis from the baseboard information (we assume the baseboard corresponds to the system index)
        locationInChassis := ""
        if i < len(baseboardInfo) {
            locationInChassis = baseboardInfo[i].LocationInChassis
        }

        systemList = append(systemList, SystemInfo{
            Manufacturer:      manufacturer,
            ProductName:       sys.ProductName,
            Version:           sys.Version,
            SerialNumber:      sys.SerialNumber,
            LocationInChassis: locationInChassis,
        })
    }

    return systemList, nil
}
