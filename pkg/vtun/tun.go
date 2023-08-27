package vtun

import (
	"fmt"
	"os"
	"sync/atomic"
	"syscall"

	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/bufferv2"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

type netTun struct {
	ep             *channel.Endpoint
	stack          *stack.Stack
	events         chan tun.Event
	incomingPacket chan *bufferv2.View

	name string
	mtu  int
	nic  tcpip.NICID
}

var globalNIC int32 = 0

func CreateTUN(name string, mtu int) (tdev *netTun, err error) {
	nic := tcpip.NICID(atomic.AddInt32(&globalNIC, 1))

	opts := stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol, icmp.NewProtocol4, icmp.NewProtocol6},
	}
	dev := &netTun{
		ep:             channel.New(1024, uint32(mtu), ""),
		stack:          stack.New(opts),
		events:         make(chan tun.Event, 10),
		incomingPacket: make(chan *bufferv2.View),

		name: name,
		mtu:  mtu,
		nic:  nic,
	}

	sackEnabledOpt := tcpip.TCPSACKEnabled(true)
	if tcpipErr := dev.stack.SetTransportProtocolOption(tcp.ProtocolNumber, &sackEnabledOpt); tcpipErr != nil {
		return nil, fmt.Errorf("could not enable TCP SACK: %v", tcpipErr)
	}
	dev.ep.AddNotify(dev)
	if tcpipErr := dev.stack.CreateNIC(dev.nic, dev.ep); tcpipErr != nil {
		return nil, fmt.Errorf("CreateNIC: %v", tcpipErr)
	}
	dev.stack.AddRoute(tcpip.Route{Destination: header.IPv4EmptySubnet, NIC: dev.nic})
	dev.stack.AddRoute(tcpip.Route{Destination: header.IPv6EmptySubnet, NIC: dev.nic})
	dev.events <- tun.EventUp
	return dev, nil
}

type GetStack interface {
	GetStack() *stack.Stack
	NIC() tcpip.NICID
}

var _ GetStack = (*netTun)(nil)

func (tun *netTun) GetStack() *stack.Stack { return tun.stack }
func (tun *netTun) NIC() tcpip.NICID       { return tun.nic }

var _ channel.Notification = (*netTun)(nil)
var _ tun.Device = (*netTun)(nil)

// the below code copy form golang.zx2c4.com/wireguard/tun/netstack/tun.go

func (tun *netTun) Name() (string, error) {
	return tun.name, nil
}

func (tun *netTun) File() *os.File {
	return nil
}

func (tun *netTun) Events() <-chan tun.Event {
	return tun.events
}

func (tun *netTun) Read(buf [][]byte, sizes []int, offset int) (int, error) {
	view, ok := <-tun.incomingPacket
	if !ok {
		return 0, os.ErrClosed
	}

	n, err := view.Read(buf[0][offset:])
	if err != nil {
		return 0, err
	}
	sizes[0] = n
	return 1, nil
}

func (tun *netTun) Write(buf [][]byte, offset int) (int, error) {
	for _, buf := range buf {
		packet := buf[offset:]
		if len(packet) == 0 {
			continue
		}

		pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: bufferv2.MakeWithData(packet)})
		switch packet[0] >> 4 {
		case 4:
			tun.ep.InjectInbound(header.IPv4ProtocolNumber, pkb)
		case 6:
			tun.ep.InjectInbound(header.IPv6ProtocolNumber, pkb)
		default:
			return 0, syscall.EAFNOSUPPORT
		}
	}
	return len(buf), nil
}

func (tun *netTun) WriteNotify() {
	pkt := tun.ep.Read()
	if pkt.IsNil() {
		return
	}

	view := pkt.ToView()
	pkt.DecRef()

	tun.incomingPacket <- view
}

func (tun *netTun) Close() error {
	tun.stack.RemoveNIC(tun.nic)

	if tun.events != nil {
		close(tun.events)
	}

	tun.ep.Close()

	if tun.incomingPacket != nil {
		close(tun.incomingPacket)
	}

	return nil
}

func (tun *netTun) MTU() (int, error) {
	return tun.mtu, nil
}

func (tun *netTun) BatchSize() int {
	return 1
}
