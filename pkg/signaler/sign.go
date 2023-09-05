package signaler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/shynome/go-x25519"
)

func SignURL(link string, privkey x25519.PrivateKey) (u *url.URL, ierr error) {
	u, ierr = url.Parse(link)
	if ierr != nil {
		return
	}
	pubkey, _ := privkey.PublicKey()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature, ierr := x25519.Sign(rand.Reader, privkey, []byte(timestamp))
	if ierr != nil {
		return
	}
	q := u.Query()
	q.Set("pubkey", hex.EncodeToString(pubkey))
	q.Set("timestamp", timestamp)
	q.Set("signature", hex.EncodeToString(signature))
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return
}
