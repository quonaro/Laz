// Package config resolves laz settings: server address and auth token.
// Resolution order: CLI flag > environment variable > config file.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the frps connection settings.
type Config struct {
	// Server is the frps control address, host or host:port
	// (e.g. "tunnel.example.com" or "tunnel.example.com:7000").
	Server string `yaml:"server"`
	// Token is the frps auth token (auth.method = "token").
	Token string `yaml:"token"`
	// Transport overrides transport.protocol (websocket, tcp, quic, kcp).
	Transport string `yaml:"transport"`
	// DefaultServerPort is assumed when Server has no port.
	DefaultServerPort int `yaml:"server_port"`
}

// Load merges the optional flag overrides with the environment and the
// config file (~/.config/laz/config.yaml, then ./laz.yaml).
func Load(flagServer, flagToken, flagTransport string) (*Config, error) {
	cfg := &Config{
		Server:    os.Getenv("LAZ_SERVER"),
		Token:     os.Getenv("LAZ_TOKEN"),
		Transport: os.Getenv("LAZ_TRANSPORT"),
	}
	if fileCfg, err := loadFile(); err != nil {
		return nil, err
	} else if fileCfg != nil {
		cfg.merge(fileCfg)
	}
	if flagServer != "" {
		cfg.Server = flagServer
	}
	if flagToken != "" {
		cfg.Token = flagToken
	}
	if flagTransport != "" {
		cfg.Transport = flagTransport
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("no server configured: pass --server, set LAZ_SERVER, or write ~/.config/laz/config.yaml")
	}
	if cfg.DefaultServerPort == 0 {
		cfg.DefaultServerPort = 7000
	}
	return cfg, nil
}

// loadFile reads the first config file that exists, in priority order.
func loadFile() (*Config, error) {
	candidates := []string{"laz.yaml", "laz.yml"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".config", "laz", "config.yaml"))
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		return &cfg, nil
	}
	return nil, nil
}

// merge fills empty fields of cfg from other.
func (c *Config) merge(other *Config) {
	if c.Server == "" {
		c.Server = other.Server
	}
	if c.Token == "" {
		c.Token = other.Token
	}
	if c.Transport == "" {
		c.Transport = other.Transport
	}
	if c.DefaultServerPort == 0 {
		c.DefaultServerPort = other.DefaultServerPort
	}
}
