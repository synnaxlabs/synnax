// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// log.
func (CreatePayload) Handle(Log) (Log, error) {
	return Log{}, errors.Wrap(
		validate.ErrValidation, "only the Core dispatches a log create",
	)
}

// Handle replaces the log's name.
func (p RenamePayload) Handle(state Log) (Log, error) {
	state.Name = p.Name
	return state, nil
}

// Handle appends a default channel entry for the given channel. No-op when the
// channel already has an entry. The inverse RemoveChannel undoes the append
// without preserving the original slice position, so a remove-then-undo cycle
// moves the channel to the end of the list.
func (p AddChannelPayload) Handle(state Log) (Log, error) {
	if channelEntryIndex(state.Channels, p.Channel) != -1 {
		return state, nil
	}
	state.Channels = append(state.Channels, defaultChannelEntry(p.Channel))
	return state, nil
}

// Handle removes the entry for the given channel. No-op when the channel has no
// entry.
func (p RemoveChannelPayload) Handle(state Log) (Log, error) {
	state.Channels = slices.DeleteFunc(state.Channels, func(e ChannelEntry) bool {
		return e.Channel == p.Channel
	})
	return state, nil
}

// Handle inserts the entry if no entry references the same channel, otherwise
// replaces the existing entry in place.
func (p SetChannelEntryPayload) Handle(state Log) (Log, error) {
	if i := channelEntryIndex(state.Channels, p.Entry.Channel); i != -1 {
		state.Channels[i] = p.Entry
		return state, nil
	}
	state.Channels = append(state.Channels, p.Entry)
	return state, nil
}

// Handle sets the display color of the entry referencing the channel. No-op when
// no entry references it.
func (p SetChannelColorPayload) Handle(state Log) (Log, error) {
	if i := channelEntryIndex(state.Channels, p.Channel); i != -1 {
		state.Channels[i].Color = p.Color
	}
	return state, nil
}

// Handle sets the numeric notation of the entry referencing the channel. No-op
// when no entry references it.
func (p SetChannelNotationPayload) Handle(state Log) (Log, error) {
	if i := channelEntryIndex(state.Channels, p.Channel); i != -1 {
		state.Channels[i].Notation = p.Notation
	}
	return state, nil
}

// Handle sets the display precision of the entry referencing the channel, or clears it
// when the payload carries none. No-op when no entry references the channel.
func (p SetChannelPrecisionPayload) Handle(state Log) (Log, error) {
	if i := channelEntryIndex(state.Channels, p.Channel); i != -1 {
		state.Channels[i].Precision = p.Precision
	}
	return state, nil
}

// Handle sets the human-readable alias of the entry referencing the channel.
// No-op when no entry references it.
func (p SetChannelAliasPayload) Handle(state Log) (Log, error) {
	if i := channelEntryIndex(state.Channels, p.Channel); i != -1 {
		state.Channels[i].Alias = p.Alias
	}
	return state, nil
}

// Handle sets the format of the values of the timestamp channel. No-op when no entry
// references it.
func (p SetChannelValueFormatPayload) Handle(state Log) (Log, error) {
	if i := channelEntryIndex(state.Channels, p.Channel); i != -1 {
		state.Channels[i].ValueFormat.Format = p.Format
	}
	return state, nil
}

// Handle sets the time zone of the values of the timestamp channel. No-op when no
// entry references it.
func (p SetChannelValueTzPayload) Handle(state Log) (Log, error) {
	if i := channelEntryIndex(state.Channels, p.Channel); i != -1 {
		state.Channels[i].ValueFormat.Tz = p.Tz
	}
	return state, nil
}

// Handle repoints the entry referencing From at To in place, preserving the entry's
// position and display configuration. No-op when no entry references From or an entry
// already references To.
func (p SwapChannelPayload) Handle(state Log) (Log, error) {
	if channelEntryIndex(state.Channels, p.To) != -1 {
		return state, nil
	}
	if i := channelEntryIndex(state.Channels, p.From); i != -1 {
		state.Channels[i].Channel = p.To
	}
	return state, nil
}

// Handle sets the log-level timestamp precision.
func (p SetTimestampPrecisionPayload) Handle(state Log) (Log, error) {
	state.TimestampPrecision = p.TimestampPrecision
	return state, nil
}

// Handle controls whether channel names are hidden.
func (p SetChannelNamesHiddenPayload) Handle(state Log) (Log, error) {
	state.ChannelNamesHidden = p.ChannelNamesHidden
	return state, nil
}

// Handle controls whether line times are hidden.
func (p SetTimestampsHiddenPayload) Handle(state Log) (Log, error) {
	state.TimestampsHidden = p.TimestampsHidden
	return state, nil
}

// Handle sets the time zone of the line times and day rows.
func (p SetTzPayload) Handle(state Log) (Log, error) {
	state.Tz = p.Tz
	return state, nil
}

// channelEntryIndex returns the index of the entry referencing the given channel,
// or -1 when no entry references it.
func channelEntryIndex(entries []ChannelEntry, key channel.Key) int {
	return slices.IndexFunc(
		entries,
		func(e ChannelEntry) bool { return e.Channel == key },
	)
}

// defaultChannelEntry builds a channel entry with the schema's default display
// configuration. A zero color renders in the theme text color.
func defaultChannelEntry(key channel.Key) ChannelEntry {
	e := ChannelEntry{Channel: key}
	e.ApplyDefaults()
	return e
}
