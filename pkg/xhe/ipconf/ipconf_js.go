package ipconf

import (
	"net/netip"

	"golang.zx2c4.com/wireguard/tun"
)

func addRoute(dev tun.Device, ip netip.Prefix) error {
	return addRouteToStack(dev, ip)
}

func up(dev tun.Device) error {
	return nil
}
