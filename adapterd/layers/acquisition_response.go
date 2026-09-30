// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"encoding/json"
	"os"
	"time"
)

// AcquisitionResponse stores the result of the General node Information Acquisition process.
// CYPHONIC adapter stores API results (General node Information Acquisition Response) from Cloud Control Service.
type AcquisitionResponse struct {
	ID         int       `json:"id"`
	DeviceName string    `json:"device_name"`
	DeviceID   string    `json:"device_id"`
	Password   string    `json:"password"`
	AuthType   int       `json:"auth_type"`
	MacAddress string    `json:"mac_address"`
	EnableIPv6 bool      `json:"enable_ipv6"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ChildDeviceInformationSave saves the obtained general node information as JSON data.
func ChildDeviceInformationSave(info []AcquisitionResponse, childDeviceInfoPath string) error {
	file, err := os.Create(childDeviceInfoPath)
	if err != nil {
		return err
	}
	defer file.Close()

	json.NewEncoder(file).Encode(info)

	return nil
}
