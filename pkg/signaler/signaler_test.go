package signaler

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
	"github.com/pion/webrtc/v3"
	"github.com/shynome/wgortc/signaler"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestSubscribe(t *testing.T) {
	key1 := try.To1(wgtypes.GeneratePrivateKey())
	key2 := try.To1(wgtypes.GeneratePrivateKey())
	s1 := New(key1[:], []string{"https://xhe.remoon.net"})
	defer s1.Close()
	s2 := New(key2[:], []string{})

	ctx, cancel := context.WithCancelCause(context.Background())
	go func() {
		ch, err := s1.Accept()
		cancel(err)
		if err != nil {
			return
		}
		for s := range ch {
			offer := s.Description()
			assert.Equal(offer.Type, webrtc.SDPTypeOffer)
			s.Resolve(&signaler.SDP{Type: webrtc.SDPTypeAnswer})
		}
	}()
	<-ctx.Done()
	try.Is(context.Cause(ctx), context.Canceled)

	offer := signaler.SDP{Type: webrtc.SDPTypeOffer}
	peer := key1.PublicKey()
	answer := try.To1(s2.Handshake("https://xhe.remoon.net?peer="+hex.EncodeToString(peer[:]), offer))
	assert.Equal(answer.Type, webrtc.SDPTypeAnswer)
}
