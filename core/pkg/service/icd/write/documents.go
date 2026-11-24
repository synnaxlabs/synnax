// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package write

import (
	"fmt"
	"iter"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Documents builds the documents of one ICD message from the commands of its channel
// bindings. Document i of a frame holds command i of each bound channel, and the last
// command of a channel with fewer. A field with no command since the last Reset sends
// its default. Documents builds nothing until each field with no default has a command.
// A command that its field cannot hold is dropped, and the field keeps its last good
// command. Document i goes out only when one of its commands is good. Documents is not
// safe for concurrent use.
type Documents struct {
	name    string
	encoder *codec.JSONEncoder
	// channels holds the command channel of each binding.
	channels channel.Keys
	received []bool
	// last holds the last good command of each binding. Each one is non-nil, because
	// Encode writes the default for a nil command.
	last [][]byte
	// next holds the commands of the document to encode, nil for a binding with none.
	next [][]byte
	// commands holds the commands of each binding in one frame.
	commands [][][]byte
}

// NewDocuments returns the Documents of wm, the message of ic that a task sends.
// channels holds the command channels of wm, and name names the sender in each skip
// description. NewDocuments returns a validation error when a command channel does not
// exist or wm does not build from ic.
func NewDocuments(
	wm icd.WriteMessage,
	ic icd.ICD,
	channels map[channel.Key]channel.Channel,
	name string,
) (*Documents, error) {
	sampled := ChannelBindings(wm)
	n := len(sampled)
	d := &Documents{
		name:     name,
		channels: make(channel.Keys, n),
		received: make([]bool, n),
		last:     make([][]byte, n),
		next:     make([][]byte, n),
		commands: make([][][]byte, n),
	}
	dataTypes := make(map[channel.Key]telem.DataType, n)
	for i, b := range sampled {
		ch, ok := channels[b.Channel]
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation, "channel %d does not exist", b.Channel,
			)
		}
		dataTypes[b.Channel] = ch.DataType
		d.channels[i] = b.Channel
		d.last[i] = []byte{}
	}
	var err error
	if d.encoder, err = newEncoder(wm, ic, dataTypes); err != nil {
		return nil, err
	}
	return d, nil
}

// ChannelBindings returns the channel bindings of wm, in binding order.
func ChannelBindings(wm icd.WriteMessage) []icd.ChannelWriteBinding {
	return lo.FilterMap(wm.Bindings, func(b icd.WriteBinding, _ int) (
		icd.ChannelWriteBinding,
		bool,
	) {
		cb, ok := b.Variant.(icd.ChannelWriteBinding)
		return cb, ok
	})
}

// Channels returns the command channel of each channel binding, in binding order.
func (d *Documents) Channels() channel.Keys { return d.channels }

// Reset forgets every command.
func (d *Documents) Reset() { clear(d.received) }

// From returns the documents of fr, each with its row: the index in fr of the commands
// that it holds. It counts each command in skips as read, and each command or document
// that it drops as skipped.
func (d *Documents) From(fr framer.Frame, skips *driver.Skips) iter.Seq2[int, []byte] {
	return func(yield func(int, []byte) bool) {
		for i := range d.gather(fr) {
			good := false
			for b, commands := range d.commands {
				if i >= len(commands) {
					continue
				}
				skips.Read()
				if err := d.encoder.Check(b, commands[i]); err != nil {
					d.skip(skips, err)
					continue
				}
				d.last[b] = append(d.last[b][:0], commands[i]...)
				d.received[b], good = true, true
			}
			if !good || !d.fill() {
				continue
			}
			doc, err := d.encoder.Encode(nil, d.next)
			if err != nil {
				d.skip(skips, err)
				continue
			}
			if !yield(i, doc) {
				return
			}
		}
	}
}

// skip counts in skips a command that err dropped.
func (d *Documents) skip(skips *driver.Skips, err error) {
	skips.Skip(time.Now(), fmt.Sprintf("Command for %s: %s", d.name, err))
}

// gather fills commands with the commands of each binding in fr, and returns the count
// of documents that they make.
func (d *Documents) gather(fr framer.Frame) int {
	for b := range d.commands {
		d.commands[b] = d.commands[b][:0]
	}
	for key, s := range fr.Entries() {
		for b, ch := range d.channels {
			if ch == key {
				d.commands[b] = slices.AppendSeq(d.commands[b], s.Samples())
			}
		}
	}
	n := 0
	for _, commands := range d.commands {
		n = max(n, len(commands))
	}
	return n
}

// fill fills next with the last good command of each binding, and reports whether each
// field with no default has one.
func (d *Documents) fill() bool {
	ready := true
	for b := range d.next {
		d.next[b] = nil
		if d.received[b] {
			d.next[b] = d.last[b]
		} else if !d.encoder.HasDefault(b) {
			ready = false
		}
	}
	return ready
}
