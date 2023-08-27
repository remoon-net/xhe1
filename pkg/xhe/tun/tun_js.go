package tun

import (
	"golang.zx2c4.com/wireguard/tun"
	"remoon.net/xhe/pkg/vtun"
)

func CreateTUN(name string, mtu int) (tdev tun.Device, err error) {
	return vtun.CreateTUN(name, mtu)
}
