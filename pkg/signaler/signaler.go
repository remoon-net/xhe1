package signaler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/r3labs/sse/v2"
	"github.com/shynome/go-x25519"
	"github.com/shynome/wgortc/signaler"
	"golang.org/x/sync/errgroup"
)

type Signaler struct {
	Key     x25519.PrivateKey
	servers []string
	Client  *http.Client

	cancel context.CancelCauseFunc
}

var _ signaler.Channel = (*Signaler)(nil)

func New(key x25519.PrivateKey, servers []string) *Signaler {
	return &Signaler{
		Key:     key,
		servers: servers,
		Client:  http.DefaultClient,
	}
}

func (s *Signaler) Handshake(endpoint string, offer signaler.SDP) (answer *signaler.SDP, ierr error) {
	logger := slog.With(
		"act", "handshake",
		"endpoint", endpoint,
	)
	logger.Debug("pending")
	defer then(&ierr, func() {
		logger.Debug("successful")
	}, func() {
		logger.Warn("failed", "err", ierr)
	})

	b, ierr := json.Marshal(offer)
	if ierr != nil {
		return
	}
	body := bytes.NewBuffer(b)
	link, ierr := SignURL(endpoint, s.Key)
	if ierr != nil {
		return
	}
	req, ierr := http.NewRequest(http.MethodPost, link.String(), body)
	if ierr != nil {
		return
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	resp, ierr := s.Client.Do(req)
	if ierr != nil {
		return
	}
	ierr = checkResponse(resp)
	if ierr != nil {
		return
	}
	answer = new(signaler.SDP)
	ierr = json.NewDecoder(resp.Body).Decode(answer)
	if ierr != nil {
		return
	}
	return
}
func (s *Signaler) Accept() (offerCh <-chan signaler.Session, ierr error) {
	ctx := context.Background()
	ctx, s.cancel = context.WithCancelCause(ctx)
	if len(s.servers) == 0 {
		ch := make(chan signaler.Session)
		close(ch)
		return ch, nil
	}
	ch, ierr := s.subscribeAll(ctx)
	if ierr != nil {
		return
	}
	return ch, nil
}
func (s *Signaler) Close() error {
	if s.cancel != nil {
		s.cancel(context.Canceled)
	}
	return nil
}

func (s *Signaler) subscribeAll(ctx context.Context) (ch chan signaler.Session, ierr error) {
	ch = make(chan signaler.Session, 512)
	g := new(errgroup.Group)
	for _, _server := range s.servers {
		server := _server
		g.Go(func() (err error) {
			return s.subscribe(ctx, ch, server)
		})
	}
	ierr = g.Wait()
	if ierr != nil {
		return
	}
	return
}

func (s *Signaler) subscribe(ctx context.Context, ch chan<- signaler.Session, server string) (ierr error) {
	logger := slog.With(
		"act", "subscribe",
		"server", server,
	)
	logger.Debug("start")
	defer then(&ierr, func() {
		logger.Debug("successful")
	}, func() {
		logger.Warn("failed", "err", ierr)
	})

	u, ierr := SignURL(server, s.Key)
	if ierr != nil {
		return
	}
	c := sse.NewClient(u.String(), func(c *sse.Client) {
		c.Connection = s.Client
		c.ReconnectStrategy = NewReconnectStrategy(ctx, time.Second)
		c.ReconnectNotify = func(err error, d time.Duration) {
			u, err := SignURL(server, s.Key)
			if err != nil {
				panic(err)
			}
			c.URL = u.String()
		}
	})
	// make sure the first connect is fine
	var errch = make(chan error)
	defer close(errch)
	c.ResponseValidator = func(c *sse.Client, resp *http.Response) (err error) {
		defer func() {
			if resp.StatusCode == http.StatusLocked {
				logger.Warn("signaler server is locked. continue try")
				return
			}
			errch <- err
		}()
		if resp.StatusCode == 200 {
			logger.Debug("subscribed")
			c.ResponseValidator = nil
			return nil
		}
		resp.Body.Close()
		err = fmt.Errorf("could not connect to stream: %s", http.StatusText(resp.StatusCode))
		return err
	}
	go func() {
		for {
			err := c.SubscribeRawWithContext(ctx, func(msg *sse.Event) {
				if len(msg.Data) == 0 {
					return
				}
				go func() (ierr error) {
					logger.Debug("connect in")
					defer then(&ierr, nil, func() {
						logger.Error("sse wrong", ierr, string(msg.Data))
					})
					var sdp signaler.SDP
					ierr = json.Unmarshal(msg.Data, &sdp)
					if ierr != nil {
						return
					}
					sess := &Session{
						ctx:  ctx,
						root: s,
						link: server,
						id:   string(msg.ID),
						sdp:  sdp,
					}
					ch <- sess
					return
				}()
			})
			// err == nil 也继续重试, 只有当手动取消时才会退出
			if errors.Is(err, context.Canceled) {
				return
			}
			time.Sleep(time.Second)
		}
	}()
	return <-errch
}

type Session struct {
	root *Signaler
	ctx  context.Context
	link string
	id   string
	sdp  signaler.SDP
}

var _ signaler.Session = (*Session)(nil)

func (s *Session) Description() (offer signaler.SDP) { return s.sdp }
func (s *Session) Resolve(answer *signaler.SDP) (ierr error) {
	logger := slog.With(
		"act", "accept handshake",
		"id", s.id,
	)
	logger.Debug("start")
	defer then(&ierr, func() {
		logger.Debug("successful")
	}, func() {
		logger.Warn("failed", "err", ierr)
	})

	body, ierr := json.Marshal(answer)
	if ierr != nil {
		return
	}

	root := s.root
	link, ierr := SignURL(s.link, root.Key)
	if ierr != nil {
		return
	}
	req, ierr := http.NewRequest(http.MethodDelete, link.String(), bytes.NewBuffer(body))
	if ierr != nil {
		return
	}
	req.Header.Set("X-Event-Id", s.id)
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	resp, ierr := root.Client.Do(req)
	if ierr != nil {
		return
	}
	ierr = checkResponse(resp)
	if ierr != nil {
		return
	}

	return
}
func (s *Session) Reject(err error) { return }

func checkResponse(r *http.Response) error {
	if !strings.HasPrefix(r.Status, "2") {
		return fmt.Errorf("status code is %s. link: %s", r.Status, r.Request.URL)
	}
	return nil
}
