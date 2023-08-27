//go:build ierr

package cmd

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"remoon.net/xhe/pkg/vtun"
	"remoon.net/xhe/pkg/xhe"
	"remoon.net/xhe/pkg/xhe/ipc"
	"remoon.net/xhe/pkg/xhe/tun"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "xhe -k {private_key}",
	Short: "WireGuard over WebRTC",
	Long:  `WireGuard over WebRTC`,
	Run: func(cmd *cobra.Command, args []string) {
		var ierr error
		defer then(&ierr, nil, func() {
			slog.Error("运行出错了", ierr)
			os.Exit(1)
		})

		key := viper.GetString("key")
		var privkey []byte
		if len(key) == 64 {
			privkey, ierr = hex.DecodeString(key)
		} else {
			privkey, ierr = base64.StdEncoding.DecodeString(key)
		}
		if len(privkey) != 32 {
			ierr = xhe.ErrNotWireGuardPubkey
			return
		}

		var logLevel slog.Level
		func() (ierr error) {
			lv := viper.GetString("log")
			b := strconv.AppendQuote(nil, lv)
			ierr = json.Unmarshal(b, &logLevel)
			h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
			slog.SetDefault(slog.New(h))
			return
		}()

		tunName := viper.GetString("tun")
		cfg := xhe.Config{
			PrivateKey: hex.EncodeToString(privkey),
			DoH:        viper.GetString("doh"),
			Port:       viper.GetUint16("port"),
			Links:      viper.GetStringSlice("link"),
			Peers:      viper.GetStringSlice("peer"),
			LogLevel:   logLevel,
			MTU:        viper.GetInt("mtu"),
		}
		log.Println(cfg.Links)

		vtunMode := viper.GetBool("vtun")
		if vtunMode {
			cfg.GoTun, ierr = vtun.CreateTUN(tunName, cfg.MTU)
		} else {
			cfg.GoTun, ierr = tun.CreateTUN(tunName, cfg.MTU)
		}

		dev, ierr := xhe.Run(cfg)
		defer dev.Close()

		errs := make(chan error)

		uapi, ierr := func() (uapi net.Listener, ierr error) {
			logger := slog.With(slog.String("act", "UAPI启动"))
			if vtunMode {
				logger.Warn("vtun mode 不支持 UAPI")
				return
			}
			logger.Debug("进行中")
			defer then(&ierr, func() {
				logger.Debug("完成")
			}, nil)

			uapi, ierr = ipc.UAPIListen(tunName)
			if uapi == nil {
				logger.
					With(slog.String("os", runtime.GOOS)).
					Warn("当前平台暂不支持 UAPI")
				return
			}
			go func() {
				for {
					conn, err := uapi.Accept()
					if err != nil {
						errs <- err
						return
					}
					go dev.IpcHandle(conn)
				}
			}()
			return
		}()
		if uapi != nil {
			defer uapi.Close()
		}

		l, ierr := func() (l net.Listener, ierr error) {
			addr := getSocksListenAddr(viper.GetString("export"))
			if addr == "" {
				return
			}
			logger := slog.With(slog.String("act", "启动socks5服务"))
			logger.Debug("进行中")
			defer then(&ierr, func() {
				logger.Info("成功")
			}, nil)

			tun, ok := cfg.GoTun.(vtun.GetStack)
			if !ok {
				return nil, fmt.Errorf("只有vtun模式下才可启动socks5服务")
			}
			s := vtun.NewSocks5Server(tun)
			l, ierr = net.Listen("tcp", addr)
			go func() {
				errs <- s.Serve(l)
			}()
			return
		}()
		if l != nil {
			defer l.Close()
		}

		term := make(chan os.Signal, 1)
		signal.Notify(term, os.Kill)
		signal.Notify(term, os.Interrupt)
		signal.Notify(term, syscall.SIGTERM)

		select {
		case <-term:
		case <-dev.Wait():
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(v string) {
	rootCmd.Version = v
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

const defaultMTU = 2*1200 - 80

func init() {
	cobra.OnInitialize(initConfig)

	f := rootCmd.Flags()

	f.StringP("key", "k", "", "the private key hex string. wg genkey")
	f.String("doh", "1.1.1.1", "doh server")
	f.StringSliceP("link", "l", []string{}, "监听的 event source 链接")
	f.StringSliceP("peer", "p", []string{}, "peers")
	f.StringSlice("ice", []string{}, "待实现. ice service.")
	f.Int("mtu", defaultMTU, "mtu")
	f.Uint16("port", 0, "listen port")
	f.String("log", "info", "日志等级. debug, info, warn, error")

	f.String("tun", "xhe", "tun name")
	f.Bool("vtun", false, "使用vtun模式, 该模式无需管理员权限即可运行")
	f.String("export", "", "使用vtun模式暴露一个socks5服务, 参数示例: 1080, 127.0.0.1:1080")

	viper.BindPFlags(f)
}

var cfgFile string

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find workdir directory.
		workdir, ierr := os.Getwd()
		if ierr != nil {
			panic(ierr)
		}

		// Search config in home directory with name ".xhe" (without extension).
		viper.AddConfigPath(workdir)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".xhe")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func getSocksListenAddr(s string) string {
	if s == "" {
		return ""
	}
	arr := strings.SplitN(s, ":", 2)
	if len(arr) == 1 {
		s = "127.0.0.1:" + s
	}
	return s
}