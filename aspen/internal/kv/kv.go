package kv

import (
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/config"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/signal"
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

// DB is a readable and writable key-value storeSink.
type DB interface {
	Writer
	Reader
	BatchWriter
	alamos.Reporter
}

type kv struct {
	kvx.DB
	Config
	leaseAlloc *leaseAllocator
	confluence.AbstractUnarySource[BatchRequest]
	confluence.EmptyFlow
}

// Set implements DB.
func (k *kv) Set(key []byte, value []byte, maybeLease ...interface{}) error {
	b := k.NewBatch()
	if err := b.Set(key, value, maybeLease...); err != nil {
		return err
	}
	return b.Commit()
}

// Delete implements DB.
func (k *kv) Delete(key []byte) error {
	b := k.NewBatch()
	if err := b.Delete(key); err != nil {
		return err
	}
	return b.Commit()
}

func (k *kv) apply(b []BatchRequest) (err error) {
	c := batchCoordinator{}
	for _, bd := range b {
		c.add(&bd)
		k.Out.Inlet() <- bd
	}
	return c.wait()
}

func (k *kv) NewBatch() kvx.Batch {
	return &batch{apply: k.apply, lease: k.leaseAlloc, Batch: k.DB.NewBatch()}
}

func (k *kv) Report() alamos.Report {
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

func Open(ctx signal.Context, cfgs ...Config) (DB, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	db := &kv{
		Config:     cfg,
		DB:         cfg.Engine,
		leaseAlloc: &leaseAllocator{Config: cfg},
	}

	cfg.Logger.Infow("opening cluster kv", cfg.Report().LogArgs()...)

	va, err := newVersionAssigner(cfg)
	if err != nil {
		return nil, err
	}

	st := newStore()

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

	pipe.Flow(ctx)

	return db, nil
}
