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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/signal"
)

// Writer is a writable key-value storeSink.
type Writer interface {
	// Writer represents the same interface to a typical key-value storeSink.
	kvx.Writer
}

type (
	// Reader is a readable key-value storeSink.
	Reader = kvx.Reader
)

type (
	BatchWriter = kvx.BatchWriter
)

type db struct {
	kvx.DB
	Config
	leaseAlloc *leaseAllocator
	confluence.AbstractUnarySource[BatchRequest]
	confluence.EmptyFlow
	shutdown context.CancelFunc
	wg       signal.WaitGroup
}

// Set implements DB.
func (k *db) Set(ctx context.Context, key []byte, value []byte, options ...interface{}) error {
	b := k.NewBatch()
	if err := b.Set(ctx, key, value, options...); err != nil {
		return err
	}
	return b.Commit(ctx)
}

// Delete implements DB.
func (k *db) Delete(ctx context.Context, key []byte) error {
	b := k.NewBatch()
	if err := b.Delete(ctx, key); err != nil {
		return err
	}
	return b.Commit(ctx)
}

func (k *db) apply(b []BatchRequest) (err error) {
	c := batchCoordinator{}
	for _, bd := range b {
		c.add(&bd)
		k.Out.Inlet() <- bd
	}
	return c.wait()
}

func (k *db) NewBatch() kvx.Batch {
	return &batch{apply: k.apply, lease: k.leaseAlloc, Batch: k.DB.NewBatch()}
}

func (k *db) Report() alamos.Report {
	return alamos.Report{
		"engine": "aspen-pebble",
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
	cfg, err := config.OverrideAndValidate(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}

	va, err := newVersionAssigner(ctx, cfg.Engine)
	if err != nil {
		return nil, err
	}

	alamos.L(ctx).Info("opening cluster db", cfg.Report().LogArgs()...)

	st := newStore()

	sCtx, cancel := signal.WithCancel(alamos.Transfer(context.Background(), ctx))

	db := &db{
		Config:     cfg,
		DB:         cfg.Engine,
		leaseAlloc: &leaseAllocator{Config: cfg},
		shutdown:   cancel,
		wg:         sCtx,
	}

	pipe := plumber.New()
	plumber.SetSource[BatchRequest](pipe, executorAddr, db)
	plumber.SetSource[BatchRequest](pipe, leaseReceiverAddr, newLeaseReceiver(cfg))
	plumber.SetSegment[BatchRequest, BatchRequest](
		pipe,
		leaseProxyAddr,
		newLeaseProxy(cfg, versionAssignerAddr, leaseSenderAddr),
	)
	plumber.SetSource[BatchRequest](pipe, operationReceiverAddr, newOperationReceiver(cfg, st))
	plumber.SetSegment[BatchRequest](
		pipe,
		versionFilterAddr,
		newVersionFilter(cfg, persistAddr, feedbackSenderAddr),
	)
	plumber.SetSegment[BatchRequest](pipe, versionAssignerAddr, va)
	plumber.SetSink[BatchRequest](pipe, leaseSenderAddr, newLeaseSender(cfg))
	plumber.SetSegment[BatchRequest, BatchRequest](pipe, persistAddr, newPersist(cfg.Engine))
	plumber.SetSource[BatchRequest](pipe, storeEmitterAddr, newStoreEmitter(st, cfg))
	plumber.SetSink[BatchRequest](pipe, storeSinkAddr, newStoreSink(st))
	plumber.SetSegment[BatchRequest, BatchRequest](
		pipe,
		operationSenderAddr,
		newOperationSender(cfg),
	)
	plumber.SetSink[BatchRequest](pipe, feedbackSenderAddr, newFeedbackSender(cfg))
	plumber.SetSource[BatchRequest](pipe, feedbackReceiverAddr, newFeedbackReceiver(cfg))
	plumber.SetSegment[BatchRequest, BatchRequest](
		pipe,
		recoveryTransformAddr,
		newRecoveryTransform(cfg),
	)

	plumber.MultiRouter[BatchRequest]{
		SourceTargets: []address.Address{executorAddr, leaseReceiverAddr},
		SinkTargets:   []address.Address{leaseProxyAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      1,
	}.MustRoute(pipe)

	plumber.MultiRouter[BatchRequest]{
		SourceTargets: []address.Address{leaseProxyAddr},
		SinkTargets:   []address.Address{versionAssignerAddr, leaseSenderAddr},
		Stitch:        plumber.StitchWeave,
		Capacity:      1,
	}.MustRoute(pipe)

	plumber.MultiRouter[BatchRequest]{
		SourceTargets: []address.Address{operationReceiverAddr, operationSenderAddr},
		SinkTargets:   []address.Address{versionFilterAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      1,
	}.MustRoute(pipe)

	plumber.MultiRouter[BatchRequest]{
		SourceTargets: []address.Address{versionFilterAddr, versionAssignerAddr},
		SinkTargets:   []address.Address{persistAddr},
		Capacity:      1,
		Stitch:        plumber.StitchUnary,
	}.MustRoute(pipe)

	plumber.UnaryRouter[BatchRequest]{
		SourceTarget: versionFilterAddr,
		SinkTarget:   feedbackSenderAddr,
		Capacity:     1,
	}.MustRoute(pipe)

	plumber.UnaryRouter[BatchRequest]{
		SourceTarget: feedbackReceiverAddr,
		SinkTarget:   recoveryTransformAddr,
		Capacity:     1,
	}.MustRoute(pipe)

	plumber.MultiRouter[BatchRequest]{
		SourceTargets: []address.Address{persistAddr, recoveryTransformAddr},
		SinkTargets:   []address.Address{storeSinkAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      1,
	}.MustRoute(pipe)

	plumber.UnaryRouter[BatchRequest]{
		SourceTarget: storeEmitterAddr,
		SinkTarget:   operationSenderAddr,
		Capacity:     1,
	}.MustRoute(pipe)

	pipe.Flow(sCtx)

	return db, nil
}

func (k *db) Close() error {
	k.shutdown()
	return k.wg.Wait()
}
