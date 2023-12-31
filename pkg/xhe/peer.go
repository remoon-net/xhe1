package xhe

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
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

// ParsePeer
// peer://{domain.com}[/preshared_key]?[keepalive=15]
// peer://{pubkey}[/preshared_key]?[keepalive=15]
// http[s]://domain/path?peer={pubkey}[&preshared=preshared_key][&keepalive=15]
func (s *DoH) ParsePeer(ctx context.Context, link string) (peer config.Peer, ierr error) {
	conn := doh.NewConn(s.Client, ctx, s.Server)
	u, ierr := url.Parse(link)
	if ierr != nil {
		return
	}
	var endpoint string
	var pubkey []byte
	preshared := strings.TrimPrefix(u.Path, "/")
	switch u.Scheme {
	case "peer":
		if strings.Index(u.Hostname(), ".") == -1 {
			pubkey, ierr = hex2pubkey(u.Hostname())
			if ierr != nil {
				return
			}
		} else {
			endpoint, ierr = GetURI(conn, u.Hostname())
			if ierr != nil {
				return
			}
			var uu *url.URL
			uu, ierr = url.Parse(endpoint)
			if ierr != nil {
				return
			}
			pubkey, ierr = hex2pubkey(uu.Query().Get("peer"))
			if ierr != nil {
				return
			}
		}
	case "http", "https":
		q := u.Query()
		pubkey, ierr = hex2pubkey(q.Get("peer"))
		endpoint = link
		preshared = q.Get("preshared")
	default:
		ierr = fmt.Errorf("unsupport schema %s", u.Scheme)
	}
	if preshared != "" {
		_, ierr = hex2pubkey(preshared)
		if ierr != nil {
			return
		}
	}

	ip, ierr := GetIP(pubkey)
	if ierr != nil {
		return
	}
	if endpoint != "" {
		var u *url.URL
		u, ierr = url.Parse(endpoint)
		if ierr != nil {
			return
		}
		u.Fragment = hex.EncodeToString(pubkey)
		endpoint = u.String()
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
	if ierr != nil {
		return
	}
	_, ierr = hasher.Write(pubkey)
	if ierr != nil {
		return
	}
	pf, ierr = netip.ParsePrefix(Subnet)
	if ierr != nil {
		return
	}
	addr := pf.Addr().As16()
	_, ierr = io.ReadFull(hasher, addr[4:])
	if ierr != nil {
		return
	}
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
	if ierr != nil {
		return
	}
	r, ierr := co.ReadMsg()
	if ierr != nil {
		return
	}
	for _, a := range r.Answer {
		switch v := a.(type) {
		case *dns.URI:
			return v.Target, nil
		}
	}
	return
}

func hex2pubkey(pubkey string) (b []byte, ierr error) {
	b, ierr = hex.DecodeString(pubkey)
	if ierr != nil {
		return
	}
	if len(b) != 32 {
		return nil, ErrNotWireGuardPubkey
	}
	return
}

func str2pubkey(pubkey string) (b []byte, ierr error) {
	if len(pubkey) == 64 {
		return hex2pubkey(pubkey)
	}
	b, ierr = base64.StdEncoding.DecodeString(pubkey)
	if ierr != nil {
		return
	}
	if len(b) != 32 {
		return nil, ErrNotWireGuardPubkey
	}
	return
}

var ErrNoCnamePubkey = errors.New("find 0 cname pubkey")
var ErrNotWireGuardPubkey = errors.New("not wireguard pubkey")
