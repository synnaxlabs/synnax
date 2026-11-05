// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/vmihailenco/msgpack/v5"
)

// These type aliases re-export the distribution-layer key types so service-layer
// callers can refer to them through this package without importing the distribution
// layer directly.
type (
	// Key is the cluster-unique identifier for a channel. It packs a leaseholder node
	// key (first 12 bits) and a node-local LocalKey (last 20 bits) into a single
	// uint32. Re-exported from [channel.Key].
	Key = channel.Key
	// LocalKey is the 20-bit, node-local portion of a channel Key. Re-exported from
	// [channel.LocalKey].
	LocalKey = channel.LocalKey
)

var _ gorp.Entry[Key] = Channel{}

// Key returns the key for the Channel.
func (c Channel) Key() Key { return channel.NewKey(c.Leaseholder, c.LocalKey) }

// GorpKey implements the gorp.Entry interface.
func (c Channel) GorpKey() Key { return c.Key() }

// SetOptions implements the gorp.Entry interface. Returns a set of options that tell an
// aspen.DB to properly lease the Channel to the node it will be recording data from.
func (c Channel) SetOptions() []any {
	if c.Free() {
		return []any{node.KeyBootstrapper}
	}
	return []any{c.Leaseholder}
}

// Free returns true if the channel is not leased to a particular node i.e. it is a
// non-leased virtual channel.
func (c Channel) Free() bool { return c.Leaseholder == node.KeyFree }

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase Go
// field names (e.g. "Type", "ResetChannel", "Duration") and new lowercase msgpack tag
// names for backward compatibility.
func (o *Operation) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Operation
	raw, err := dec.DecodeRaw()
	if err != nil {
		return errors.Wrap(err, "failed to read raw operation msgpack")
	}
	if err = msgpack.Unmarshal(raw, (*alias)(o)); err != nil {
		return errors.Wrap(err, "failed to decode operation from msgpack")
	}
	if len(o.Type) == 0 {
		var legacy struct {
			Type         OperationType
			ResetChannel Key
			Duration     telem.TimeSpan
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return errors.Wrap(err, "failed to decode legacy operation from msgpack")
		}
		o.Type = legacy.Type
		o.ResetChannel = legacy.ResetChannel
		o.Duration = legacy.Duration
	}
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy "node_id" and
// new "leaseholder" field names for backward compatibility.
func (c *Channel) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Channel
	raw, err := dec.DecodeRaw()
	if err != nil {
		return errors.Wrap(err, "failed to read raw channel msgpack")
	}
	if err = msgpack.Unmarshal(raw, (*alias)(c)); err != nil {
		return errors.Wrap(err, "failed to decode channel from msgpack")
	}
	if c.Leaseholder == 0 {
		var legacy struct {
			NodeID node.Key `msgpack:"node_id"`
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return errors.Wrap(err, "failed to decode legacy node_id from msgpack")
		}
		c.Leaseholder = legacy.NodeID
	}
	return nil
}
