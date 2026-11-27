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
	"maps"
	"runtime/pprof"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// feed adds the samples that one commit gives its change channels to a frame.
type feed interface {
	append(c gorp.Commit, fr framer.Frame) framer.Frame
	// records returns the Record of each channel of the feed. It runs once the channels
	// have keys.
	records() map[channel.Key]Record
}

// add creates channels as free, virtual, internal channels, stores their keys in them,
// and adds f to the feeds that Run writes and its records to Records.
func (p *Provider) add(
	ctx context.Context,
	f feed,
	channels ...*channel.Channel,
) error {
	if p.started.Load() {
		return errors.New("[signals] - cannot publish a change channel after Run")
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
	maps.Copy(p.records, f.records())
	return nil
}

// Run writes each commit of the DB to the channels that PublishTable and PublishActions
// created, as one frame, until ctx ends. The committing goroutine writes the frame, and
// waits while the writer is full. A commit before Run starts, or after Run returns,
// writes nothing. Run returns the error of a failed writer. A second Run, or a Run
// after Close, returns an error.
func (p *Provider) Run(ctx context.Context) (err error) {
	if err = p.guard.Begin(); err != nil {
		return err
	}
	defer p.guard.End()
	p.started.Store(true)
	pprof.Do(ctx, pprof.Labels("routine", "signals"), func(ctx context.Context) {
		err = p.run(ctx)
	})
	return err
}

func (p *Provider) run(ctx context.Context) error {
	if len(p.feeds) == 0 {
		<-ctx.Done()
		return nil
	}
	// The writer outlives ctx, so that it writes the frames it holds before it closes.
	sCtx, cancel := signal.WithCancel(
		context.WithoutCancel(ctx),
		signal.WithInstrumentation(p.cfg.Instrumentation),
	)
	defer cancel()
	sw, err := p.cfg.Framer.NewStreamWriter(sCtx, framer.WriterConfig{
		Keys:        p.keys,
		Start:       telem.Now(),
		Authorities: []control.Authority{control.AuthorityAbsolute},
	})
	if err != nil {
		return err
	}
	requests := confluence.NewStream[framer.WriterRequest](requestBufferSize)
	responses := confluence.NewStream[framer.WriterResponse]()
	sw.InFrom(requests)
	sw.OutTo(responses)
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
	}
	disconnect := p.cfg.DB.Observe().OnChange(w.write)
	select {
	case <-ctx.Done():
	case <-sCtx.Done():
	}
	disconnect()
	w.close()
	return sCtx.Wait()
}

// Close releases the resources of the Provider. Call it after Run returns: while Run
// runs, Close returns an error.
func (p *Provider) Close() error {
	_, err := p.guard.Close()
	return err
}

// changeWriter writes each commit of a DB to the change channels as one frame.
type changeWriter struct {
	feeds []feed
	L     *alamos.Logger
	// mu guards closed and requests, so a frame never goes to a closed writer.
	mu       sync.Mutex
	closed   bool
	requests confluence.Inlet[framer.WriterRequest]
	// stopped closes when the writer stops.
	stopped <-chan struct{}
}

func (w *changeWriter) write(ctx context.Context, c gorp.Commit) {
	var fr framer.Frame
	for _, f := range w.feeds {
		fr = f.append(c, fr)
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

// close stops writing commits, so that the writer closes once it writes the frames it
// holds.
func (w *changeWriter) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	w.requests.Close()
}
