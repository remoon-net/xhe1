//go:build ierr

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

	key, ierr := hex.DecodeString(cfg.PrivateKey)
	server := signaler.New(key, cfg.Links)
	bind := newBind(server)
	logger := device.NewLogger(
		toDeviceLogLv(cfg.LogLevel),
		fmt.Sprintf("(%s) ", try.To1(cfg.GoTun.Name())),
	)
	dev = device.NewDevice(cfg.GoTun, bind, logger)
	bind.init(dev)

	ierr = func() (ierr error) { // 设置 WireGuard
		logger := slog.With(slog.String("act", "配置 wg"))
		logger.Debug("进行中")
		defer then(&ierr, func() {
			logger.Debug("完成")
		}, nil)

		conf := config.Device{
			PrivateKey: hex.EncodeToString(key),
			ListenPort: cfg.Port,
		}
		ierr = dev.IpcSet(conf.String())
		return
	}()

	ierr = func() (ierr error) { //设置 Peers
		logger := slog.With(slog.String("act", "Peers"))
		logger.Debug("解析中")
		count := 0
		defer then(&ierr, func() {
			logger.Debug("解析完成", "count", count)
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
				conf += peer.String()
				count++
				return
			})
		}
		ierr = eg.Wait()

		logger.Debug("应用中")
		defer then(&ierr, func() {
			logger.Debug("应用完成")
		}, nil)
		ierr = dev.IpcSet(conf)

		return
	}()

	ierr = func() (ierr error) { // 启动 WireGuard
		logger := slog.With(slog.String("act", "WireGuard 启动"))
		logger.Debug("进行中")
		defer then(&ierr, func() {
			logger.Info("成功")
		}, nil)

		ierr = dev.Up()
		return
	}()

	pubkey := wgtypes.Key(key).PublicKey()
	pf, ierr := GetIP(pubkey[:])
	pf = netip.PrefixFrom(pf.Addr(), 24)
	ierr = ipconf.AddRoute(cfg.GoTun, pf)
	ierr = ipconf.Up(cfg.GoTun)

	return
}
