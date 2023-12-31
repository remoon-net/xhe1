package xhe

import (
	"encoding/base64"
	"testing"

	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
	"github.com/shynome/doh-client"
)

var (
	key1, _ = base64.StdEncoding.DecodeString("SA7wvbecJtRXtb9ATH9h7Vu+GLq4qoOVPg/SrxIGP0w=")
	key2, _ = base64.StdEncoding.DecodeString("oKL7+pbuh/kJvD1pleelYM5r/F5i/G5iCZ7fNqPT8lU=")
)

func TestGetIP(t *testing.T) {
	pubkey := try.To1(base64.StdEncoding.DecodeString("yDEt6rccWlIfDTUTxUCDd7O5DjiONNwIonvcn94UDlI="))
	ip := try.To1(GetIP(pubkey))
	ipStr := ip.String()
	assert.Equal(ipStr, "fdd9:f800:b4e8:cb59:95e3:c464:9fff:b8c8/128")
	t.Log(ip)
}

func TestGetEndpoint(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		conn := doh.NewConn(nil, nil, "1.1.1.1")
		endpoint := try.To1(GetURI(conn, "test-xhe.remoon.net"))
		assert.Equal(endpoint, "https://xhe.remoon.net?peer=81dea2c5c077bf78b34a518eda9851cfbe718656fdc470970bde057cbceef23e")
		t.Log(endpoint)
	})
	t.Run("not exists", func(t *testing.T) {
		conn := doh.NewConn(nil, nil, "1.1.1.1")
		endpoint := try.To1(GetURI(conn, "test2-xhe.remoon.net"))
		assert.Equal(endpoint, "")
		t.Log(endpoint)
	})
}
