package ipconf

import (
	"net/netip"
	"os"
	"os/exec"

	"golang.zx2c4.com/wireguard/tun"
)

func addRoute(dev tun.Device, ip netip.Prefix) error {
	name, err := dev.Name()
	if err != nil {
		return err
	}
	cmd := exec.Command("ip", "addr", "add", ip.String(), "dev", name)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func up(dev tun.Device) error {
	name, err := dev.Name()
	if err != nil {
		return err
	}
	cmd := exec.Command("ip", "link", "set", name, "up")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
