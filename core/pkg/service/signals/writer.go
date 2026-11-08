// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"context"
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// feed adds the samples that one commit gives its change channels to a frame.
type feed interface {
	append(ctx context.Context, c gorp.Commit, fr framer.Frame) framer.Frame
}

// add creates channels as free, virtual, internal channels, stores their keys in them,
// and adds f to the feeds that Start writes.
func (p *Provider) add(
	ctx context.Context,
	f feed,
	channels ...*channel.Channel,
) error {
	if p.started {
		return errors.New("[signals] - cannot publish a change channel after Start")
	}
	created := make([]channel.Channel, len(channels))
	for i, ch := range channels {
		created[i] = *ch
		created[i].Virtual = true
		created[i].Internal = true
		created[i].Leaseholder = node.KeyFree
	}
	if err := p.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return p.cfg.Channel.NewWriter(tx).CreateMany(
			ctx,
			&created,
			channel.RetrieveIfNameExists(),
			channel.OverwriteIfNameExistsAndDifferentProperties(),
		)
	}); err != nil {
		return err
	}
	for i, ch := range channels {
		*ch = created[i]
		p.keys = append(p.keys, ch.Key())
	}
	p.feeds = append(p.feeds, f)
	return nil
}

// Start opens one writer for every channel that PublishTable and PublishActions
// created, and writes each later commit of the DB to them as one frame. A commit before
// Start writes nothing. The committing goroutine writes the frame, and waits while the
// writer is full. Close the returned io.Closer to stop writing.
func (p *Provider) Start(ctx context.Context) (io.Closer, error) {
	if p.started {
		return nil, errors.New("[signals] - Start already ran")
	}
	p.started = true
	if len(p.feeds) == 0 {
		return xio.NopCloser, nil
	}
	sw, err := p.cfg.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		Keys:        p.keys,
		Start:       telem.Now(),
		Authorities: []control.Authority{control.AuthorityAbsolute},
	})
	if err != nil {
		return nil, err
	}
	requests := confluence.NewStream[framer.WriterRequest](requestBufferSize)
	responses := confluence.NewStream[framer.WriterResponse]()
	sw.InFrom(requests)
	sw.OutTo(responses)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(p.cfg.Instrumentation))
	sw.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.CancelOnFail(),
		confluence.RecoverWithErrOnPanic(),
	)
	sCtx.Go(func(context.Context) error {
		for res := range responses.Outlet() {
			p.cfg.L.Error(
				"unexpected change channel writer response",
				zap.Int("seq_num", res.SeqNum),
				zap.Error(res.Err),
			)
		}
		return nil
	}, signal.WithKey("responses"))
	w := &changeWriter{
		feeds:    p.feeds,
		L:        p.cfg.L,
		requests: requests,
		stopped:  sCtx.Stopped(),
		shutdown: signal.NewGracefulShutdown(sCtx, cancel),
	}
	w.disconnect = p.cfg.DB.Observe().OnChange(w.write)
	return w, nil
}

// changeWriter writes each commit of a DB to the change channels as one frame.
type changeWriter struct {
	feeds      []feed
	L          *alamos.Logger
	disconnect observe.Disconnect
	// mu guards closed and requests, so a frame never goes to a closed writer.
	mu       sync.Mutex
	closed   bool
	requests confluence.Inlet[framer.WriterRequest]
	// stopped closes when the writer stops.
	stopped  <-chan struct{}
	shutdown io.Closer
}

func (w *changeWriter) write(ctx context.Context, c gorp.Commit) {
	var fr framer.Frame
	for _, f := range w.feeds {
		fr = f.append(ctx, c, fr)
	}
	if fr.Empty() {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	select {
	case w.requests.Inlet() <- framer.WriterRequest{
		Command: framer.WriterCommandWrite,
		Frame:   fr,
	}:
	case <-w.stopped:
		w.L.Error("change channel writer stopped", zap.Uint64("commit", c.Seq))
	}
}

// Close stops writing commits and closes the writer once it writes the frames it holds.
func (w *changeWriter) Close() error {
	w.disconnect()
	w.mu.Lock()
	w.closed = true
	w.requests.Close()
	w.mu.Unlock()
	return w.shutdown.Close()
}
