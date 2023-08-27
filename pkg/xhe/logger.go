package xhe

import (
	"log/slog"

	"golang.zx2c4.com/wireguard/device"
)

func toDeviceLogLv(lv slog.Level) int {
	switch lv {
	case slog.LevelDebug, slog.LevelWarn:
		return device.LogLevelVerbose
	case slog.LevelInfo:
		return device.LogLevelSilent
	case slog.LevelError:
		return device.LogLevelError
	}
	return device.LogLevelError
}
