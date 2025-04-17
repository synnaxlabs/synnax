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
#include <streambuf>

/// internal
#include "client/cpp/framer/framer.h"

namespace synnax {
class VectorStreamBuf final : public std::streambuf {
public:
    explicit VectorStreamBuf(const std::vector<uint8_t>& vec) {
        const auto begin = reinterpret_cast<char*>(const_cast<uint8_t*>(vec.data()));
        setg(begin, begin, begin + vec.size());
    }
};

class VectorIStream final : public std::istream {
public:
    explicit VectorIStream(const std::vector<uint8_t>& vec)
        : std::istream(nullptr), buf(vec) {
        rdbuf(&buf);
    }

private:
    VectorStreamBuf buf;
};

void synnax::BinaryWriter::uint8(const uint8_t value) {
    buf[offset++] = value;
}

void synnax::BinaryWriter::uint32(const uint32_t value) {
    buf[offset++] = static_cast<uint8_t>(value);
    buf[offset++] = static_cast<uint8_t>(value >> 8);
    buf[offset++] = static_cast<uint8_t>(value >> 16);
    buf[offset++] = static_cast<uint8_t>(value >> 24);
}

void synnax::BinaryWriter::uint64(const uint64_t value) {
    buf[offset++] = static_cast<uint8_t>(value);
    buf[offset++] = static_cast<uint8_t>(value >> 8);
    buf[offset++] = static_cast<uint8_t>(value >> 16);
    buf[offset++] = static_cast<uint8_t>(value >> 24);
    buf[offset++] = static_cast<uint8_t>(value >> 32);
    buf[offset++] = static_cast<uint8_t>(value >> 40);
    buf[offset++] = static_cast<uint8_t>(value >> 48);
    buf[offset++] = static_cast<uint8_t>(value >> 56);
}

void synnax::BinaryWriter::write(const void *data, const size_t size) {
    std::memcpy(buf.data() + offset, data, size);
    offset += size;
}

uint8_t CodecFlags::encode() const {
    uint8_t b = 0;
    b = set_bit(b, FlagPosition::EqualLengths, equal_lens);
    b = set_bit(b, FlagPosition::EqualTimeRanges, equal_time_ranges);
    b = set_bit(b, FlagPosition::TimeRangesZero, time_ranges_zero);
    b = set_bit(b, FlagPosition::AllChannelsPresent, all_channels_present);
    b = set_bit(b, FlagPosition::EqualAlignments, equal_alignments);
    b = set_bit(b, FlagPosition::ZeroAlignments, zero_alignments);
    return b;
}

CodecFlags CodecFlags::decode(const uint8_t b) {
    CodecFlags f;
    f.equal_lens = get_bit(b, FlagPosition::EqualLengths);
    f.equal_time_ranges = get_bit(b, FlagPosition::EqualTimeRanges);
    f.time_ranges_zero = get_bit(b, FlagPosition::TimeRangesZero);
    f.all_channels_present = get_bit(b, FlagPosition::AllChannelsPresent);
    f.equal_alignments = get_bit(b, FlagPosition::EqualAlignments);
    f.zero_alignments = get_bit(b, FlagPosition::ZeroAlignments);
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

    keys.reserve(channels.size());
    key_data_types.reserve(channels.size());

    for (const auto &[key, data_type]: pairs) {
        keys.push_back(key);
        key_data_types[key] = data_type;
    }
}

Codec::Codec(const std::vector<Channel> &channels) {
    key_data_types.reserve(channels.size());
    keys.reserve(channels.size());
    for (const auto &ch: channels) {
        key_data_types[ch.key] = ch.data_type;
        keys.push_back(ch.key);
    }
    std::sort(keys.begin(), keys.end());
}

telem::TimeRange Codec::read_time_range(std::istream &stream) {
    telem::TimeRange tr;
    uint64_t start, end;

    stream.read(reinterpret_cast<char *>(&start), sizeof(start));
    stream.read(reinterpret_cast<char *>(&end), sizeof(end));

    if (!stream.good()) { throw std::runtime_error("Failed to read time range"); }

    tr.start = telem::TimeStamp(start);
    tr.end = telem::TimeStamp(end);
    return tr;
}

void Codec::write_time_range(BinaryWriter &writer, const telem::TimeRange &tr) {
    writer.uint64(tr.start.nanoseconds());
    writer.uint64(tr.end.nanoseconds());
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
    ssize_t cur_data_size = -1;
    telem::TimeRange ref_tr;
    uint64_t ref_alignment = 0;

    size_t byte_array_size = start_offset + 1;

    if (frame.channels->size() != keys.size()) {
        flags.all_channels_present = false;
        byte_array_size += frame.channels->size() * 4; // 4 bytes per channel key
    }

    for (const auto &[key, idx]: sorting_indices) {
        const telem::Series &series = frame.series->at(idx);

        if (cur_data_size == -1) {
            cur_data_size = series.size();
            ref_tr = series.time_range;
            ref_alignment = series.alignment;
        }

        if (static_cast<ssize_t>(series.size()) != cur_data_size)
            flags.equal_lens = false;

        if (series.time_range != ref_tr) flags.equal_time_ranges = false;

        if (series.alignment != ref_alignment) flags.equal_alignments = false;

        byte_array_size += series.byte_size();
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

    BinaryWriter buf(data, byte_array_size, start_offset);
    buf.uint8(flags.encode());

    if (flags.equal_lens) { buf.uint32(static_cast<uint32_t>(cur_data_size)); }

    if (flags.equal_time_ranges && !flags.time_ranges_zero)
        write_time_range(buf, ref_tr);

    if (flags.equal_alignments && !flags.zero_alignments)
        buf.uint64(ref_alignment);

    for (const auto &[key, idx]: sorting_indices) {
        const telem::Series &series = frame.series->at(idx);
        const uint32_t series_data_length = static_cast<uint32_t>(series.byte_size());
        const uint32_t data_size = static_cast<uint32_t>(series.data_type().density());
        if (!flags.all_channels_present) { buf.uint32(key); }
        if (!flags.equal_lens) { buf.uint32(series_data_length / data_size); }
        buf.write(series.data(), series.byte_size());
        if (!flags.equal_time_ranges) { write_time_range(buf, series.time_range); }
        if (!flags.equal_alignments) { buf.uint64(series.alignment); }
    }
}

Frame Codec::decode(const std::vector<uint8_t> &data) const {
    auto stream = VectorIStream(data);
    return decode_stream(stream);
}

Frame Codec::decode_stream(std::istream &stream) const {
    Frame frame;
    uint32_t data_len = 0;
    telem::TimeRange ref_tr;
    uint64_t ref_alignment = 0;
    uint8_t flag_byte;

    stream.read(reinterpret_cast<char *>(&flag_byte), 1);
    if (!stream.good()) { throw std::runtime_error("Failed to read codec flags"); }

    CodecFlags flags = CodecFlags::decode(flag_byte);

    if (flags.equal_lens) {
        stream.read(reinterpret_cast<char *>(&data_len), sizeof(data_len));
        if (!stream.good()) { throw std::runtime_error("Failed to read data length"); }
    }

    if (flags.equal_time_ranges && !flags.time_ranges_zero)
        ref_tr = read_time_range(stream);
    else if (flags.time_ranges_zero)
        ref_tr = telem::TimeRange{telem::TimeStamp(0), telem::TimeStamp(0)};

    // Read common alignment if equal and not zero
    if (flags.equal_alignments && !flags.zero_alignments) {
        stream.read(reinterpret_cast<char *>(&ref_alignment), sizeof(ref_alignment));
        if (!stream.good()) { throw std::runtime_error("Failed to read alignment"); }
    }

    // Function to decode a single series given a channel key
    auto decode_series = [&](const ChannelKey key) {
        uint32_t local_data_len = data_len;
        if (!flags.equal_lens) {
            stream.read(
                reinterpret_cast<char *>(&local_data_len),
                sizeof(local_data_len)
            );
            if (!stream.good())
                throw std::runtime_error("Failed to read series length");
        }

        const auto it = key_data_types.find(key);
        if (it == key_data_types.end())
            throw std::runtime_error("Unknown channel key: " + std::to_string(key));

        auto s = telem::Series(it->second, local_data_len);
        s.time_range = ref_tr;
        s.alignment = ref_alignment;

        s.write_from_stream(stream, s.byte_cap());
        if (!stream.good()) { throw std::runtime_error("Failed to read series data"); }

        if (!flags.equal_time_ranges) { s.time_range = read_time_range(stream); }

        if (!flags.equal_alignments) {
            stream.read(reinterpret_cast<char *>(&s.alignment), sizeof(s.alignment));
            if (!stream.good())
                throw std::runtime_error("Failed to read series alignment");
        }

        if (!frame.channels) {
            frame.channels = std::make_unique<std::vector<ChannelKey>>();
            frame.series = std::make_unique<std::vector<telem::Series>>();
        }

        frame.emplace(key, std::move(s));
        return true;
    };

    if (flags.all_channels_present)
        for (const auto &key: keys)
            decode_series(key);
    else {
        while (stream.good() && !stream.eof()) {
            ChannelKey key;
            stream.read(reinterpret_cast<char *>(&key), sizeof(key));
            if (stream.eof()) break;
            if (!stream.good()) throw std::runtime_error("Failed to read channel key");
            decode_series(key);
        }
    }

    return frame;
}
}
