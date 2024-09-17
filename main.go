package main

import (
    "strings"
    "context"
    "log"
    "fmt"
    "os"
    "encoding/json"
    "github.com/joho/godotenv"
    "github.com/iglov/netbox-agent/lib/ipmi"
    "github.com/iglov/netbox-agent/lib/dmidecode"

    "github.com/netbox-community/go-netbox/v4"
)

type FullSystemInfo struct {
    Memory  []dmidecode.MemoryDeviceInfo `json:"memory"`
    CPU     []dmidecode.CPUInfo          `json:"cpu"`
    IPMI    ipmi.BmcInfo                 `json:"ipmi"`
    Chassis []dmidecode.ChassisInfo      `json:"chassis"`
    System  []dmidecode.SystemInfo       `json:"system"`
}

func main() {

    // Fetch memory device information
    memDevices, err := dmidecode.GetMemoryDevices()
    if err != nil {
        log.Fatalf("Error fetching memory devices: %v", err)
    }

    // Fetch CPU information
    cpuInfo, err := dmidecode.GetCPUInfo()
    if err != nil {
        log.Fatalf("Error fetching CPU information: %v", err)
    }

    // Fetch IPMI information
    bmcInfo := ipmi.GetBmcInfo()

    // Fetch chassis information
    chassisInfo, err := dmidecode.GetChassisInfo()
    if err != nil {
        log.Fatalf("Error fetching chassis information: %v", err)
    }

    // Fetch system information
    systemInfo, err := dmidecode.GetSystemInfo()
    if err != nil {
        log.Fatalf("Error fetching system information: %v", err)
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
        log.Fatalf("Error marshalling final JSON: %v", err)
    }

    // Print the final JSON
    fmt.Println(string(finalJSON))

    godotenv.Load()
    ctx := context.Background()

    API_URL   := os.Getenv("API_URL")
    API_TOKEN := os.Getenv("API_TOKEN")


    c := netbox.NewAPIClientFor(API_URL, API_TOKEN)

    roleRequest := netbox.NewDeviceRoleRequestWithDefaults()
    roleRequest.SetName("default device role")
    roleRequest.SetSlug("default-device-role")
    roleRequest.SetDescription("It's just a default role after server creation by API, it should be changed after server creation.")
    response1, httpRes1, err := c.DcimAPI.DcimDeviceRolesCreate(ctx).DeviceRoleRequest(*roleRequest).Execute()

    if err != nil {
        log.Printf("Error creating device: %v", err) 
    }

    log.Printf("Response: %+v", response1)
    log.Printf("HTTP Response: %+v", httpRes1)

    hostname, err := os.Hostname()
    if err != nil {
            log.Printf("Error get hostname: %v", err) 
    }

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

    // add blade chassis
    if chassisSerial != productSerial {
        device := netbox.NewWritableDeviceWithConfigContextRequestWithDefaults()
        device.SetSite(netbox.SiteRequest{Name: site, Slug: site})
        device.SetRole(netbox.DeviceRoleRequest{Name: "default device role", Slug: "default-device-role"})
        device.SetComments(otherInfo)
        device.SetDeviceType(netbox.DeviceTypeRequest{Model: chassisVersion, Slug: chassisVendorName, Manufacturer: netbox.ManufacturerRequest{Name: chassisVendor, Slug: chassisVendorLower}})
        device.SetName(chassisSerial)
        device.SetSerial(chassisSerial)

        response, httpRes, err := c.DcimAPI.DcimDevicesCreate(ctx).WritableDeviceWithConfigContextRequest(*device).Execute()

        if err != nil {
            log.Printf("Error creating device: %v", err) 
        }

        log.Printf("Response: %+v", response)
        log.Printf("HTTP Response: %+v", httpRes)

    }

    device := netbox.NewWritableDeviceWithConfigContextRequestWithDefaults()
    device.SetSite(netbox.SiteRequest{Name: site, Slug: site})
    device.SetRole(netbox.DeviceRoleRequest{Name: "default device role", Slug: "default-device-role"})
    device.SetComments(otherInfo)
    device.SetDeviceType(netbox.DeviceTypeRequest{Model: productName, Slug: productVendorName, Manufacturer: netbox.ManufacturerRequest{Name: productVendor, Slug: productVendorLower}})
    device.SetName(hostname)
    device.SetSerial(productSerial)
    device.SetLocalContextData(&fullSystemInfo)

    response, httpRes, err := c.DcimAPI.DcimDevicesCreate(ctx).WritableDeviceWithConfigContextRequest(*device).Execute()

    if err != nil {
        log.Printf("Error creating device: %v", err) 
    }

    log.Printf("Response: %+v", response)
    log.Printf("HTTP Response: %+v", httpRes)

    for i := range fullSystemInfo.CPU {
        inv := netbox.NewInventoryItemRequestWithDefaults()
        inv.SetName("CPU")
        inv.SetPartId(fullSystemInfo.CPU[i].Version)
        inv.SetCustomFields(map[string]interface{}{
            "cpu_cores": fullSystemInfo.CPU[i].CoreCount,
            "cpu_threads": fullSystemInfo.CPU[i].ThreadCount,
        })
        inv.SetDevice(netbox.DeviceRequest{Name: *netbox.NewNullableString(&hostname)})
        r1,r2,err := c.DcimAPI.DcimInventoryItemsCreate(context.Background()).InventoryItemRequest(*inv).Execute()

        if err != nil {
            log.Printf("Error creating device: %v", err) 
        }

        log.Printf("Response: %+v", r1)
        log.Printf("HTTP Response: %+v",r2)
    }

    for i := range fullSystemInfo.Memory {
        man := netbox.NewManufacturerRequestWithDefaults()
        man.SetName(fullSystemInfo.Memory[i].Manufacturer)
        man.SetSlug(strings.ToLower(fullSystemInfo.Memory[i].Manufacturer))
        m1,m2,err := c.DcimAPI.DcimManufacturersCreate(context.Background()).ManufacturerRequest(*man).Execute()
        if err != nil {
            log.Printf("Error creating device: %v", err) 
        }

        log.Printf("Response: %+v", m1)
        log.Printf("HTTP Response: %+v",m2)


        inv := netbox.NewInventoryItemRequestWithDefaults()
        inv.SetName("MEMORY")
        inv.SetManufacturer(*man)
        inv.SetPartId(fullSystemInfo.Memory[i].PartNumber)
        inv.SetSerial(fullSystemInfo.Memory[i].SerialNumber)
        inv.SetCustomFields(map[string]interface{}{
            "memory_size": fullSystemInfo.Memory[i].Size / 1024, // to GBs
            "memory_slot": fullSystemInfo.Memory[i].DeviceLocator,
            "memory_speed": fullSystemInfo.Memory[i].Speed,
            "memory_type": fullSystemInfo.Memory[i].Type,
        })
        inv.SetDevice(netbox.DeviceRequest{Name: *netbox.NewNullableString(&hostname)})
        r1,r2,err := c.DcimAPI.DcimInventoryItemsCreate(context.Background()).InventoryItemRequest(*inv).Execute()

        if err != nil {
            log.Printf("Error creating device: %v", err) 
        }

        log.Printf("Response: %+v", r1)
        log.Printf("HTTP Response: %+v",r2)
    }

    int1 := netbox.NewWritableInterfaceRequestWithDefaults()
    int1.SetName("IMPI")
    int1.SetDevice(netbox.DeviceRequest{Name: *netbox.NewNullableString(&hostname)})
    int1.SetType("1000base-tx")
    rr1,rr2,err := c.DcimAPI.DcimInterfacesCreate(context.Background()).WritableInterfaceRequest(*int1).Execute()

    if err != nil {
        log.Printf("Error creating device: %v", err) 
    }

    log.Printf("Response: %+v", rr1)
    log.Printf("HTTP Response: %+v",rr2)

}
