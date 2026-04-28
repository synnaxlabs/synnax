// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"io"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
)

type DB struct {
	xkv.DB
	source struct {
		confluence.NopFlow
		confluence.AbstractUnarySource[TxRequest]
	}
	// txObservable fires after a TxRequest has been persisted, carrying the
	// full TxRequest (not just its flattened reader) so per-handler wrappers
	// can inspect leaseholder / sender metadata before invoking the user
	// callback. Wired in Open from the persistDelta stage.
	txObservable *confluence.ObservableSubscriber[TxRequest]
	shutdown     io.Closer
	leaseAlloc   *leaseAllocator
	config       Config
}

var _ xkv.DB = (*DB)(nil)

func (d *DB) Set(
	ctx context.Context,
	key, value []byte,
	maybeLease ...any,
) (err error) {
	b := d.OpenTx()
	defer func() { err = errors.Combine(err, b.Close()) }()
	if err = b.Set(ctx, key, value, maybeLease...); err != nil {
		return err
	}
	err = b.Commit(ctx)
	return
}

func (d *DB) Delete(
	ctx context.Context,
	key []byte,
	maybeLease ...any,
) (err error) {
	b := d.OpenTx()
	defer func() { err = errors.Combine(err, b.Close()) }()
	if err = b.Delete(ctx, key, maybeLease...); err != nil {
		return err
	}
	err = b.Commit(ctx)
	return
}

func (d *DB) OpenTx() xkv.Tx {
	return &tx{
		Instrumentation: d.config.Instrumentation,
		apply:           d.apply,
		lease:           d.leaseAlloc,
		Tx:              d.DB.OpenTx(),
	}
}

// OnChange satisfies xkv.Observable: handler receives every persisted
// TxRequest as a flattened xkv.TxReader, regardless of which node was the
// leaseholder. For filtered views over the same stream, see NewObservable.
func (d *DB) OnChange(handler func(ctx context.Context, reader xkv.TxReader)) observe.Disconnect {
	return d.NewObservable().OnChange(handler)
}

// NewObservable returns a view over this DB's persisted tx stream.
//
// Without options, the returned observable is equivalent to OnChange: every
// persisted TxRequest is yielded to subscribers as a flattened xkv.TxReader.
// Options reshape what subscribers see — for example, IgnoreHostLeaseholder
// drops TxRequests led by the host node so subscribers only observe writes
// that originated on a remote node and were replicated here via gossip.
//
// Each call to NewObservable returns an independent observable; multiple
// subscribers to the same observable share its filter, while subscribers
// that need different filters should construct separate observables.
func (d *DB) NewObservable(opts ...ObservableOption) observe.Observable[xkv.TxReader] {
	return &observable{db: d, opts: resolveObservableOptions(opts)}
}

// observable is the concrete observable returned by NewObservable. It
// subscribes to the DB's TxRequest pump on every OnChange call, applies the
// configured filter, and forwards accepted requests to handler as flattened
// xkv.TxReaders.
type observable struct {
	db   *DB
	opts observableOptions
}

// OnChange implements observe.Observable.
func (o *observable) OnChange(
	handler func(ctx context.Context, reader xkv.TxReader),
) observe.Disconnect {
	return o.db.txObservable.OnChange(func(ctx context.Context, tx TxRequest) {
		if o.opts.ignoreHostLeaseholder && tx.Leaseholder == o.db.config.Cluster.HostKey() {
			return
		}
		handler(ctx, tx.reader())
	})
}

func (d *DB) apply(b []TxRequest) (err error) {
	c := txCoordinator{}
	for _, bd := range b {
		c.add(&bd)
		if sendErr := signal.SendUnderContext(
			bd.Context, d.source.Out.Inlet(), bd,
		); sendErr != nil {
			bd.done(sendErr)
		}
	}
	return c.wait()
}

func (d *DB) Report() alamos.Report {
	return alamos.Report{
		"engine":  "aspen",
		"wrapped": d.DB.Report(),
	}
}

const (
	versionFilterAddr     = "version_filter"
	versionAssignerAddr   = "version_assigner"
	persistAddr           = "persist"
	persistDeltaAddr      = "persist_delta"
	storeEmitterAddr      = "store_emitter"
	storeSinkAddr         = "store_sink"
	observableAddr        = "observable"
	operationSenderAddr   = "op_sender"
	operationReceiverAddr = "op_receiver"
	feedbackSenderAddr    = "feedback_sender"
	feedbackReceiverAddr  = "feedback_receiver"
	recoveryTransformAddr = "recovery_transform"
	leaseSenderAddr       = "lease_sender"
	leaseReceiverAddr     = "lease_receiver"
	leaseProxyAddr        = "lease_proxy"
	executorAddr          = "executor"
	chanBuffer            = 100
	observableRelayBuffer = 500
)

func Open(ctx context.Context, cfgs ...Config) (db *DB, err error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	db = &DB{
		config:     cfg,
		DB:         cfg.Engine,
		leaseAlloc: &leaseAllocator{Config: cfg},
		shutdown:   signal.NewHardShutdown(sCtx, cancel),
	}
	defer func() {
		if err != nil {
			err = errors.Combine(err, db.shutdown.Close())
		}
	}()

	va, err := newVersionAssigner(ctx, cfg)
	if err != nil {
		return nil, err
	}

	db.config.L.Debug("opening cluster KV", db.config.Report().ZapFields()...)

	st := newStore()

	pipe := plumber.New()
	plumber.SetSource[TxRequest](pipe, executorAddr, &db.source)
	plumber.SetSource[TxRequest](pipe, leaseReceiverAddr, newLeaseReceiver(cfg))
	plumber.SetSegment[TxRequest, TxRequest](
		pipe,
		leaseProxyAddr,
		newLeaseProxy(cfg, versionAssignerAddr, leaseSenderAddr),
	)
	opServer := newOperationServer(cfg, st)
	plumber.SetSource[TxRequest](pipe, operationReceiverAddr, opServer)
	plumber.SetSegment[TxRequest](
		pipe,
		versionFilterAddr,
		newVersionFilter(cfg, persistAddr, feedbackSenderAddr),
	)
	plumber.SetSegment[TxRequest](pipe, versionAssignerAddr, va)
	plumber.SetSink[TxRequest](pipe, leaseSenderAddr, newLeaseSender(cfg))
	plumber.SetSegment[TxRequest, TxRequest](pipe, persistAddr, newPersist(cfg.Engine))
	plumber.SetSource[TxRequest](pipe, storeEmitterAddr, newStoreEmitter(st, cfg))
	plumber.SetSink[TxRequest](pipe, storeSinkAddr, newStoreSink(st))
	plumber.SetSegment[TxRequest, TxRequest](
		pipe,
		operationSenderAddr,
		newOperationClient(cfg),
	)
	plumber.SetSink[TxRequest](pipe, feedbackSenderAddr, newFeedbackSender(cfg))
	fbReceiver := newFeedbackReceiver(cfg)
	plumber.SetSource[TxRequest](pipe, feedbackReceiverAddr, fbReceiver)
	plumber.SetSegment[TxRequest, TxRequest](
		pipe,
		recoveryTransformAddr,
		newGossipRecoveryTransform(cfg),
	)

	plumber.SetSegment[TxRequest, TxRequest](
		pipe,
		persistDeltaAddr,
		newPersistSplitter(observableRelayBuffer, cfg.Instrumentation),
	)

	// The observable fans persisted TxRequests out to subscribers, preserving
	// the full TxRequest (rather than the flattened reader) so per-handler
	// wrappers can filter on leaseholder / sender before yielding to the user
	// callback. Each handler builds its own xkv.TxReader from tx.reader(),
	// which produces a fresh iterator per call, so there is no need for the
	// generator-style observable used previously.
	observable := confluence.NewObservableSubscriber[TxRequest]()
	observable.Observer = observe.NewAsync[TxRequest](
		sCtx,
		signal.WithRetryOnPanic(100),
		signal.WithBaseRetryInterval(500*time.Millisecond),
		signal.WithRetryScale(1.1),
	)
	plumber.SetSink[TxRequest](pipe, observableAddr, observable)
	db.txObservable = observable

	plumber.MultiRouter[TxRequest]{
		SourceTargets: []address.Address{executorAddr, leaseReceiverAddr},
		SinkTargets:   []address.Address{leaseProxyAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      chanBuffer,
	}.MustRoute(pipe)

	plumber.MultiRouter[TxRequest]{
		SourceTargets: []address.Address{leaseProxyAddr},
		SinkTargets:   []address.Address{versionAssignerAddr, leaseSenderAddr},
		Stitch:        plumber.StitchWeave,
		Capacity:      chanBuffer,
	}.MustRoute(pipe)

	plumber.MultiRouter[TxRequest]{
		SourceTargets: []address.Address{operationReceiverAddr, operationSenderAddr},
		SinkTargets:   []address.Address{versionFilterAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      chanBuffer,
	}.MustRoute(pipe)

	plumber.MultiRouter[TxRequest]{
		SourceTargets: []address.Address{versionFilterAddr, versionAssignerAddr},
		SinkTargets:   []address.Address{persistAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      chanBuffer,
	}.MustRoute(pipe)

	plumber.UnaryRouter[TxRequest]{
		SourceTarget: persistAddr,
		SinkTarget:   persistDeltaAddr,
		Capacity:     chanBuffer,
	}.MustRoute(pipe)

	plumber.UnaryRouter[TxRequest]{
		SourceTarget: versionFilterAddr,
		SinkTarget:   feedbackSenderAddr,
		Capacity:     chanBuffer,
	}.MustRoute(pipe)

	plumber.UnaryRouter[TxRequest]{
		SourceTarget: feedbackReceiverAddr,
		SinkTarget:   recoveryTransformAddr,
		Capacity:     chanBuffer,
	}.MustRoute(pipe)

	plumber.MultiRouter[TxRequest]{
		SourceTargets: []address.Address{persistDeltaAddr, recoveryTransformAddr},
		SinkTargets:   []address.Address{storeSinkAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      chanBuffer,
	}.MustRoute(pipe)

	plumber.UnaryRouter[TxRequest]{
		SourceTarget: persistDeltaAddr,
		SinkTarget:   observableAddr,
		// Setting the capacity higher here allows us to unclog the pipeline in case
		// we have a slow observer.
		Capacity: chanBuffer,
	}.MustRoute(pipe)

	plumber.UnaryRouter[TxRequest]{
		SourceTarget: storeEmitterAddr,
		SinkTarget:   operationSenderAddr,
		Capacity:     chanBuffer,
	}.MustRoute(pipe)
	newRecoveryServer(cfg)
	// Bind RPC handlers after routing so outlets are wired before any incoming gossip
	// message can invoke the handler.
	cfg.BatchTransportServer.BindHandler(opServer.handle)
	cfg.FeedbackTransportServer.BindHandler(fbReceiver.handle)
	pipe.Flow(
		sCtx,
		confluence.RecoverWithoutErrOnPanic(),
		confluence.WithRetryOnPanic(100),
	)
	return db, runRecovery(ctx, cfg)
}

func (d *DB) Close() error { return d.shutdown.Close() }
