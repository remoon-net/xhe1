package vtun

import (
	"context"
	"net"
	"net/netip"

	"github.com/armon/go-socks5"
	"github.com/lainio/err2/try"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
)

func NewSocks5Server(vtun GetStack) *socks5.Server {
	s := vtun.GetStack()
	nic := vtun.NIC()
	conf := socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			ap, err := netip.ParseAddrPort(addr)
			if err != nil {
				return nil, err
			}
			fa, pn := convertToFullAddr(nic, ap)
			return gonet.DialContextTCP(ctx, s, fa, pn)
		},
	}
	return try.To1(socks5.New(&conf))
}

func convertToFullAddr(NICID tcpip.NICID, endpoint netip.AddrPort) (tcpip.FullAddress, tcpip.NetworkProtocolNumber) {
	var protoNumber tcpip.NetworkProtocolNumber
	if endpoint.Addr().Is4() {
		protoNumber = ipv4.ProtocolNumber
	} else {
		protoNumber = ipv6.ProtocolNumber
	}
	return tcpip.FullAddress{
		NIC:  NICID,
		Addr: tcpip.Address(endpoint.Addr().AsSlice()),
		Port: endpoint.Port(),
	}, protoNumber
}
