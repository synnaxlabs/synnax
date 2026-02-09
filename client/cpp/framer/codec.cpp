// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <memory>
#include <sstream>
#include <vector>

#include "client/cpp/framer/framer.h"

namespace synnax::framer {
uint8_t CodecFlags::encode() const {
    uint8_t b = 0;
    b = x::binary::set_bit(b, FlagPosition::EqualLengths, equal_lens);
    b = x::binary::set_bit(b, FlagPosition::EqualTimeRanges, equal_time_ranges);
    b = x::binary::set_bit(b, FlagPosition::TimeRangesZero, time_ranges_zero);
    b = x::binary::set_bit(b, FlagPosition::AllChannelsPresent, all_channels_present);
    b = x::binary::set_bit(b, FlagPosition::EqualAlignments, equal_alignments);
    b = x::binary::set_bit(b, FlagPosition::ZeroAlignments, zero_alignments);
    return b;
}

CodecFlags CodecFlags::decode(const uint8_t b) {
    CodecFlags f;
    f.equal_lens = x::binary::get_bit(b, FlagPosition::EqualLengths);
    f.equal_time_ranges = x::binary::get_bit(b, FlagPosition::EqualTimeRanges);
    f.time_ranges_zero = x::binary::get_bit(b, FlagPosition::TimeRangesZero);
    f.all_channels_present = x::binary::get_bit(b, FlagPosition::AllChannelsPresent);
    f.equal_alignments = x::binary::get_bit(b, FlagPosition::EqualAlignments);
    f.zero_alignments = x::binary::get_bit(b, FlagPosition::ZeroAlignments);
    return f;
}

x::errors::Error Codec::update(const std::vector<channel::Key> &keys) {
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
    return x::errors::NIL;
}

void Codec::throw_if_uninitialized() const {
    if (this->seq_num < 1) throw std::runtime_error("codec is uninitialized");
}

Codec::Codec(
    const std::vector<channel::Key> &channels,
    const std::vector<x::telem::DataType> &data_types
):
    seq_num(1), channel_client(nullptr, nullptr) {
    Codec::State state;
    state.key_data_types.reserve(channels.size());
    for (size_t i = 0; i < channels.size(); i++) {
        auto k = channels[i];
        const auto &dt = data_types[i];
        state.keys.insert(k);
        state.key_data_types[k] = dt;
        if (dt.is_variable()) state.has_variable_data_types = true;
    }
    this->states[this->seq_num] = state;
}

constexpr std::size_t ALIGNMENT_SIZE = 8;
constexpr std::size_t DATA_LENGTH_SIZE = 4;
constexpr std::size_t KEY_SIZE = 4;
constexpr std::size_t FLAGS_SIZE = 1;
constexpr std::size_t SEQ_NUM_SIZE = 4;
constexpr std::size_t TIME_RANGE_SIZE = 16;

x::errors::Error
Codec::encode(const x::telem::Frame &frame, std::vector<uint8_t> &output) {
    this->throw_if_uninitialized();
    CodecFlags flags;
    size_t byte_array_size = FLAGS_SIZE + SEQ_NUM_SIZE;

    auto state = this->states[this->seq_num];

    if (frame.channels->size() != state.keys.size()) {
        flags.all_channels_present = false;
        byte_array_size += frame.channels->size() * KEY_SIZE;
    }

    this->sorting_indices.resize(frame.size());
    for (size_t i = 0; i < frame.channels->size(); i++) {
        auto k = frame.channels->at(i);
        auto &ser = frame.series->at(i);
        auto dt = state.key_data_types.find(k);
        if (dt == state.key_data_types.end())
            return x::errors::Error(
                x::errors::VALIDATION,
                "frame contains extra key " + std::to_string(k) +
                    "not provided when opening the writer"
            );
        if (dt->second != ser.data_type())
            return x::errors::Error(
                x::errors::VALIDATION,
                "data type " + dt->second + " for channel + " + std::to_string(k) +
                    " does not match series data type " + ser.data_type()
            );
        this->sorting_indices[i] = {k, i};
    }
    std::sort(sorting_indices.begin(), sorting_indices.end());

    flags.equal_lens = !state.has_variable_data_types;
    size_t cur_data_size = 0;
    bool first_series = true;
    x::telem::TimeRange ref_tr = {};
    x::telem::Alignment ref_alignment;

    for (const auto &[key, idx]: sorting_indices) {
        const x::telem::Series &series = frame.series->at(idx);
        byte_array_size += series.byte_size();
        if (first_series) {
            cur_data_size = series.size();
            ref_tr = series.time_range;
            ref_alignment = series.alignment;
            first_series = false;
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
        byte_array_size += frame.channels->size() * DATA_LENGTH_SIZE;
    else
        byte_array_size += DATA_LENGTH_SIZE;

    if (!flags.time_ranges_zero) {
        if (!flags.equal_time_ranges)
            byte_array_size += frame.channels->size() * TIME_RANGE_SIZE;
        else
            byte_array_size += TIME_RANGE_SIZE;
    }

    if (!flags.zero_alignments) {
        if (!flags.equal_alignments)
            byte_array_size += frame.channels->size() * ALIGNMENT_SIZE;
        else
            byte_array_size += ALIGNMENT_SIZE;
    }

    x::binary::Writer buf(output, byte_array_size);

    if (buf.uint8(flags.encode()) != 1)
        return x::errors::Error(x::errors::UNEXPECTED, "failed to write flags");
    if (buf.uint32(this->seq_num) != 4)
        return x::errors::Error(
            x::errors::UNEXPECTED,
            "failed to write sequence number"
        );

    if (flags.equal_lens) {
        if (buf.uint32(static_cast<uint32_t>(cur_data_size)) != 4)
            return x::errors::Error(
                x::errors::UNEXPECTED,
                "failed to write data length"
            );
    }

    if (flags.equal_time_ranges && !flags.time_ranges_zero) {
        if (buf.int64(ref_tr.start.nanoseconds()) != 8)
            return x::errors::Error(
                x::errors::UNEXPECTED,
                "failed to write time range start"
            );
        if (buf.int64(ref_tr.end.nanoseconds()) != 8)
            return x::errors::Error(
                x::errors::UNEXPECTED,
                "failed to write time range end"
            );
    }

    if (flags.equal_alignments && !flags.zero_alignments) {
        if (buf.uint64(ref_alignment.uint64()) != 8)
            return x::errors::Error(x::errors::UNEXPECTED, "failed to write alignment");
    }

    for (const auto &[key, idx]: sorting_indices) {
        const x::telem::Series &ser = frame.series->at(idx);
        const auto byte_size = ser.byte_size();

        if (!flags.all_channels_present) {
            if (buf.uint32(key) != 4)
                return x::errors::Error(
                    x::errors::UNEXPECTED,
                    "failed to write channel key"
                );
        }

        if (!flags.equal_lens) {
            const auto size = ser.data_type().is_variable() ? byte_size : ser.size();
            if (buf.uint32(static_cast<uint32_t>(size)) != 4)
                return x::errors::Error(
                    x::errors::UNEXPECTED,
                    "failed to write series length"
                );
        }

        if (buf.write(ser.data(), byte_size) != byte_size)
            return x::errors::Error(
                x::errors::UNEXPECTED,
                "failed to write series data: expected " + std::to_string(byte_size) +
                    " bytes"
            );

        if (!flags.equal_time_ranges) {
            if (buf.int64(ser.time_range.start.nanoseconds()) != 8)
                return x::errors::Error(
                    x::errors::UNEXPECTED,
                    "failed to write series time range start"
                );
            if (buf.int64(ser.time_range.end.nanoseconds()) != 8)
                return x::errors::Error(
                    x::errors::UNEXPECTED,
                    "failed to write series time range end"
                );
        }

        if (!flags.equal_alignments) {
            if (buf.uint64(ser.alignment.uint64()) != 8)
                return x::errors::Error(
                    x::errors::UNEXPECTED,
                    "failed to write series alignment"
                );
        }
    }

    return x::errors::NIL;
}

std::pair<x::telem::Frame, x::errors::Error>
Codec::decode(const std::vector<uint8_t> &data) const {
    return this->decode(data.data(), data.size());
}

std::pair<x::telem::Frame, x::errors::Error>
Codec::decode(const uint8_t *data, const size_t size) const {
    this->throw_if_uninitialized();
    auto reader = x::binary::Reader(data, size);
    x::telem::Frame frame;
    uint32_t data_len = 0;
    x::telem::TimeRange ref_tr = {};
    x::telem::Alignment ref_alignment;
    auto flags = CodecFlags::decode(reader.uint8());

    auto seq_num = reader.uint32();
    auto state = this->states.at(seq_num);

    if (flags.equal_lens) data_len = reader.uint32();

    if (flags.equal_time_ranges && !flags.time_ranges_zero) {
        ref_tr.start = x::telem::TimeStamp(reader.int64());
        ref_tr.end = x::telem::TimeStamp(reader.int64());
    } else if (flags.time_ranges_zero)
        ref_tr = x::telem::TimeRange{x::telem::TimeStamp(0), x::telem::TimeStamp(0)};

    if (flags.equal_alignments && !flags.zero_alignments)
        ref_alignment = x::telem::Alignment(reader.uint64());

    auto decode_series = [&](const channel::Key key) {
        // when the series is a variable data type, we use its byte capacity instead
        // of its length.
        uint32_t local_data_len_or_byte_cap = data_len;
        if (!flags.equal_lens) local_data_len_or_byte_cap = reader.uint32();

        const auto it = state.key_data_types.find(key);
        if (it == state.key_data_types.end())
            throw std::runtime_error("Unknown channel key: " + std::to_string(key));

        auto s = x::telem::Series(it->second, local_data_len_or_byte_cap);
        s.time_range = ref_tr;
        s.alignment = ref_alignment;

        s.fill_from(reader);

        if (!flags.equal_time_ranges) {
            s.time_range.start = x::telem::TimeStamp(reader.int64());
            s.time_range.end = x::telem::TimeStamp(reader.int64());
        }

        if (!flags.equal_alignments) s.alignment = x::telem::Alignment(reader.uint64());

        if (frame.channels == nullptr) {
            frame.channels = std::make_unique<std::vector<channel::Key>>();
            frame.series = std::make_unique<std::vector<x::telem::Series>>();
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

    return {std::move(frame), x::errors::NIL};
}
}
