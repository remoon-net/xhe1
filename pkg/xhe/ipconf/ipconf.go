package ipconf

import (
	"net/netip"

	"golang.org/x/exp/slog"
	"golang.zx2c4.com/wireguard/tun"
	"remoon.net/xhe/pkg/vtun"
)

func AddRoute(dev tun.Device, ip netip.Prefix) (err error) {
	logger := slog.With(
		slog.String("act", "添加路由"),
		slog.String("ip", ip.String()),
	)
	logger.Debug("进行中")
	defer then(&err, func() {
		logger.Debug("成功")
	}, nil)

	if _, ok := dev.(vtun.GetStack); ok {
		logger.Debug("vtun模式")
		return addRouteToStack(dev, ip)
	}
	return addRoute(dev, ip)
}

func Up(dev tun.Device) (err error) {
	logger := slog.With(
		slog.String("act", "网卡设备启动"),
	)
	logger.Debug("进行中")
	defer then(&err, func() {
		logger.Debug("成功")
	}, nil)

	if _, ok := dev.(vtun.GetStack); ok {
		logger.Debug("vtun模式")
		return nil
	}
	return up(dev)
}
