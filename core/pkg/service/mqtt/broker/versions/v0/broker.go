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
	"strconv"

	"github.com/synnaxlabs/gorp"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ gorp.Entry[Key] = Broker{}

// CustomTypeName gives brokers a table prefix that cannot collide with the legacy
// device table or another integration's resource.
func (Broker) CustomTypeName() string { return "mqtt_broker" }

// GorpKey returns the key of the broker.
func (b Broker) GorpKey() Key { return b.Key }

// ResourceID returns the resource ID of the broker.
func (b Broker) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeMqttBroker, Key: b.Key.String()}
}

type sparkplugTag struct{ group, edgeNode, device, tag string }

// ValidateWrite returns the error of Validate. It also refuses a keep alive of a
// fraction of a second, a Sparkplug group that cannot be a topic level, and a binding
// with the natural key of another binding in its list. Each error wraps
// validate.ErrValidation. Writes check it, so stored brokers that break it stay
// readable.
func (b Broker) ValidateWrite() error {
	if err := b.Validate(); err != nil {
		return err
	}
	v := validate.New("Broker")
	v.Ternary(
		"keep_alive",
		b.KeepAlive%telem.Second != 0,
		"must be a whole number of seconds",
	)
	for i, g := range b.Sparkplug.Groups {
		v.Exec(func() error {
			return validate.PathedError(
				sparkplug.ValidateID("group", g),
				"sparkplug", "groups", strconv.Itoa(i),
			)
		})
	}
	validateReads(v, b.Reads)
	validateWrites(v, b.Writes)
	validateEdgeCommands(v, b.EdgeCommands)
	return v.Error()
}

func validateReads(v *validate.Validator, reads []Read) {
	type plainKey struct {
		topic   string
		message icd.MessageKey
		pointer string
	}
	plain := make(set.Set[plainKey], len(reads))
	tags := make(set.Set[sparkplugTag], len(reads))
	for i, r := range reads {
		path := "reads." + strconv.Itoa(i)
		switch r := r.Variant.(type) {
		case PlainRead:
			k := plainKey{topic: r.Topic, message: r.Message, pointer: r.Pointer}
			v.Ternaryf(
				path,
				plain.Contains(k),
				"duplicates the read of %q from topic %s",
				r.Pointer,
				r.Topic,
			)
			plain.Add(k)
		case SparkplugRead:
			k := sparkplugTag{r.Group, r.EdgeNode, r.Device, r.Tag}
			v.Ternaryf(path, tags.Contains(k), "duplicates the read of tag %s", r.Tag)
			tags.Add(k)
		}
	}
}

func validateWrites(v *validate.Validator, writes []Write) {
	type plainKey struct {
		topic string
		field icd.FieldKey
	}
	plain := make(set.Set[plainKey], len(writes))
	tags := make(set.Set[sparkplugTag], len(writes))
	for i, w := range writes {
		path := "writes." + strconv.Itoa(i)
		switch w := w.Variant.(type) {
		case PlainWrite:
			k := plainKey{topic: w.Topic, field: w.Field}
			v.Ternaryf(
				path,
				plain.Contains(k),
				"duplicates the write to topic %s",
				w.Topic,
			)
			plain.Add(k)
		case SparkplugWrite:
			k := sparkplugTag{w.Group, w.EdgeNode, w.Device, w.Tag}
			v.Ternaryf(path, tags.Contains(k), "duplicates the write to tag %s", w.Tag)
			tags.Add(k)
		}
	}
}

func validateEdgeCommands(v *validate.Validator, commands []EdgeCommand) {
	seen := make(set.Set[sparkplugTag], len(commands))
	for i, c := range commands {
		k := sparkplugTag{group: c.Group, edgeNode: c.EdgeNode, tag: c.Tag}
		v.Ternaryf(
			"edge_commands."+strconv.Itoa(i),
			seen.Contains(k),
			"duplicates the command of tag %s",
			c.Tag,
		)
		seen.Add(k)
	}
}
