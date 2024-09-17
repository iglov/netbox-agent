package ipmi

import (
	"fmt"
	"log"

	"github.com/u-root/u-root/pkg/ipmi"
)

type BmcInfo struct {
	DeviceID  string `json:"deviceID"`
	DeviceRev string `json:"deviceRevision"`
	FwRev     string `json:"firmwareRevision"`
	IpmiVer   string `json:"ipmiVersion"`
	ManID     string `json:"manufacturerID"`
	ProdID    string `json:"productID"`
	Ipaddr    string `json:"ipAddress"`
	Subnet    string `json:"subnetMask"`
	Macaddr   string `json:"macAddress"`
}

// If a deferred function has any return values, they are discarded when the function completes
func Check(f func() error) {
	if err := f(); err != nil {
		fmt.Println("Received error:", err)
	}
}

func GetBmcInfo() BmcInfo {
	bmc := BmcInfo{}
	laninfo := LanConfig()
	devinfo := DeviceInfo()

	bmc.Ipaddr = laninfo["ipaddress"]
	bmc.Subnet = laninfo["subnetmask"]
	bmc.Macaddr = laninfo["macaddress"]
	bmc.DeviceID = devinfo["deviceID"]
	bmc.DeviceRev = devinfo["deviceRevision"]
	bmc.FwRev = devinfo["firmwareRevision"]
	bmc.IpmiVer = devinfo["ipmiVersion"]
	bmc.ManID = devinfo["manufacturerID"]
	bmc.ProdID = devinfo["productID"]

	return bmc
}

func LanConfig() map[string]string {
	netMap := make(map[string]string)

	const (
		setInProgress byte = iota
		_
		_
		IPAddress
		IPAddressSrc
		MACAddress
		SubnetMask
	)

	ipmi, err := ipmi.Open(0)
	if err != nil {
		log.Fatal(err)
	}
	defer Check(ipmi.Close)

	// data 1   completion code
	// data 2   parameter revision, 0x11
	// data 3:N data

	// ip address
	if ipaddress, err := ipmi.GetLanConfig(1, IPAddress); err != nil {
		log.Panic("Could not get an IP address.")
	} else {
		if len(ipaddress) == 6 {
			netMap["ipaddress"] = fmt.Sprintf("%d.%d.%d.%d", ipaddress[2], ipaddress[3], ipaddress[4], ipaddress[5])
		} else {
			netMap["ipaddress"] = "Unknown"
		}
	}
	// MAC address
	if macaddress, err := ipmi.GetLanConfig(1, MACAddress); err != nil {
		log.Panic("Could not get a MAC address.")
	} else {
		if len(macaddress) == 8 {
			netMap["macaddress"] = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", macaddress[2], macaddress[3], macaddress[4], macaddress[5], macaddress[6], macaddress[7])
		} else {
			netMap["macaddress"] = "Unknown"
		}
	}
	// subnet mask
	if subnetmask, err := ipmi.GetLanConfig(1, SubnetMask); err != nil {
		log.Panic("Could not get a Subnet Mask.")
	} else {
		if len(subnetmask) == 6 {
			netMap["subnetmask"] = fmt.Sprintf("%d.%d.%d.%d", subnetmask[2], subnetmask[3], subnetmask[4], subnetmask[5])
		} else {
			netMap["subnetmask"] = "Unknown"
		}
	}

	return netMap
}

func DeviceInfo() map[string]string {

	devMap := make(map[string]string)

	ipmi, err := ipmi.Open(0)
	if err != nil {
		fmt.Printf("Failed to open ipmi device: %v\n", err)
	}
	defer Check(ipmi.Close)

	if info, err := ipmi.GetDeviceID(); err != nil {
		fmt.Printf("Failed to get device ID information: %v\n", err)
	} else {
		devMap["deviceID"] = fmt.Sprintf("%d", info.DeviceID)
		devMap["deviceRevision"] = fmt.Sprintf("%d", (info.DeviceRevision & 0x0F))
		devMap["firmwareRevision"] = fmt.Sprintf("%d.%02x", (info.FwRev1 & 0x3F), info.FwRev2)
		spec := uint8(info.IpmiVersion)
		devMap["ipmiVersion"] = fmt.Sprintf("%x.%x", spec&0x0F, (spec&0xF0)>>4)

		var mid uint32
		mid = uint32(info.ManufacturerID[2]) << 16
		mid |= uint32(info.ManufacturerID[1]) << 8
		mid |= uint32(info.ManufacturerID[0])

		devMap["manufacturerID"] = fmt.Sprintf("%d (0x%04X)", mid, mid)

		var pid uint16
		pid = uint16(info.ProductID[1]) << 8
		pid |= uint16(info.ProductID[0])

		devMap["productID"] = fmt.Sprintf("%d (0x%04X)", pid, pid)
	}

	return devMap
}
