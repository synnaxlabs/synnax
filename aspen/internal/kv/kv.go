// Copyright 2025 Synnax Labs, Inc.
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
	xkv.Observable
	config     Config
	leaseAlloc *leaseAllocator
	source     struct {
		confluence.AbstractUnarySource[TxRequest]
		confluence.NopFlow
	}
	shutdown io.Closer
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

func (d *DB) OnChange(f func(ctx context.Context, reader xkv.TxReader)) observe.Disconnect {
	return d.Observable.OnChange(f)
}

func (d *DB) apply(b []TxRequest) (err error) {
	c := txCoordinator{}
	for _, bd := range b {
		c.add(&bd)
		d.source.Out.Inlet() <- bd
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
)

func Open(ctx context.Context, cfgs ...Config) (*DB, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	db_ := &DB{
		config:     cfg,
		DB:         cfg.Engine,
		leaseAlloc: &leaseAllocator{Config: cfg},
		shutdown:   signal.NewHardShutdown(sCtx, cancel),
	}

	va, err := newVersionAssigner(ctx, cfg)
	if err != nil {
		return nil, err
	}

	db_.config.L.Debug("opening cluster KV", db_.config.Report().ZapFields()...)

	st := newStore()

	pipe := plumber.New()
	plumber.SetSource[TxRequest](pipe, executorAddr, &db_.source)
	plumber.SetSource[TxRequest](pipe, leaseReceiverAddr, newLeaseReceiver(cfg))
	plumber.SetSegment[TxRequest, TxRequest](
		pipe,
		leaseProxyAddr,
		newLeaseProxy(cfg, versionAssignerAddr, leaseSenderAddr),
	)
	plumber.SetSource[TxRequest](pipe, operationReceiverAddr, newOperationServer(cfg, st))
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
	plumber.SetSource[TxRequest](pipe, feedbackReceiverAddr, newFeedbackReceiver(cfg))
	plumber.SetSegment[TxRequest, TxRequest](
		pipe,
		recoveryTransformAddr,
		newGossipRecoveryTransform(cfg),
	)

	plumber.SetSegment[TxRequest, TxRequest](
		pipe,
		persistDeltaAddr,
		&confluence.DeltaMultiplier[TxRequest]{},
	)

	// We use a generator observable to generate a unique transaction reader for
	// each handler in the observable chain. This is necessary because the transaction
	// reader can be exhausted.
	observable := confluence.NewGeneratorTransformObservable[TxRequest, xkv.TxReader](
		func(_ context.Context, tx TxRequest) (func() xkv.TxReader, bool, error) {
			return func() xkv.TxReader { return tx.reader() }, true, nil
		})
	plumber.SetSink[TxRequest](pipe, observableAddr, observable)
	db_.Observable = observable

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
	pipe.Flow(
		sCtx,
		confluence.RecoverWithoutErrOnPanic(),
		confluence.WithRetryOnPanic(100),
	)
	return db_, runRecovery(ctx, cfg)
}

func (d *DB) Close() error { return d.shutdown.Close() }
