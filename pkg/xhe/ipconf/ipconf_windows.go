package ipconf

import (
	"errors"
	"net"
	"net/netip"
	"sync"

	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

var once = &sync.Once{}
var luid winipcfg.LUID

func addRoute(dev tun.Device, ip netip.Prefix) (ierr error) {
	if err := addRouteToStack(dev, ip); !errors.Is(err, ErrNotGVisorStack) {
		return err
	}

	name, err := dev.Name()
	if err != nil {
		return
	}
	once.Do(func() {
		var iface any
		iface, ierr = net.InterfaceByName(name)
		luid, ierr = winipcfg.LUIDFromIndex(uint32(iface.Index))
	})
	luid.AddIPAddress(ip)
	return
}

func up(dev tun.Device) error {
	return nil
}
