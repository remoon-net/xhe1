package cmd

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	"remoon.net/xhe/pkg/xhe"
)

// ipCmd represents the ip command
var ipCmd = &cobra.Command{
	Use:   "ip {pubkey}",
	Short: "get pubkey ip",
	Long:  `get pubkey ip`,
	Run: func(cmd *cobra.Command, args []string) {
		var ierr error
		defer then(&ierr, nil, func() {
			slog.Error("get ip failed", "err", ierr)
		})
		if len(args) == 0 {
			ierr = fmt.Errorf("pubkey is required")
			if ierr != nil {
				return
			}
		}
		pubkeyStr := args[0]
		var pubkey []byte
		if len(pubkeyStr) == 64 {
			pubkey, ierr = hex.DecodeString(pubkeyStr)
			if ierr != nil {
				return
			}
		} else {
			pubkey, ierr = base64.StdEncoding.DecodeString(pubkeyStr)
			if ierr != nil {
				return
			}
		}
		if len(pubkey) != 32 {
			ierr = xhe.ErrNotWireGuardPubkey
			if ierr != nil {
				return
			}
		}
		pf, ierr := xhe.GetIP(pubkey)
		if ierr != nil {
			return
		}
		fmt.Println(pf.Addr().String())
	},
}

func init() {
	rootCmd.AddCommand(ipCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ipCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ipCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
