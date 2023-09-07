package ipconf

import (
	"net/netip"

	"golang.org/x/exp/slog"
	"golang.zx2c4.com/wireguard/tun"
	"remoon.net/xhe/pkg/vtun"
)

func AddRoute(dev tun.Device, ip netip.Prefix) (err error) {
	logger := slog.With(
		slog.String("act", "add ip route"),
		slog.String("ip", ip.String()),
	)
	logger.Debug("pending")
	defer then(&err, func() {
		logger.Debug("successful")
	}, nil)

	if _, ok := dev.(vtun.GetStack); ok {
		logger.Debug("vtun mode")
		return addRouteToStack(dev, ip)
	}
	return addRoute(dev, ip)
}

func Up(dev tun.Device) (err error) {
	logger := slog.With(
		slog.String("act", "device tun up"),
	)
	logger.Debug("pending")
	defer then(&err, func() {
		logger.Debug("successful")
	}, nil)

	if _, ok := dev.(vtun.GetStack); ok {
		logger.Debug("vtun mode")
		return nil
	}
	return up(dev)
}
