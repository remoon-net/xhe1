//go:build !js

package tun

import "golang.zx2c4.com/wireguard/tun"

func CreateTUN(name string, mtu int) (tun.Device, error) {
	return tun.CreateTUN(name, mtu)
}
