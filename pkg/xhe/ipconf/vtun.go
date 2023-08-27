package ipconf

import (
	"errors"
	"fmt"
	"net/netip"

	"golang.org/x/exp/slog"
	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"remoon.net/xhe/pkg/vtun"
)

type getStack = vtun.GetStack

var ErrNotGVisorStack = errors.New("the tun device can't get a gVisor stack")

func addRouteToStack(dev tun.Device, ip netip.Prefix) (err error) {
	slog.With(slog.String("ip", ip.String())).Debug("add route")
	tdev, ok := dev.(getStack)
	if !ok {
		return ErrNotGVisorStack
	}
	stk := tdev.GetStack()
	protoNumber := ipv6.ProtocolNumber
	if ip.Addr().Is4() {
		protoNumber = ipv4.ProtocolNumber
	}
	protoAddr := tcpip.ProtocolAddress{
		Protocol: protoNumber,
		AddressWithPrefix: tcpip.AddressWithPrefix{
			Address:   tcpip.Address(ip.Addr().AsSlice()),
			PrefixLen: ip.Bits(),
		},
	}
	tcpipErr := stk.AddProtocolAddress(tdev.NIC(), protoAddr, stack.AddressProperties{})
	if tcpipErr != nil {
		return fmt.Errorf("AddProtocolAddress(%v): %v", ip.String(), tcpipErr)
	}
	return nil
}
