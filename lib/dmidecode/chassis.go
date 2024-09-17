package dmidecode

import (
	"strings"

	"github.com/yumaojun03/dmidecode"
)

// ChassisInfo holds the details of the chassis.
type ChassisInfo struct {
	Version      string `json:"version"`
	SerialNumber string `json:"serial_number"`
	Height       string `json:"height"`
	Manufacturer string `json:"manufacturer"`
}

// GetChassisInfo fetches and returns the chassis information as a list.
func GetChassisInfo() ([]ChassisInfo, error) {
	dmi, err := dmidecode.New()
	if err != nil {
		return nil, err
	}

	// Fetch chassis information
	chassisInfo, err := dmi.Chassis()
	if err != nil {
		return nil, err
	}

	// Convert chassis information to ChassisInfo structs
	var chassisList []ChassisInfo
	for _, chassis := range chassisInfo {
		// Extract only the first part of Manufacturer (before space)
		manufacturerParts := strings.Split(chassis.Manufacturer, " ")
		manufacturer := manufacturerParts[0]

		chassisList = append(chassisList, ChassisInfo{
			Version:      chassis.Version,
			SerialNumber: chassis.SerialNumber,
			Height:       chassis.Height.String(),
			Manufacturer: manufacturer,
		})
	}

	return chassisList, nil
}
