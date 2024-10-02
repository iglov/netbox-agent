package dmidecode

import (
	"github.com/yumaojun03/dmidecode"
	"strings"
)

// CPUInfo holds the details of a CPU.
type CPUInfo struct {
	Manufacturer      string `json:"manufacturer"`
	SocketDesignation string `json:"socket_designation"`
	Version           string `json:"version"`
	CoreCount         uint8  `json:"core_count"`
	ThreadCount       uint8  `json:"thread_count"`
}

// GetCPUInfo fetches and returns the CPU information as a list.
func GetCPUInfo() ([]CPUInfo, error) {
	dmi, err := dmidecode.New()
	if err != nil {
		return nil, err
	}

	// Fetch processor information
	cpuInfo, err := dmi.Processor()
	if err != nil {
		return nil, err
	}

	// Convert processor information to CPUInfo structs
	var cpus []CPUInfo
	for _, cpu := range cpuInfo {
		cpus = append(cpus, CPUInfo{
			Manufacturer:      strings.Trim(cpu.Manufacturer, " "),
			SocketDesignation: cpu.SocketDesignation,
			Version:           cpu.Version,
			CoreCount:         cpu.CoreCount,
			ThreadCount:       cpu.ThreadCount,
		})
	}

	return cpus, nil
}
