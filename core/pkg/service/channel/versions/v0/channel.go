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
	"encoding/json"
	"fmt"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
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

// IsCalculated returns true if the channel is a calculated channel, false otherwise.
func (c Channel) IsCalculated() bool { return c.Expression != "" }

// Key returns the key for the Channel.
func (c Channel) Key() Key { return channel.NewKey(c.Leaseholder, c.LocalKey) }

// Index returns the key for the Channel's index channel.
func (c Channel) Index() Key {
	if c.LocalIndex == 0 {
		return 0
	}
	return channel.NewKey(c.Leaseholder, c.LocalIndex)
}

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

// Distribution returns the minimal distribution-layer representation of the channel,
// carrying only the storage and routing metadata the distribution layer needs.
func (c Channel) Distribution() channel.Channel {
	return channel.Channel{
		Name:        c.Name,
		Leaseholder: c.Leaseholder,
		DataType:    c.DataType,
		IsIndex:     c.IsIndex,
		LocalKey:    c.LocalKey,
		LocalIndex:  c.LocalIndex,
		Virtual:     c.Virtual,
		Concurrency: c.Concurrency,
	}
}

// Equals returns true if the two channels are meaningfully equal to each other. If the
// exclude parameter is provided, the function will ignore the fields specified in the
// exclude parameter.
func (c Channel) Equals(other Channel, exclude ...string) bool {
	comparisons := []struct {
		field string
		equal bool
	}{
		{"Name", c.Name == other.Name},
		{"Leaseholder", c.Leaseholder == other.Leaseholder},
		{"DataType", c.DataType == other.DataType},
		{"IsIndex", c.IsIndex == other.IsIndex},
		{"LocalKey", c.LocalKey == other.LocalKey},
		{"LocalIndex", c.LocalIndex == other.LocalIndex},
		{"Virtual", c.Virtual == other.Virtual},
		{"Concurrency", c.Concurrency == other.Concurrency},
		{"Internal", c.Internal == other.Internal},
		{"Expression", c.Expression == other.Expression},
	}
	for _, comp := range comparisons {
		if !comp.equal && !slices.Contains(exclude, comp.field) {
			return false
		}
	}
	if !slices.Contains(exclude, "Operations") {
		if !slices.Equal(c.Operations, other.Operations) {
			return false
		}
	}
	return true
}

// String implements stringer, returning a nicely formatted string representation of the
// Channel.
func (c Channel) String() string {
	if c.Name != "" {
		return fmt.Sprintf("[%s]<%d>", c.Name, c.Key())
	}
	return fmt.Sprintf("<%d>", c.Key())
}

// UnmarshalJSON implements json.Unmarshaler, supporting both legacy "node_id" and new
// "leaseholder" field names for backward compatibility.
func (c *Channel) UnmarshalJSON(data []byte) error {
	type alias Channel
	if err := json.Unmarshal(data, (*alias)(c)); err != nil {
		return errors.Wrap(err, "failed to decode channel from JSON")
	}
	if c.Leaseholder == 0 {
		var legacy struct {
			NodeID node.Key `json:"node_id"`
		}
		if err := json.Unmarshal(data, &legacy); err != nil {
			return errors.Wrap(err, "failed to decode legacy node_id from JSON")
		}
		c.Leaseholder = legacy.NodeID
	}
	return nil
}

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

// OntologyID returns the ontology.ID for the channel.
func (c Channel) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeChannel, Key: c.Key().String()}
}

// ToPayload returns a map representation of the channel for use in ontology resources
// and signal marshaling. The "operations" key is omitted when the channel has no
// operations.
func (c Channel) ToPayload() map[string]any {
	p := map[string]any{
		"key":         c.Key(),
		"name":        c.Name,
		"leaseholder": c.Leaseholder,
		"is_index":    c.IsIndex,
		"index":       c.Index(),
		"data_type":   c.DataType,
		"internal":    c.Internal,
		"virtual":     c.Virtual,
		"expression":  c.Expression,
	}
	if len(c.Operations) > 0 {
		p["operations"] = c.Operations
	}
	return p
}
