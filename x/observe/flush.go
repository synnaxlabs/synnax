package observe

import (
	"github.com/arya-analytics/x/binary"
	"github.com/arya-analytics/x/kv"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"time"
)

// |||||| FLUSH ||||||

// FlushSubscriber is used to flush an observable whose contents implement
//
//	the kv.Flusher interface.
type FlushSubscriber[S any] struct {
	// Key is the key to flush the contents of the observable into.
	Key []byte
	// Store is the store to flush the contents of the observable into.
	Store kv.DB
	// MinInterval specifies the minimum interval between flushes. If the observable
	// updates more quickly than min interval, the FlushSubscriber will not flush the
	// contents.
	MinInterval time.Duration
	// LastFlush stores the last time the observable was flushed.
	LastFlush time.Time
	// Encoder is the encoder to use when flushing the contents of the state
	Encoder binary.Encoder
	// Logger is the witness of it all.
	Logger *zap.SugaredLogger
}

// Flush is the handler to bind to the Observable.
func (f *FlushSubscriber[S]) Flush(state S) {
	if time.Since(f.LastFlush) < f.MinInterval {
		return
	}
	go f.FlushSync(state)
}

func (f *FlushSubscriber[S]) FlushSync(state S) {
	if err := f.Store.Set(f.Key, lo.Must(f.Encoder.Encode(state))); err != nil {
		f.Logger.Errorw("failed to flush", "err", err)
	} else {
		f.LastFlush = time.Now()
	}
}
