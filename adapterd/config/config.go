package config

import (
	"fmt"
	"net/netip"

	"gopkg.in/yaml.v3"
)

type Global struct {
	ASFQDN         string
	ASPort         int
	NMSFQDN        string
	NMSPort        int
	TRSFQDN        string
	TRSPort        int
	ControllerFQDN string
	ControllerPort int
}

type Adapterd struct {
	AuthType                           int
	AdapterID                          string
	Password                           string
	LoginRequestCertificatePath        string
	DnsTunAddress                      netip.Addr
	DnsTun6Address                     netip.Addr
	InternalInterface                  string
	ChildDeviceInformationPath         string
	RootCertificatePath                string
	TlsClientCertificatePath           string
	TlsClientCertificatePrivateKeyPath string
	VirtualIPType                      string
}

type Redis struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type Logging struct {
	Output   bool              // ログ出力の有無
	Encoding string            // ログエンコード方式
	LogDir   string            // ファイル出力先ディレクトリ（指定なし=コンソール出力のみ）
	FileName string            // ファイル名（指定なし=コンソール出力のみ）
	Rotate   *LogRotateSetting // ログローテートの有無
}

// LogRotateSetting: log rotation settings
type LogRotateSetting struct {
	MaxSize    int // MB
	MaxBackups int
	MaxAge     int // days
	Compress   bool
}

type Config struct {
	Global   *Global
	Adapterd *Adapterd
	Logging  *Logging
	Redis    *Redis
}

// Get gets the config from the yaml file and returns the value.
func Get() (*Config, error) {
	var config YamlConfig

	yamlFile, err := load()
	if err != nil {
		return nil, fmt.Errorf("failed to load yaml file: %w", err)
	}

	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	global := &Global{
		ASFQDN:         config.GlobalConfig.AS[0].Fqdn,
		ASPort:         config.GlobalConfig.AS[1].Port,
		NMSFQDN:        config.GlobalConfig.NMS[0].Fqdn,
		NMSPort:        config.GlobalConfig.NMS[1].Port,
		TRSFQDN:        config.GlobalConfig.TRS[0].Fqdn,
		TRSPort:        config.GlobalConfig.TRS[1].Port,
		ControllerFQDN: config.GlobalConfig.Controller[0].Fqdn,
		ControllerPort: config.GlobalConfig.Controller[1].Port,
	}

	adapterd := &Adapterd{
		AuthType:                           config.AdapterdConfig.AuthType,
		AdapterID:                          config.AdapterdConfig.AdapterID,
		Password:                           config.AdapterdConfig.Password,
		LoginRequestCertificatePath:        config.AdapterdConfig.LoginRequestCertificatePath,
		DnsTunAddress:                      netip.MustParseAddr(config.AdapterdConfig.DnsTunAddress),
		DnsTun6Address:                     netip.MustParseAddr(config.AdapterdConfig.DnsTun6Address),
		InternalInterface:                  config.AdapterdConfig.InternalInterface,
		ChildDeviceInformationPath:         config.AdapterdConfig.ChildDeviceInformationPath,
		RootCertificatePath:                config.AdapterdConfig.RootCertificatePath,
		TlsClientCertificatePath:           config.AdapterdConfig.TlsClientCertificatePath,
		TlsClientCertificatePrivateKeyPath: config.AdapterdConfig.TlsClientCertificatePrivateKeyPath,
		VirtualIPType:                      config.AdapterdConfig.VirtualIPType,
	}

	logging := &Logging{
		Output:   config.LoggingConfig.Output,
		Encoding: config.LoggingConfig.Encoding,
		LogDir:   config.LoggingConfig.LogDir,
		FileName: config.LoggingConfig.FileName,
	}

	redis := &Redis{
		Host:     config.RedisConfig.Host,
		Port:     config.RedisConfig.Port,
		Password: config.RedisConfig.Password,
		DB:       config.RedisConfig.DB,
	}

	return &Config{
		Global:   global,
		Adapterd: adapterd,
		Logging:  logging,
		Redis:    redis,
	}, nil
}
