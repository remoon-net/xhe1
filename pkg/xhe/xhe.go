package xhe

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	"github.com/lainio/err2/try"
	"golang.org/x/sync/errgroup"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"remoon.net/xhe/pkg/config"
	"remoon.net/xhe/pkg/signaler"
	"remoon.net/xhe/pkg/xhe/ipconf"
)

func Run(cfg Config) (dev *device.Device, ierr error) {
	cfg.Normalize()

	key, ierr := str2pubkey(cfg.PrivateKey)
	if ierr != nil {
		return
	}
	server := signaler.New(key, cfg.Links)
	bind := newBind(server)
	logger := device.NewLogger(
		toDeviceLogLv(cfg.LogLevel),
		fmt.Sprintf("(%s) ", try.To1(cfg.GoTun.Name())),
	)
	dev = device.NewDevice(cfg.GoTun, bind, logger)
	bind.init(dev)

	ierr = func() (ierr error) { // 设置 WireGuard
		logger := slog.With(slog.String("act", "configure WireGuard"))
		logger.Debug("pending")
		defer then(&ierr, func() {
			logger.Debug("successful")
		}, nil)

		conf := config.Device{
			PrivateKey: hex.EncodeToString(key),
			ListenPort: cfg.Port,
		}
		ierr = dev.IpcSet(conf.String())
		if ierr != nil {
			return
		}
		return
	}()
	if ierr != nil {
		return
	}

	ierr = func() (ierr error) { //设置 Peers
		logger := slog.With(slog.String("act", "Peers"))
		logger.Debug("parse")
		count := 0
		defer then(&ierr, func() {
			logger.Debug("parse successful", "count", count)
		}, nil)
		conf := ""
		s := &DoH{Server: cfg.DoH}
		eg := new(errgroup.Group)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		for _, _p := range cfg.Peers {
			p := _p
			eg.Go(func() (ierr error) {
				peer, ierr := s.ParsePeer(ctx, p)
				if ierr != nil {
					return
				}
				conf += peer.String()
				count++
				return
			})
		}
		ierr = eg.Wait()
		if ierr != nil {
			return
		}

		logger.Debug("add to WireGuard")
		defer then(&ierr, func() {
			logger.Debug("add to WireGuard successful")
		}, nil)
		ierr = dev.IpcSet(conf)
		if ierr != nil {
			return
		}

		return
	}()
	if ierr != nil {
		return
	}

	ierr = func() (ierr error) { // 启动 WireGuard
		logger := slog.With(slog.String("act", "WireGuard start"))
		logger.Debug("pending")
		defer then(&ierr, func() {
			logger.Info("successful")
		}, nil)

		ierr = dev.Up()
		if ierr != nil {
			return
		}
		return
	}()
	if ierr != nil {
		return
	}

	pubkey := wgtypes.Key(key).PublicKey()
	pf, ierr := GetIP(pubkey[:])
	if ierr != nil {
		return
	}
	pf = netip.PrefixFrom(pf.Addr(), 24)
	ierr = ipconf.AddRoute(cfg.GoTun, pf)
	if ierr != nil {
		return
	}
	ierr = ipconf.Up(cfg.GoTun)
	if ierr != nil {
		return
	}

	return
}
