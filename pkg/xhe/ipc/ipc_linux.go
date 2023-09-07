package ipc

import (
	"net"

	"golang.zx2c4.com/wireguard/ipc"
)

func UAPIListen(name string) (uapi net.Listener, ierr error) {
	fileUAPI, ierr := ipc.UAPIOpen(name)
	if ierr != nil {
		return
	}
	uapi, ierr = ipc.UAPIListen(name, fileUAPI)
	if ierr != nil {
		return
	}
	return
}
