package main

import (
	"os"

	"remoon.net/xhe/cmd"
)

var Version = "dev"

func main() {
	cmd.Execute(Version)
}

func init() {
	return
	os.Args = append(os.Args,
		"--vtun",
		"--log", "debug",
		"-k", "SA7wvbecJtRXtb9ATH9h7Vu+GLq4qoOVPg/SrxIGP0w=",
		"-l", "https://xhe.remoon.net",
		"-p", "peer://8066d0db32b6dda61541d4513a431504599cb296b250f0b6855c7c30bcaab8.62.xhe.remoon.net",
	)
}
