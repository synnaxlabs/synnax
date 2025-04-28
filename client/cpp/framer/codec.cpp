// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// For detailed information about the specifications,
// please refer to the official RFC 0016 document.
// Document here: docs/tech/rfc/0016-231001-frame-flight-protocol.md

/// std
#include <algorithm>
#include <memory>
#include <sstream>
#include <vector>

/// internal
#include "client/cpp/framer/framer.h"

namespace synnax {
uint8_t CodecFlags::encode() const {
    uint8_t b = 0;
    b = binary::set_bit(b, FlagPosition::EqualLengths, equal_lens);
    b = binary::set_bit(b, FlagPosition::EqualTimeRanges, equal_time_ranges);
    b = binary::set_bit(b, FlagPosition::TimeRangesZero, time_ranges_zero);
    b = binary::set_bit(b, FlagPosition::AllChannelsPresent, all_channels_present);
    b = binary::set_bit(b, FlagPosition::EqualAlignments, equal_alignments);
    b = binary::set_bit(b, FlagPosition::ZeroAlignments, zero_alignments);
    return b;
}

CodecFlags CodecFlags::decode(const uint8_t b) {
    CodecFlags f;
    f.equal_lens = binary::get_bit(b, FlagPosition::EqualLengths);
    f.equal_time_ranges = binary::get_bit(b, FlagPosition::EqualTimeRanges);
    f.time_ranges_zero = binary::get_bit(b, FlagPosition::TimeRangesZero);
    f.all_channels_present = binary::get_bit(b, FlagPosition::AllChannelsPresent);
    f.equal_alignments = binary::get_bit(b, FlagPosition::EqualAlignments);
    f.zero_alignments = binary::get_bit(b, FlagPosition::ZeroAlignments);
    return f;
}

xerrors::Error Codec::update(const std::vector<ChannelKey> &keys) {
    this->seq_num++;
    auto [channels, err] = this->channel_client.retrieve(keys);
    if (err) return err;
    Codec::State state;
    for (const auto &ch: channels) {
        state.keys.insert(ch.key);
        state.key_data_types[ch.key] = ch.data_type;
        if (ch.data_type.is_variable()) state.has_variable_data_types = true;
    }
    this->states[seq_num] = state;
    return xerrors::NIL;
}

Codec::Codec(
    const std::vector<ChannelKey> &channels,
    const std::vector<telem::DataType> &data_types
):
    seq_num(1), channel_client(nullptr, nullptr) {
    Codec::State state;
    state.key_data_types.reserve(channels.size());
    for (auto i = 0; i < channels.size(); i++) {
        auto k = channels[i];
        auto dt = data_types[i];
        state.keys.insert(k);
        state.key_data_types[k] = dt;
        if (dt.is_variable()) state.has_variable_data_types = true;
    }
    this->states[this->seq_num] = state;
}

xerrors::Error Codec::encode(const Frame &frame, std::vector<uint8_t> &output) {
    CodecFlags flags;
    size_t byte_array_size = 1 + 4;

    auto state = this->states[this->seq_num];

    if (frame.channels->size() != state.keys.size()) {
        flags.all_channels_present = false;
        byte_array_size += frame.channels->size() * 4; // 4 bytes per channel key
    }

    this->sorting_indices.resize(frame.size());
    for (size_t i = 0; i < frame.channels->size(); i++) {
        auto k = frame.channels->at(i);
        if (!state.keys.contains(k))
            return xerrors::Error(
                xerrors::VALIDATION,
                "frame contains extra key " + std::to_string(k) +
                    "not provided when opening the writer"
            );
        this->sorting_indices[i] = {k, i};
    }
    std::sort(sorting_indices.begin(), sorting_indices.end());

    flags.equal_lens = !state.has_variable_data_types;
    size_t cur_data_size = -1;
    telem::TimeRange ref_tr = {};
    uint64_t ref_alignment = 0;

    for (const auto &[key, idx]: sorting_indices) {
        const telem::Series &series = frame.series->at(idx);
        byte_array_size += series.byte_size();
        if (cur_data_size == -1) {
            cur_data_size = series.size();
            ref_tr = series.time_range;
            ref_alignment = series.alignment;
            continue;
        }
        if (series.size() != cur_data_size) flags.equal_lens = false;
        if (series.time_range != ref_tr) flags.equal_time_ranges = false;
        if (series.alignment != ref_alignment) flags.equal_alignments = false;
    }

    flags.time_ranges_zero = flags.equal_time_ranges &&
                             ref_tr.start.nanoseconds() == 0 &&
                             ref_tr.end.nanoseconds() == 0;

    flags.zero_alignments = flags.equal_alignments && ref_alignment == 0;

    if (!flags.equal_lens)
        byte_array_size += frame.channels->size() * 4;
    else
        byte_array_size += 4;

    if (!flags.time_ranges_zero) {
        if (!flags.equal_time_ranges)
            byte_array_size += frame.channels->size() * 16;
        else
            byte_array_size += 16;
    }

    if (!flags.zero_alignments) {
        if (!flags.equal_alignments)
            byte_array_size += frame.channels->size() * 8;
        else
            byte_array_size += 8;
    }

    binary::Writer buf(output, byte_array_size);
    buf.uint8(flags.encode());
    buf.uint32(this->seq_num);

    if (flags.equal_lens) { buf.uint32(static_cast<uint32_t>(cur_data_size)); }

    if (flags.equal_time_ranges && !flags.time_ranges_zero) {
        buf.int64(ref_tr.start.nanoseconds());
        buf.int64(ref_tr.end.nanoseconds());
    }

    if (flags.equal_alignments && !flags.zero_alignments) buf.uint64(ref_alignment);

    for (const auto &[key, idx]: sorting_indices) {
        const telem::Series &ser = frame.series->at(idx);
        const auto byte_size = ser.byte_size();
        if (!flags.all_channels_present) buf.uint32(key);
        if (!flags.equal_lens) {
            const auto size = ser.data_type().is_variable() ? byte_size : ser.size();
            buf.uint32(static_cast<uint32_t>(size));
        }
        buf.write(ser.data(), byte_size);
        if (!flags.equal_time_ranges) {
            buf.int64(ser.time_range.start.nanoseconds());
            buf.int64(ser.time_range.end.nanoseconds());
        }
        if (!flags.equal_alignments) buf.uint64(ser.alignment);
    }

    return xerrors::NIL;
}

std::pair<Frame, xerrors::Error> Codec::decode(const std::vector<uint8_t> &data) const {
    return this->decode(data.data(), data.size());
}


std::pair<Frame, xerrors::Error>
Codec::decode(const uint8_t *data, const size_t size) const {
    auto reader = binary::Reader(data, size);
    Frame frame;
    uint32_t data_len = 0;
    telem::TimeRange ref_tr = {};
    uint64_t ref_alignment = 0;
    auto flags = CodecFlags::decode(reader.uint8());

    auto seq_num = reader.uint32();
    auto state = this->states.at(seq_num);

    if (flags.equal_lens) data_len = reader.uint32();

    if (flags.equal_time_ranges && !flags.time_ranges_zero) {
        ref_tr.start = telem::TimeStamp(reader.int64());
        ref_tr.end = telem::TimeStamp(reader.int64());
    } else if (flags.time_ranges_zero)
        ref_tr = telem::TimeRange{telem::TimeStamp(0), telem::TimeStamp(0)};

    if (flags.equal_alignments && !flags.zero_alignments)
        ref_alignment = reader.uint64();

    auto decode_series = [&](const ChannelKey key) {
        // when the series is a variable data type, we use its byte capacity instead
        // of its length.
        uint32_t local_data_len_or_byte_cap = data_len;
        if (!flags.equal_lens) local_data_len_or_byte_cap = reader.uint32();

        const auto it = state.key_data_types.find(key);
        if (it == state.key_data_types.end())
            throw std::runtime_error("Unknown channel key: " + std::to_string(key));

        auto s = telem::Series(it->second, local_data_len_or_byte_cap);
        s.time_range = ref_tr;
        s.alignment = ref_alignment;

        s.fill_from(reader);

        if (!flags.equal_time_ranges) {
            s.time_range.start = telem::TimeStamp(reader.int64());
            s.time_range.end = telem::TimeStamp(reader.int64());
        }

        if (!flags.equal_alignments) s.alignment = reader.uint64();

        if (frame.channels == nullptr) {
            frame.channels = std::make_unique<std::vector<ChannelKey>>();
            frame.series = std::make_unique<std::vector<telem::Series>>();
        }

        frame.emplace(key, std::move(s));
        return !reader.exhausted();
    };

    if (flags.all_channels_present) {
        frame.reserve(state.keys.size());
        for (const auto &key: state.keys)
            decode_series(key);
    } else
        while (decode_series(reader.uint32())) {}

    return {std::move(frame), xerrors::NIL};
}
}
