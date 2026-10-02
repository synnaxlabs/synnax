// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <tuple>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/mem/local_shared.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/types/types.h"

namespace arc::stl::channels {
using Series = x::mem::local_shared<x::telem::Series>;

struct Digest {
    types::ChannelKey key;
    x::telem::DataType data_type;
    types::ChannelKey index;
};

class State {
    std::unordered_map<types::ChannelKey, types::ChannelKey> indexes;
    std::unordered_map<types::ChannelKey, std::vector<Series>> reads;
    /// @brief holds this cycle's samples, except body writes to indexed channels.
    std::unordered_map<types::ChannelKey, Series> writes;
    /// @brief holds body writes to indexed channels until flush_into stamps them.
    std::unordered_map<types::ChannelKey, Series> unstamped;
    std::vector<types::ChannelKey> active_write_keys;
    /// @brief holds the last timestamp flushed to each index.
    std::unordered_map<types::ChannelKey, int64_t> last_stamps;
    /// @brief counts the channels that wrote to each index this cycle.
    std::unordered_map<types::ChannelKey, size_t> writers;

    /// @brief returns the index of key, or zero when key has none or is an index.
    [[nodiscard]] types::ChannelKey index_of(types::ChannelKey key) const;

    /// @brief returns the buffers that body writes to key go into. Writes to an
    /// indexed channel wait in unstamped until flush_into gives them timestamps.
    std::unordered_map<types::ChannelKey, Series> &body_writes(types::ChannelKey key);

    /// @brief records key as written this cycle, unless it already is.
    void activate(types::ChannelKey key);

    /// @brief appends data to the buffer for key in bufs.
    void append_write(
        std::unordered_map<types::ChannelKey, Series> &bufs,
        types::ChannelKey key,
        const Series &data
    );

    template<typename T>
    void append_fixed_sample(types::ChannelKey key, x::telem::DataType dt, T value) {
        auto &buf = this->body_writes(key)[key];
        if (buf == nullptr || buf->empty()) this->activate(key);
        if (buf == nullptr) {
            buf = x::mem::make_local_shared<x::telem::Series>(dt, 1);
        } else if (buf->data_type() != dt) {
            buf = x::mem::make_local_shared<x::telem::Series>(dt, 1);
            buf->write(value);
            return;
        } else if (buf->cap() < buf->size() + 1) {
            const auto grown_cap = std::max(buf->size() + 1, buf->cap() * 2 + 1);
            auto grown = x::mem::make_local_shared<x::telem::Series>(dt, grown_cap);
            grown->write(*buf);
            grown->time_range = buf->time_range;
            grown->alignment = buf->alignment;
            buf = std::move(grown);
        }
        buf->write(value);
    }

    /// @brief gives every sample written this cycle a timestamp and returns the
    /// highest timestamp it synthesized.
    x::telem::TimeStamp stamp_indexes(x::telem::TimeStamp now);

    /// @brief sorts the samples key wrote with timestamps into time order, then
    /// appends its body writes with timestamps after them.
    /// @returns the last timestamp it synthesized, or zero when key made no body
    /// writes.
    x::telem::TimeStamp
    stamp_alone(types::ChannelKey key, types::ChannelKey idx, x::telem::TimeStamp now);

    /// @brief appends key's body writes to its timestamped samples. When the shared
    /// index has no timestamps yet, stamps one per sample of key, starting at now.
    /// @returns the last timestamp it synthesized, or zero when it synthesized none.
    x::telem::TimeStamp
    stamp_group(types::ChannelKey key, types::ChannelKey idx, x::telem::TimeStamp now);

public:
    template<typename T>
    void write_channel_typed(types::ChannelKey key, x::telem::DataType dt, T value) {
        this->append_fixed_sample(key, dt, value);
    }

    explicit State(const std::vector<Digest> &digests);

    State() = default;

    void ingest(const x::telem::Frame &frame);

    std::pair<x::telem::MultiSeries, bool> read_value(types::ChannelKey key);

    /// @brief buffers data with the timestamps in time, as a sink does.
    void write_value(types::ChannelKey key, const Series &data, const Series &time);

    /// @brief buffers data without timestamps, as a body write does. flush_into
    /// stamps it when the channel has an index.
    void write_value(types::ChannelKey key, const Series &data);

    void write_channel_u8(types::ChannelKey key, uint8_t value);
    void write_channel_u16(types::ChannelKey key, uint16_t value);
    void write_channel_u32(types::ChannelKey key, uint32_t value);
    void write_channel_u64(types::ChannelKey key, uint64_t value);
    void write_channel_i8(types::ChannelKey key, int8_t value);
    void write_channel_i16(types::ChannelKey key, int16_t value);
    void write_channel_i32(types::ChannelKey key, int32_t value);
    void write_channel_i64(types::ChannelKey key, int64_t value);
    void write_channel_f32(types::ChannelKey key, float value);
    void write_channel_f64(types::ChannelKey key, double value);
    void write_channel_bool(types::ChannelKey key, bool value);

    std::tuple<x::telem::MultiSeries, x::telem::MultiSeries, bool>
    read_series(types::ChannelKey key);

    void write_series(types::ChannelKey key, const Series &data, const Series &time);

    /// @brief flushes read and write state directly into the provided frame, avoiding
    /// intermediate allocations. A channel alone on its index keeps its supplied
    /// timestamps, sorted by time. Its body writes follow, 1ns apart, starting after
    /// the later of now and the last timestamp of the index. Channels that share an
    /// index are stamped from now, and only when none of them supplied timestamps.
    /// @returns the highest timestamp it synthesized, so the caller's clock can resume
    /// above it. Zero when nothing was synthesized.
    x::telem::TimeStamp flush_into(x::telem::Frame &out, x::telem::TimeStamp now);

    void reset();
};

}
