//go:build ierr

package xhe

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/miekg/dns"
	"github.com/shynome/doh-client"
	"golang.org/x/crypto/blake2s"
	"remoon.net/xhe/pkg/config"
)

type DoH struct {
	Server string
	Client *http.Client
}

// ParsePeer peer://{pubkey}.xhe.remoon.net[/preshared_key]?keepalive=15
func (s *DoH) ParsePeer(ctx context.Context, link string) (peer config.Peer, ierr error) {
	conn := doh.NewConn(s.Client, ctx, s.Server)
	u, ierr := url.Parse(link)
	var endpoint string
	var pubkey []byte
	if strings.Index(u.Hostname(), ".") == -1 {
		pubkey, ierr = hex.DecodeString(u.Hostname())
	} else {
		endpoint, ierr = GetURI(conn, u.Hostname())
		pubkey, ierr = GetPubkey(conn, u.Hostname())
	}
	ip, ierr := GetIP(pubkey)
	if endpoint != "" {
		var u *url.URL
		u, ierr = url.Parse(endpoint)
		u.Fragment = hex.EncodeToString(pubkey)
		endpoint = u.String()
	}
	preshared := strings.TrimPrefix(u.Path, "/")
	if preshared != "" {
		_, ierr = hex2pubkey(preshared)
	}
	peer = config.Peer{
		PublicKey:    hex.EncodeToString(pubkey),
		AllowedIPs:   []string{ip.String()},
		PresharedKey: preshared,
		Endpoint:     endpoint,

		PersistentKeepalive: u.Query().Get("keepalive"),
	}
	return
}

const Subnet = "fdd9:f800::/24"

func GetIP(pubkey []byte) (pf netip.Prefix, ierr error) {
	hasher, ierr := blake2s.NewXOF(12, nil)
	_, ierr = hasher.Write(pubkey)
	pf, ierr = netip.ParsePrefix(Subnet)
	addr := pf.Addr().As16()
	_, ierr = io.ReadFull(hasher, addr[4:])
	pf = netip.PrefixFrom(netip.AddrFrom16(addr), 128)
	return
}

func GetURI(conn *doh.Conn, name string) (endpoint string, ierr error) {
	conn.Reset()
	q := dns.Question{
		Name:   dns.Fqdn(name),
		Qtype:  dns.TypeURI,
		Qclass: dns.ClassINET,
	}
	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			Opcode:           dns.OpcodeQuery,
			RecursionDesired: true,
		},
		Question: []dns.Question{q},
	}
	co := &dns.Conn{Conn: conn}
	ierr = co.WriteMsg(m)
	r, ierr := co.ReadMsg()
	for _, a := range r.Answer {
		switch v := a.(type) {
		case *dns.URI:
			return v.Target, nil
		}
	}
	return
}

const PubkeyDomainSuffix = ".xhe.remoon.net."

func GetPubkey(conn *doh.Conn, name string) (pubkey []byte, ierr error) {
	conn.Reset()
	name = dns.Fqdn(name)
	if strings.HasSuffix(name, PubkeyDomainSuffix) {
		return hex2pubkey(name)
	}
	q := dns.Question{
		Name:   name,
		Qtype:  dns.TypeCNAME,
		Qclass: dns.ClassINET,
	}
	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			Opcode:           dns.OpcodeQuery,
			RecursionDesired: true,
		},
		Question: []dns.Question{q},
	}
	co := &dns.Conn{Conn: conn}
	ierr = co.WriteMsg(m)
	r, ierr := co.ReadMsg()
	for _, a := range r.Answer {
		switch v := a.(type) {
		case *dns.CNAME:
			return hex2pubkey(v.Target)
		}
	}
	return nil, ErrNoCnamePubkey
}

func hex2pubkey(pubkey string) (b []byte, ierr error) {
	pubkey = strings.TrimSuffix(pubkey, PubkeyDomainSuffix)
	pubkey = strings.ReplaceAll(pubkey, ".", "")
	b, ierr = hex.DecodeString(pubkey)
	if len(b) != 32 {
		return nil, ErrNotWireGuardPubkey
	}
	return
}

var ErrNoCnamePubkey = errors.New("find 0 cname pubkey")
var ErrNotWireGuardPubkey = errors.New("not wireguard pubkey")
