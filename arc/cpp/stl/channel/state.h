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

namespace arc::stl::channel {
using Series = x::mem::local_shared<x::telem::Series>;

struct Digest {
    types::ChannelKey key;
    x::telem::DataType data_type;
    types::ChannelKey index;
};

class State {
    std::unordered_map<types::ChannelKey, types::ChannelKey> indexes;
    std::unordered_map<types::ChannelKey, std::vector<Series>> reads;
    std::unordered_map<types::ChannelKey, Series> writes;
    std::vector<types::ChannelKey> active_write_keys;
    ::x::telem::MonoClock clock;

    template<typename T>
    void append_fixed_sample(types::ChannelKey key, x::telem::DataType dt, T value) {
        auto &buf = this->writes[key];
        if (buf == nullptr || buf->empty()) this->active_write_keys.push_back(key);
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

public:
    template<typename T>
    void write_channel_typed(types::ChannelKey key, x::telem::DataType dt, T value) {
        this->append_fixed_sample(key, dt, value);
        if (const auto idx_iter = this->indexes.find(key);
            idx_iter != this->indexes.end() && idx_iter->second != 0)
            this->append_fixed_sample(
                idx_iter->second,
                x::telem::TIMESTAMP_T,
                this->clock.now()
            );
    }
    explicit State(const std::vector<Digest> &digests);

    State() = default;

    void ingest(const x::telem::Frame &frame);

    std::pair<x::telem::MultiSeries, bool> read_value(types::ChannelKey key);

    void write_value(types::ChannelKey key, const Series &data, const Series &time);

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

    std::tuple<x::telem::MultiSeries, x::telem::MultiSeries, bool>
    read_series(types::ChannelKey key);

    void write_series(types::ChannelKey key, const Series &data, const Series &time);

    /// @brief flushes read and write state directly into the provided frame, avoiding
    /// intermediate allocations.
    void flush_into(x::telem::Frame &out);

    void reset();
};

}
