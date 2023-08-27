package signaler

import (
	"context"
	"time"

	"gopkg.in/cenkalti/backoff.v1"
)

type Retry struct {
	ctx      context.Context
	duration time.Duration
}

var _ backoff.BackOff = (*Retry)(nil)

func NewReconnectStrategy(ctx context.Context, d time.Duration) *Retry {
	return &Retry{
		ctx:      ctx,
		duration: d,
	}
}

func (r *Retry) NextBackOff() time.Duration {
	if r.ctx.Err() != nil {
		return backoff.Stop
	}
	return r.duration
}
func (r *Retry) Reset() {
	// donthing
}
