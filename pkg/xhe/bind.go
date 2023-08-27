//go:build ierr

package xhe

import (
	"encoding/hex"
	"net/url"

	"github.com/shynome/wgortc"
	"github.com/shynome/wgortc/endpoint"
	"golang.org/x/exp/slog"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"remoon.net/xhe/pkg/signaler"
)

type Bind struct {
	conn.Bind
	dev *device.Device
	m   map[string]bool
}

// 包一层实现快速重连
func newBind(server *signaler.Signaler) *Bind {
	bind := wgortc.NewBind(server)
	return &Bind{
		Bind: bind,
		m:    make(map[string]bool),
	}
}

func (b *Bind) init(dev *device.Device) {
	b.dev = dev
}

func (b *Bind) Send(bufs [][]byte, ep conn.Endpoint) error {
	err := b.Bind.Send(bufs, ep)
	go b.check(ep, err)
	return err
}

func (b *Bind) check(_ep conn.Endpoint, err error) {
	if b.dev == nil {
		return
	}
	ep, ok := _ep.(*endpoint.Outbound)
	if !ok {
		return
	}

	id := string(ep.DstToBytes())

	if err == nil {
		if _, ok := b.m[id]; !ok {
			return
		}
		b.m[id] = false
		return
	}

	if removed, ok := b.m[id]; ok && removed {
		return
	}
	b.m[id] = true

	var ierr error
	_ = ierr
	slog.With("connect", id).Warn("连接已关闭, 需要重新握手")
	link, ierr := url.Parse(id)
	pubkey, ierr := hex.DecodeString(link.Fragment)
	pk := device.NoisePublicKey(pubkey)
	peer := b.dev.LookupPeer(pk)
	peer.ExpireCurrentKeypairs()
}
