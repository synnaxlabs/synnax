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
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Payloads builds the payloads of one ICD message from the commands of its channel
// bindings. Payload i of a frame holds command i of each bound channel, and the last
// command of a channel with fewer. A field with no command since the last Reset sends
// its default. Payloads builds nothing until each field with no default has a command.
// A command that its field cannot hold is dropped, and the field keeps its last good
// command. Payload i goes out only when one of its commands is good. Payloads is not
// safe for concurrent use.
type Payloads struct {
	name    string
	encoder *codec.JSONEncoder
	// channels holds the command channel of each binding.
	channels channel.Keys
	received []bool
	// last holds the last good command of each binding. Each one is non-nil, because
	// Encode writes the default for a nil command.
	last [][]byte
	// next holds the commands of the payload to encode, nil for a binding with none.
	next [][]byte
	// commands holds the commands of each binding in one frame.
	commands [][][]byte
	// size is the capacity of the next payload: the length of the last one, and an
	// eighth more for values that grow.
	size int
}

// NewPayloads returns the Payloads of wm, the message of ic that a task sends. channels
// holds the command channels of wm, and name names the sender in each skip description.
// NewPayloads returns a validation error when a command channel does not exist or wm
// does not build from ic.
func NewPayloads(
	wm icd.WriteMessage,
	ic icd.ICD,
	channels map[channel.Key]icdv0.Channel,
	name string,
) (*Payloads, error) {
	sampled := ChannelBindings(wm)
	n := len(sampled)
	p := &Payloads{
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
		p.channels[i] = b.Channel
		p.last[i] = []byte{}
	}
	var err error
	if p.encoder, err = newEncoder(wm, ic, dataTypes); err != nil {
		return nil, err
	}
	return p, nil
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
func (p *Payloads) Channels() channel.Keys { return p.channels }

// Reset forgets every command.
func (p *Payloads) Reset() { clear(p.received) }

// From returns the payloads of fr, each with its row: the index in fr of the commands
// that it holds. It counts each command in skips as read, and each command or payload
// that it drops as skipped.
func (p *Payloads) From(fr framer.Frame, skips *driver.Skips) iter.Seq2[int, []byte] {
	return func(yield func(int, []byte) bool) {
		for i := range p.gather(fr) {
			good := false
			for b, commands := range p.commands {
				if i >= len(commands) {
					continue
				}
				skips.Read()
				if err := p.encoder.Check(b, commands[i]); err != nil {
					p.skip(skips, err)
					continue
				}
				p.last[b] = append(p.last[b][:0], commands[i]...)
				p.received[b], good = true, true
			}
			if !good || !p.fill() {
				continue
			}
			payload, err := p.encoder.Encode(make([]byte, 0, p.size), p.next)
			if err != nil {
				p.skip(skips, err)
				continue
			}
			p.size = len(payload) + len(payload)/8
			if !yield(i, payload) {
				return
			}
		}
	}
}

// skip counts in skips a command that err dropped.
func (p *Payloads) skip(skips *driver.Skips, err error) {
	skips.Skip(time.Now(), fmt.Sprintf("Command for %s: %s", p.name, err))
}

// gather fills commands with the commands of each binding in fr, and returns the count
// of payloads that they make.
func (p *Payloads) gather(fr framer.Frame) int {
	for b := range p.commands {
		p.commands[b] = p.commands[b][:0]
	}
	for key, s := range fr.Entries() {
		for b, ch := range p.channels {
			if ch == key {
				p.commands[b] = slices.AppendSeq(p.commands[b], s.Samples())
			}
		}
	}
	n := 0
	for _, commands := range p.commands {
		n = max(n, len(commands))
	}
	return n
}

// fill fills next with the last good command of each binding, and reports whether each
// field with no default has one.
func (p *Payloads) fill() bool {
	ready := true
	for b := range p.next {
		p.next[b] = nil
		if p.received[b] {
			p.next[b] = p.last[b]
		} else if !p.encoder.HasDefault(b) {
			ready = false
		}
	}
	return ready
}
