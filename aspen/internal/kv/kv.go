// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/signal"
	"io"
)

type db struct {
	kvx.DB
	Config
	leaseAlloc *leaseAllocator
	confluence.AbstractUnarySource[TxRequest]
	confluence.EmptyFlow
	shutdown io.Closer
}

var _ kvx.DB = (*db)(nil)

func (d *db) Set(
	ctx context.Context,
	key, value []byte,
	maybeLease ...interface{},
) (err error) {
	b := d.OpenTx()
	defer func() { err = errors.CombineErrors(err, b.Close()) }()
	if err = b.Set(ctx, key, value, maybeLease...); err != nil {
		return err
	}
	err = b.Commit(ctx)
	return
}

func (d *db) Delete(
	ctx context.Context,
	key []byte,
	maybeLease ...interface{},
) (err error) {
	b := d.OpenTx()
	defer func() { err = errors.CombineErrors(err, b.Close()) }()
	if err = b.Delete(ctx, key, maybeLease...); err != nil {
		return err
	}
	err = b.Commit(ctx)
	return
}

func (d *db) OpenTx() kvx.Tx {
	return &tx{
		Instrumentation: d.Instrumentation,
		apply:           d.apply,
		lease:           d.leaseAlloc,
		Tx:              d.DB.OpenTx(),
	}
}

func (d *db) apply(b []TxRequest) (err error) {
	c := txCoordinator{}
	for _, bd := range b {
		c.add(&bd)
		d.Out.Inlet() <- bd
	}
	return c.wait()
}

func (d *db) Report() alamos.Report {
	return alamos.Report{
		"engine":  "aspen",
		"wrapped": d.DB.Report(),
	}
}

const (
	versionFilterAddr     = "versionFilter"
	versionAssignerAddr   = "versionAssigner"
	persistAddr           = "persist"
	storeEmitterAddr      = "storeEmitter"
	storeSinkAddr         = "storeSink"
	operationSenderAddr   = "opSender"
	operationReceiverAddr = "opReceiver"
	feedbackSenderAddr    = "feedbackSender"
	feedbackReceiverAddr  = "feedbackReceiver"
	recoveryTransformAddr = "recoveryTransform"
	leaseSenderAddr       = "leaseSender"
	leaseReceiverAddr     = "leaseReceiver"
	leaseProxyAddr        = "leaseProxy"
	executorAddr          = "executor"
)

func Open(ctx context.Context, cfgs ...Config) (kvx.DB, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	db_ := &db{
		Config:     cfg,
		DB:         cfg.Engine,
		leaseAlloc: &leaseAllocator{Config: cfg},
		shutdown:   signal.NewShutdown(sCtx, cancel),
	}

	va, err := newVersionAssigner(ctx, cfg)
	if err != nil {
		return nil, err
	}

	db_.L.Info("opening cluster db", db_.Report().ZapFields()...)

	st := newStore()

	pipe := plumber.New()
	plumber.SetSource[TxRequest](pipe, executorAddr, db_)
	plumber.SetSource[TxRequest](pipe, leaseReceiverAddr, newLeaseReceiver(cfg))
	plumber.SetSegment[TxRequest, TxRequest](
		pipe,
		leaseProxyAddr,
		newLeaseProxy(cfg, versionAssignerAddr, leaseSenderAddr),
	)
	plumber.SetSource[TxRequest](pipe, operationReceiverAddr, newOperationReceiver(cfg, st))
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
		newOperationSender(cfg),
	)
	plumber.SetSink[TxRequest](pipe, feedbackSenderAddr, newFeedbackSender(cfg))
	plumber.SetSource[TxRequest](pipe, feedbackReceiverAddr, newFeedbackReceiver(cfg))
	plumber.SetSegment[TxRequest, TxRequest](
		pipe,
		recoveryTransformAddr,
		newRecoveryTransform(cfg),
	)

	plumber.MultiRouter[TxRequest]{
		SourceTargets: []address.Address{executorAddr, leaseReceiverAddr},
		SinkTargets:   []address.Address{leaseProxyAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      1,
	}.MustRoute(pipe)

	plumber.MultiRouter[TxRequest]{
		SourceTargets: []address.Address{leaseProxyAddr},
		SinkTargets:   []address.Address{versionAssignerAddr, leaseSenderAddr},
		Stitch:        plumber.StitchWeave,
		Capacity:      1,
	}.MustRoute(pipe)

	plumber.MultiRouter[TxRequest]{
		SourceTargets: []address.Address{operationReceiverAddr, operationSenderAddr},
		SinkTargets:   []address.Address{versionFilterAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      1,
	}.MustRoute(pipe)

	plumber.MultiRouter[TxRequest]{
		SourceTargets: []address.Address{versionFilterAddr, versionAssignerAddr},
		SinkTargets:   []address.Address{persistAddr},
		Capacity:      1,
		Stitch:        plumber.StitchUnary,
	}.MustRoute(pipe)

	plumber.UnaryRouter[TxRequest]{
		SourceTarget: versionFilterAddr,
		SinkTarget:   feedbackSenderAddr,
		Capacity:     1,
	}.MustRoute(pipe)

	plumber.UnaryRouter[TxRequest]{
		SourceTarget: feedbackReceiverAddr,
		SinkTarget:   recoveryTransformAddr,
		Capacity:     1,
	}.MustRoute(pipe)

	plumber.MultiRouter[TxRequest]{
		SourceTargets: []address.Address{persistAddr, recoveryTransformAddr},
		SinkTargets:   []address.Address{storeSinkAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      1,
	}.MustRoute(pipe)

	plumber.UnaryRouter[TxRequest]{
		SourceTarget: storeEmitterAddr,
		SinkTarget:   operationSenderAddr,
		Capacity:     1,
	}.MustRoute(pipe)

	pipe.Flow(sCtx)

	return db_, nil
}

func (d *db) Close() error {
	if err := d.DB.Close(); err != nil {
		return err
	}
	return d.shutdown.Close()
}
