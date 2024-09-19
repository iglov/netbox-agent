package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/iglov/netbox-agent/lib/dmidecode"
	"github.com/iglov/netbox-agent/lib/ipmi"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"os"
	"strings"

	"github.com/netbox-community/go-netbox/v4"
)

var (
	version  = flag.Bool("v", false, "Print current version and exit.")
	logLevel = flag.String("loglevel", "info", "Set log level: DEBUG, INFO, WARN, ERROR")
)

// Version contains main version of build. Get from compiler variables
var Version string

// Initiate log
var log = logrus.New()

// FullSystemInfo is the common struct for all of our hardware components
type FullSystemInfo struct {
	Memory  []dmidecode.MemoryDeviceInfo `json:"memory"`
	CPU     []dmidecode.CPUInfo          `json:"cpu"`
	IPMI    ipmi.BmcInfo                 `json:"ipmi"`
	Chassis []dmidecode.ChassisInfo      `json:"chassis"`
	System  []dmidecode.SystemInfo       `json:"system"`
}

func main() {

	flag.Parse()

	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}

	// Set log output to stdout
	log.Out = os.Stdout

	// Parse the log level and set it
	level, err := logrus.ParseLevel(strings.ToLower(*logLevel))
	if err != nil {
		log.Fatalf("Invalid log level: %s", *logLevel)
	}
	log.SetLevel(level)

	// Fetch memory device information
	memDevices, err := dmidecode.GetMemoryDevices()
	if err != nil {
		log.Fatalf("Error fetching memory devices: %s", err)
	}

	// Fetch CPU information
	cpuInfo, err := dmidecode.GetCPUInfo()
	if err != nil {
		log.Fatalf("Error fetching CPU information: %s", err)
	}

	// Fetch IPMI information
	bmcInfo := ipmi.GetBmcInfo()

	// Fetch chassis information
	chassisInfo, err := dmidecode.GetChassisInfo()
	if err != nil {
		log.Fatalf("Error fetching chassis information: %s", err)
	}

	// Fetch system information
	systemInfo, err := dmidecode.GetSystemInfo()
	if err != nil {
		log.Fatalf("Error fetching system information: %s", err)
	}

	// Combine memory and CPU data into SystemInfo struct
	fullSystemInfo := FullSystemInfo{
		Memory:  memDevices,
		CPU:     cpuInfo,
		Chassis: chassisInfo,
		IPMI:    bmcInfo,
		System:  systemInfo,
	}

	// Convert the SystemInfo struct to JSON
	finalJSON, err := json.MarshalIndent(fullSystemInfo, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling final JSON: %s", err)
	}

	log.Debug(string(finalJSON))

	err = godotenv.Load()
	if err != nil {
		log.Warn("Error loading .env , using local variables")
	}

	ctx := context.Background()

	apiURL := os.Getenv("API_URL")
	apiTOKEN := os.Getenv("API_TOKEN")

	c := netbox.NewAPIClientFor(apiURL, apiTOKEN)

	roleRequest := netbox.NewDeviceRoleRequestWithDefaults()
	roleRequest.SetName("default device role")
	roleRequest.SetSlug("default-device-role")
	roleRequest.SetDescription("It's just a default role after server creation by API, it should be changed after server creation.")
	roleRes, httpRes, err := c.DcimAPI.DcimDeviceRolesCreate(ctx).DeviceRoleRequest(*roleRequest).Execute()

	if err != nil {
		log.Errorf("Error creating role: %s", err)
	}

	log.Debugf("Response: %+v", roleRes)
	log.Debugf("HTTP Response: %+v", httpRes)

	hostname, err := os.Hostname()
	if err != nil {
		log.Errorf("Error get hostname: %s", err)
	}

	// Parse and set all variables
	site := strings.Split(hostname, ".")[1]
	productName := fullSystemInfo.System[0].ProductName
	productVendor := fullSystemInfo.System[0].Manufacturer
	productSerial := fullSystemInfo.System[0].SerialNumber
	chassisVersion := fullSystemInfo.Chassis[0].Version
	chassisSerial := fullSystemInfo.Chassis[0].SerialNumber
	chassisVendor := fullSystemInfo.Chassis[0].Manufacturer

	otherInfo := "Serial: " + productSerial + " | Chassis name: " + chassisVersion + " | Chassis serial: " + chassisSerial + " | Chassis vendor: " + chassisVendor

	productVendorLower := strings.ToLower(productVendor)
	productNameLower := strings.ToLower(productName)
	productNameHyphenated := strings.ReplaceAll(productNameLower, " ", "-")
	productVendorName := productVendorLower + "-" + productNameHyphenated

	chassisVendorLower := strings.ToLower(chassisVendor)
	chassisVersionLower := strings.ToLower(chassisVersion)
	chassisVersionHyphenated := strings.ReplaceAll(chassisVersionLower, " ", "-")
	chassisVendorName := chassisVendorLower + "-" + chassisVersionHyphenated

	// Start creating objects
	siteRequest := netbox.NewWritableSiteRequestWithDefaults()
	siteRequest.SetName(site)
	siteRequest.SetSlug(site)
	siteRequest.SetDescription("It's just a default Site after server creation by API, it should be changed after server creation.")
	siteRes, httpRes, err := c.DcimAPI.DcimSitesCreate(ctx).WritableSiteRequest(*siteRequest).Execute()

	if err != nil {
		log.Errorf("Error creating site: %s", err)
	}

	log.Debugf("Response: %+v", siteRes)
	log.Debugf("HTTP Response: %+v", httpRes)

	// Add blade chassis if exists
	if chassisSerial != productSerial {
		device := netbox.NewWritableDeviceWithConfigContextRequestWithDefaults()
		device.SetSite(netbox.SiteRequest{Name: site, Slug: site})
		device.SetRole(netbox.DeviceRoleRequest{Name: "default device role", Slug: "default-device-role"})
		device.SetComments(otherInfo)
		device.SetDeviceType(netbox.DeviceTypeRequest{Model: chassisVersion, Slug: chassisVendorName, Manufacturer: netbox.ManufacturerRequest{Name: chassisVendor, Slug: chassisVendorLower}})
		device.SetName(chassisSerial)
		device.SetSerial(chassisSerial)

		deviceRes, httpRes, err := c.DcimAPI.DcimDevicesCreate(ctx).WritableDeviceWithConfigContextRequest(*device).Execute()

		if err != nil {
			log.Errorf("Error creating device: %v", err)
		}

		log.Debugf("Response: %+v", deviceRes)
		log.Debugf("HTTP Response: %+v", httpRes)

	}

	device := netbox.NewWritableDeviceWithConfigContextRequestWithDefaults()
	device.SetSite(netbox.SiteRequest{Name: site, Slug: site})
	device.SetRole(netbox.DeviceRoleRequest{Name: "default device role", Slug: "default-device-role"})
	device.SetComments(otherInfo)
	device.SetDeviceType(netbox.DeviceTypeRequest{Model: productName, Slug: productVendorName, Manufacturer: netbox.ManufacturerRequest{Name: productVendor, Slug: productVendorLower}})
	device.SetName(hostname)
	device.SetSerial(productSerial)
	device.SetLocalContextData(&fullSystemInfo)

	deviceRes, httpRes, err := c.DcimAPI.DcimDevicesCreate(ctx).WritableDeviceWithConfigContextRequest(*device).Execute()

	if err != nil {
		log.Errorf("Error creating device: %v", err)
	}

	log.Debugf("Response: %+v", deviceRes)
	log.Debugf("HTTP Response: %+v", httpRes)

	for i := range fullSystemInfo.CPU {
		man := netbox.NewManufacturerRequestWithDefaults()
		man.SetName(fullSystemInfo.CPU[i].Manufacturer)
		man.SetSlug(strings.ToLower(fullSystemInfo.CPU[i].Manufacturer))
		manRes, httpRes, err := c.DcimAPI.DcimManufacturersCreate(ctx).ManufacturerRequest(*man).Execute()
		if err != nil {
			log.Errorf("Error creating device: %v", err)
		}

		log.Debugf("Response: %+v", manRes)
		log.Debugf("HTTP Response: %+v", httpRes)

		inv := netbox.NewInventoryItemRequestWithDefaults()
		inv.SetName("CPU")
		inv.SetManufacturer(*man)
		inv.SetPartId(fullSystemInfo.CPU[i].Version)
		inv.SetCustomFields(map[string]interface{}{
			"cpu_cores":   fullSystemInfo.CPU[i].CoreCount,
			"cpu_threads": fullSystemInfo.CPU[i].ThreadCount,
		})
		inv.SetDevice(netbox.DeviceRequest{Name: *netbox.NewNullableString(&hostname)})
		invRes, httpRes, err := c.DcimAPI.DcimInventoryItemsCreate(ctx).InventoryItemRequest(*inv).Execute()

		if err != nil {
			log.Errorf("Error creating device: %v", err)
		}

		log.Debugf("Response: %+v", invRes)
		log.Debugf("HTTP Response: %+v", httpRes)
	}

	for i := range fullSystemInfo.Memory {
		man := netbox.NewManufacturerRequestWithDefaults()
		man.SetName(fullSystemInfo.Memory[i].Manufacturer)
		man.SetSlug(strings.ToLower(fullSystemInfo.Memory[i].Manufacturer))
		manRes, httpRes, err := c.DcimAPI.DcimManufacturersCreate(ctx).ManufacturerRequest(*man).Execute()
		if err != nil {
			log.Errorf("Error creating device: %v", err)
		}

		log.Debugf("Response: %+v", manRes)
		log.Debugf("HTTP Response: %+v", httpRes)

		inv := netbox.NewInventoryItemRequestWithDefaults()
		inv.SetName("MEMORY")
		inv.SetManufacturer(*man)
		inv.SetPartId(fullSystemInfo.Memory[i].PartNumber)
		inv.SetSerial(fullSystemInfo.Memory[i].SerialNumber)
		inv.SetCustomFields(map[string]interface{}{
			"memory_size":  fullSystemInfo.Memory[i].Size,
			"memory_slot":  fullSystemInfo.Memory[i].DeviceLocator,
			"memory_speed": fullSystemInfo.Memory[i].Speed,
			"memory_type":  fullSystemInfo.Memory[i].Type,
		})
		inv.SetDevice(netbox.DeviceRequest{Name: *netbox.NewNullableString(&hostname)})
		invRes, httpRes, err := c.DcimAPI.DcimInventoryItemsCreate(ctx).InventoryItemRequest(*inv).Execute()

		if err != nil {
			log.Errorf("Error creating device: %v", err)
		}

		log.Debugf("Response: %+v", invRes)
		log.Debugf("HTTP Response: %+v", httpRes)
	}

	netInt := netbox.NewWritableInterfaceRequestWithDefaults()
	netInt.SetName("IMPI")
	netInt.SetDevice(netbox.DeviceRequest{Name: *netbox.NewNullableString(&hostname)})
	netInt.SetType("1000base-tx")
	netIntRes, httpRes, err := c.DcimAPI.DcimInterfacesCreate(ctx).WritableInterfaceRequest(*netInt).Execute()

	if err != nil {
		log.Errorf("Error creating device: %v", err)
	}

	log.Debugf("Response: %+v", netIntRes)
	log.Debugf("HTTP Response: %+v", httpRes)

}
