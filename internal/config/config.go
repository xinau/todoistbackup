package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing configuration file %s: %w", filename, err)
	}

	return &config, nil
}

type Config struct {
	Client *ClientConfig
	Store  *StoreConfig
}

type ClientConfig struct {
	Token   string `json:"token"`
	Timeout int    `json:"timeout"`
}

func (c *ClientConfig) UnmarshalJSON(b []byte) error {
	cc := &ClientConfig{}
	type plain ClientConfig
	if err := json.Unmarshal(b, (*plain)(cc)); err != nil {
		return err
	}

	if cc.Timeout == 0 {
		cc.Timeout = 5
	}

	*c = *cc
	return nil
}

type StoreConfig struct {
	Bucket   string `json:"bucket"`
	Endpoint string `json:"endpoint"`
	Region   string `json:"region"`

	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`

	Insecure bool `json:"insecure"`
}
