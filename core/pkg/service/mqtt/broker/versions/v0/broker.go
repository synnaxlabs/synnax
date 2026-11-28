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
	"regexp"
	"strconv"
	"strings"

	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var sparkplugID = regexp.MustCompile("^[^/+#\x00]+$")

type sparkplugTag struct{ group, edgeNode, device, tag string }

// ValidateWrite returns the error of Validate. It also refuses a keep alive of a
// fraction of a second, a client ID that holds NUL, a Sparkplug group that cannot be a
// topic level or repeats another, and a binding with the natural key of another binding
// in its list. Each error wraps validate.ErrValidation. Writes check it, so stored
// brokers that break it stay readable.
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
	v.Ternary(
		"client_id",
		b.ClientID != nil && strings.ContainsRune(*b.ClientID, 0),
		"must not hold NUL",
	)
	groups := make(set.Set[string], len(b.Sparkplug.Groups))
	for i, g := range b.Sparkplug.Groups {
		path := "sparkplug.groups." + strconv.Itoa(i)
		v.Ternaryf(
			path,
			!sparkplugID.MatchString(g),
			"%q must not be empty or hold /, +, #, or NUL",
			g,
		)
		v.Ternaryf(path, groups.Contains(g), "duplicates the group %q", g)
		groups.Add(g)
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
