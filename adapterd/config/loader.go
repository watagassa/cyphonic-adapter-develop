package config

import (
	"errors"
	"fmt"
	"os"
)

type GlobalConfig struct {
	AS []struct {
		Fqdn string `yaml:"fqdn"`
		Port int    `yaml:"port"`
	} `yaml:"as"`
	NMS []struct {
		Fqdn string `yaml:"fqdn"`
		Port int    `yaml:"port"`
	} `yaml:"nms"`
	TRS []struct {
		Fqdn string `yaml:"fqdn"`
		Port int    `yaml:"port"`
	} `yaml:"trs"`
	Controller []struct {
		Fqdn string `yaml:"fqdn"`
		Port int    `yaml:"port"`
	} `yaml:"controller"`
}

type AdapterdConfig struct {
	AuthType                           int    `yaml:"auth_type"`
	AdapterID                          string `yaml:"adapter_id"`
	Password                           string `yaml:"password"`
	LoginRequestCertificatePath        string `yaml:"login_request_certificate_path"`
	DnsTunAddress                      string `yaml:"dns_tun_address"`
	DnsTun6Address                     string `yaml:"dns_tun6_address"`
	InternalInterface                  string `yaml:"internal_interface"`
	ChildDeviceInformationPath         string `yaml:"child_device_information_path"`
	RootCertificatePath                string `yaml:"root_certificate_path"`
	TlsClientCertificatePath           string `yaml:"tls_client_certificate_path"`
	TlsClientCertificatePrivateKeyPath string `yaml:"tls_client_certificate_privatekey_path"`
	VirtualIPType                      string `yaml:"virtual_ip_type"`
}

type LoggingConfig struct {
	Output   bool   `yaml:"output"`
	Encoding string `yaml:"encoding"`
	LogDir   string `yaml:"log_directory"`
	FileName string `yaml:"log_file_name"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type YamlConfig struct {
	GlobalConfig   `yaml:"global"`
	AdapterdConfig `yaml:"adapterd"`
	LoggingConfig  `yaml:"logging"`
	RedisConfig    `yaml:"redis"`
}

// load loads the config file and returns the bytes.
func load() ([]byte, error) {
	var yamlFile []byte
	var err error

	filePath, err := findConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find config file: %w", err)
	}

	yamlFile, err = os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return yamlFile, nil
}

// findConfigFile returns the path of the config file.
func findConfigFile() (string, error) {
	filePath := checkFileExists(configRootPath+"/config.yml", configRootPath+"/config.yaml")
	if filePath != "" {
		return filePath, nil
	} else {
		err := errors.New("config file path not found")
		return "", err
	}
}

// checkFileExists checks if the file exists and returns the file path.
func checkFileExists(files ...string) string {
	for _, file := range files {
		_, err := os.Stat(file)
		if err == nil {
			return file
		}
	}

	return ""
}
