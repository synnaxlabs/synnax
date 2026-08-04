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
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Oracle does not emit Go struct defaults, so the defaults a newly added channel entry
// receives are duplicated here. They must stay in sync with the field defaults declared
// on ChannelEntry in schemas/synnax/log.oracle and with the TypeScript
// defaultChannelEntry in client/ts/src/log/actions.ts.
const (
	defaultPrecision         int32                 = -1
	defaultNotation          notation.Notation     = notation.NotationStandard
	defaultTimestampFormat   telem.TimestampFormat = telem.TimestampFormatPreciseDate
	defaultTimestampTimeZone telem.TimeZone        = telem.TimeZoneLocal
)

// minTimestampPrecision and maxTimestampPrecision bound the log-level timestamp
// precision. They mirror the @validate constraints on Log.timestamp_precision in
// schemas/synnax/log.oracle.
const (
	minTimestampPrecision int32 = 0
	maxTimestampPrecision int32 = 3
)

// minChannelPrecision and maxChannelPrecision bound a channel entry's display
// precision. They mirror the @validate constraints on ChannelEntry.precision in
// schemas/synnax/log.oracle.
const (
	minChannelPrecision int32 = -1
	maxChannelPrecision int32 = 17
)

// Handle replaces the document with its created state.
func (p CreatePayload) Handle(Log) (Log, error) {
	return p.Log, nil
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

// Handle sets the display precision of the entry referencing the channel. It
// returns validate.ErrValidation when the precision is outside the inclusive
// range [-1, 17], and is a no-op when no entry references the channel.
func (p SetChannelPrecisionPayload) Handle(state Log) (Log, error) {
	if p.Precision < minChannelPrecision || p.Precision > maxChannelPrecision {
		return Log{}, errors.Wrapf(
			validate.ErrValidation,
			"channel precision %d out of range [%d, %d]",
			p.Precision,
			minChannelPrecision,
			maxChannelPrecision,
		)
	}
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

// Handle sets the timestamp render format of the entry referencing the channel.
// No-op when no entry references it.
func (p SetChannelTimestampFormatPayload) Handle(state Log) (Log, error) {
	if i := channelEntryIndex(state.Channels, p.Channel); i != -1 {
		state.Channels[i].Timestamp.Format = p.Format
	}
	return state, nil
}

// Handle sets the timestamp time zone of the entry referencing the channel.
// No-op when no entry references it.
func (p SetChannelTimestampTzPayload) Handle(state Log) (Log, error) {
	if i := channelEntryIndex(state.Channels, p.Channel); i != -1 {
		state.Channels[i].Timestamp.Tz = p.Tz
	}
	return state, nil
}

// Handle replaces the entire ordered list of channel entries.
func (p SetChannelsPayload) Handle(state Log) (Log, error) {
	state.Channels = p.Channels
	return state, nil
}

// Handle repoints the entry referencing From at To in place, preserving the entry's
// position and display configuration. No-op when no entry references From.
func (p SwapChannelPayload) Handle(state Log) (Log, error) {
	if i := channelEntryIndex(state.Channels, p.From); i != -1 {
		state.Channels[i].Channel = p.To
	}
	return state, nil
}

// Handle sets the log-level timestamp precision. It returns validate.ErrValidation
// when the precision is outside the inclusive range [0, 3].
func (p SetTimestampPrecisionPayload) Handle(state Log) (Log, error) {
	if p.TimestampPrecision < minTimestampPrecision || p.TimestampPrecision > maxTimestampPrecision {
		return Log{}, errors.Wrapf(
			validate.ErrValidation,
			"timestamp precision %d out of range [%d, %d]",
			p.TimestampPrecision,
			minTimestampPrecision,
			maxTimestampPrecision,
		)
	}
	state.TimestampPrecision = p.TimestampPrecision
	return state, nil
}

// Handle controls whether channel names are hidden.
func (p SetHideChannelNamesPayload) Handle(state Log) (Log, error) {
	state.HideChannelNames = p.HideChannelNames
	return state, nil
}

// Handle controls whether the receipt timestamp column is hidden.
func (p SetHideReceiptTimestampPayload) Handle(state Log) (Log, error) {
	state.HideReceiptTimestamp = p.HideReceiptTimestamp
	return state, nil
}

// channelEntryIndex returns the index of the entry referencing the given channel,
// or -1 when no entry references it.
func channelEntryIndex(entries []ChannelEntry, key channel.Key) int {
	return slices.IndexFunc(entries, func(e ChannelEntry) bool { return e.Channel == key })
}

// defaultChannelEntry builds a channel entry with the schema's default display
// configuration. Color is left as the zero value; the rendering layer resolves a
// palette color for entries without an explicit color.
func defaultChannelEntry(key channel.Key) ChannelEntry {
	return ChannelEntry{
		Channel:   key,
		Notation:  defaultNotation,
		Precision: defaultPrecision,
		Timestamp: TimestampConfig{
			Format: defaultTimestampFormat,
			Tz:     defaultTimestampTimeZone,
		},
	}
}
