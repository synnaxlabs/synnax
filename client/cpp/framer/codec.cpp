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

Codec::Codec(
    const std::vector<telem::DataType> &data_types,
    const std::vector<ChannelKey> &channels
) {
    std::vector<std::pair<ChannelKey, telem::DataType>> pairs;
    pairs.reserve(channels.size());
    for (size_t i = 0; i < channels.size(); i++)
        pairs.emplace_back(channels[i], data_types[i]);
    std::sort(pairs.begin(), pairs.end(), [](const auto &a, const auto &b) {
        return a.first < b.first;
    });
    this->keys.reserve(channels.size());
    this->key_data_types.reserve(channels.size());
    for (const auto &[key, data_type]: pairs) {
        this->keys.push_back(key);
        this->key_data_types[key] = data_type;
        if (data_type.is_variable()) this->has_variable_data_types = true;
    }
}

Codec::Codec(const std::vector<Channel> &channels) {
    this->key_data_types.reserve(channels.size());
    this->keys.reserve(channels.size());
    for (const auto &ch: channels) {
        this->key_data_types[ch.key] = ch.data_type;
        if (ch.data_type.is_variable()) this->has_variable_data_types = true;
        this->keys.push_back(ch.key);
    }
    std::sort(this->keys.begin(), this->keys.end());
}

void Codec::encode(
    const Frame &frame,
    const size_t start_offset,
    std::vector<uint8_t> &data
) {
    this->sorting_indices.resize(frame.size());
    for (size_t i = 0; i < frame.channels->size(); i++)
        this->sorting_indices[i] = {frame.channels->at(i), i};
    std::sort(sorting_indices.begin(), sorting_indices.end());

    CodecFlags flags;
    flags.equal_lens = !this->has_variable_data_types;
    size_t cur_data_size = -1;
    telem::TimeRange ref_tr = {};
    uint64_t ref_alignment = 0;

    size_t byte_array_size = start_offset + 1;

    if (frame.channels->size() != keys.size()) {
        flags.all_channels_present = false;
        byte_array_size += frame.channels->size() * 4; // 4 bytes per channel key
    }

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

    if (!flags.time_ranges_zero)
        if (!flags.equal_time_ranges)
            byte_array_size += frame.channels->size() * 16;
        else
            byte_array_size += 16;

    if (!flags.zero_alignments) {
        if (!flags.equal_alignments)
            byte_array_size += frame.channels->size() * 8;
        else
            byte_array_size += 8;
    }

    binary::Writer buf(data, byte_array_size, start_offset);
    buf.uint8(flags.encode());

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
}

Frame Codec::decode(const std::vector<uint8_t> &data) const {
    auto reader = binary::Reader(data);
    Frame frame;
    uint32_t data_len = 0;
    telem::TimeRange ref_tr = {};
    uint64_t ref_alignment = 0;
    auto flags = CodecFlags::decode(reader.uint8());
    if (flags.equal_lens) data_len = reader.uint32();

    if (flags.equal_time_ranges && !flags.time_ranges_zero) {
        ref_tr.start = telem::TimeStamp(reader.int64());
        ref_tr.end = telem::TimeStamp(reader.int64());
    } else if (flags.time_ranges_zero)
        ref_tr = telem::TimeRange{telem::TimeStamp(0), telem::TimeStamp(0)};

    if (flags.equal_alignments && !flags.zero_alignments) ref_alignment = reader.uint64();

    auto decode_series = [&](const ChannelKey key) {
        uint32_t local_data_len = data_len;
        if (!flags.equal_lens) local_data_len = reader.uint32();

        const auto it = key_data_types.find(key);
        if (it == key_data_types.end())
            throw std::runtime_error("Unknown channel key: " + std::to_string(key));

        auto s = telem::Series(it->second, local_data_len);
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
        frame.reserve(this->keys.size());
        for (const auto &key: this->keys)
            decode_series(key);
    } else
        while (decode_series(reader.uint32())) {}

    return frame;
}
}
