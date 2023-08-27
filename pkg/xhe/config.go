package xhe

import (
	"log/slog"

	"golang.zx2c4.com/wireguard/tun"
)

type Config struct {
	LogLevel   slog.Level `json:"log_level"`
	PrivateKey string     `json:"private_key"`
	DoH        string     `json:"doh"`
	Links      []string   `json:"links"`
	Peers      []string   `json:"peers"`
	Port       uint16     `json:"port"`
	MTU        int        `json:"mtu"`
	GoTun      tun.Device
}

func (cfg Config) Normalize() {
	if cfg.MTU == 0 {
		cfg.MTU = 2400 - 80
	}
	if cfg.DoH == "" {
		cfg.DoH = "1.1.1.1"
	}
}
