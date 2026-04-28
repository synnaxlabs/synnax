// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

// ObservableOption configures an observable returned by DB.NewObservable.
// ObservableOption is sealed: implementations live in this package.
type ObservableOption interface {
	apply(*observableOptions)
}

// observableOptions is the resolved option set produced by applying every
// ObservableOption to a zero value.
type observableOptions struct {
	// ignoreHostLeaseholder, when true, suppresses notifications for any
	// TxRequest whose Leaseholder is the host node. See IgnoreHostLeaseholder.
	ignoreHostLeaseholder bool
}

func resolveObservableOptions(opts []ObservableOption) observableOptions {
	var o observableOptions
	for _, opt := range opts {
		opt.apply(&o)
	}
	return o
}

type ignoreHostLeaseholderOpt struct{}

func (ignoreHostLeaseholderOpt) apply(o *observableOptions) {
	o.ignoreHostLeaseholder = true
}

// IgnoreHostLeaseholder, when passed to DB.NewObservable, returns an observable
// that filters out TxRequests whose Leaseholder is the host node. Subscribers
// to that observable see only writes that originated on a remote node and
// were replicated here via gossip. Use this when a higher layer has its own
// path for propagating local writes (e.g. a per-tx delta flush) and only
// wants the observer for replicated writes.
var IgnoreHostLeaseholder ObservableOption = ignoreHostLeaseholderOpt{}
