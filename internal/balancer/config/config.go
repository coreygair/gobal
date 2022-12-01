package config

import (
	"encoding/json"
	"fmt"
	"go-balancer/internal/util"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Strategy StrategyConfig `yaml:"strategy"`
	Backends []BackendInfo  `yaml:"backends"`
	Port     int            `yaml:"port"`
	Sticky   bool           `yaml:"sticky"`
}

type backendInfo struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type BackendInfo struct {
	backendInfo

	URL *url.URL
}

// Just used for testing to get quick infos
func NewBackendInfo(host string, port int) BackendInfo {
	url, err := url.Parse(fmt.Sprintf("http://%s:%d", host, port))
	if err != nil {
		panic(err)
	}

	return BackendInfo{
		backendInfo: backendInfo{
			Host: host,
			Port: port,
		},
		URL: url,
	}
}

func (u *BackendInfo) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var b backendInfo
	err := unmarshal(&b)
	if err != nil {
		return fmt.Errorf("Parsing backend failed: %s", err.Error())
	}

	portValid, portErr := util.ValidatePortInt(b.Port)
	if !portValid {
		return fmt.Errorf("Parsing backend port failed: %s", portErr.Error())
	}

	url, err := url.Parse(fmt.Sprintf("http://%s:%d", b.Host, b.Port))
	if err != nil {
		return fmt.Errorf("Parsing backend host failed: %s", err.Error())
	}

	u.Host = b.Host
	u.Port = b.Port
	u.URL = url

	return nil
}

func (u *BackendInfo) UnmarshalJSON(bytes []byte) error {
	var b backendInfo
	err := json.Unmarshal(bytes, &b)
	if err != nil {
		return fmt.Errorf("Parsing backend failed: %s", err.Error())
	}

	portValid, portErr := util.ValidatePortInt(b.Port)
	if !portValid {
		return fmt.Errorf("Parsing backend port failed: %s", portErr.Error())
	}

	url, err := url.Parse(fmt.Sprintf("http://%s:%d", b.Host, b.Port))
	if err != nil {
		return fmt.Errorf("Parsing backend host failed: %s", err.Error())
	}

	u.Host = b.Host
	u.Port = b.Port
	u.URL = url

	return nil
}

type StrategyConfig struct {
	Name       string      `yaml:"name"`
	Properties interface{} `yaml:"properties"`
}

func ReadConfig(filename string) (Config, error) {
	var config Config

	// read config
	f, openErr := os.Open("config.yaml")
	if openErr != nil {
		return config, openErr
	}

	// yaml decode
	decodeErr := yaml.NewDecoder(f).Decode(&config)
	if decodeErr != nil {
		return config, decodeErr
	}

	// verify port is valid
	if config.Port < 0 || config.Port > 65535 {
		return config, fmt.Errorf("Invalid port number '%d' in config file.", config.Port)
	}

	return config, nil
}

func CastProperties[PropsT any](props interface{}, out *PropsT) error {
	marshalled, err := yaml.Marshal(props)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(marshalled, out)
	return err
}
