// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <ranges>

#include "arc/cpp/stl/channel/state.h"

namespace arc::stl::channel {

State::State(const std::vector<Digest> &digests) {
    for (const auto &digest: digests)
        this->indexes[digest.key] = digest.index;
}

void State::ingest(const x::telem::Frame &frame) {
    for (size_t i = 0; i < frame.size(); i++)
        this->reads[frame.channels->at(i)].push_back(
            x::mem::local_shared(std::move(frame.series->at(i)))
        );
}

std::pair<x::telem::MultiSeries, bool> State::read_value(const types::ChannelKey key) {
    const auto it = this->reads.find(key);
    if (it == this->reads.end() || it->second.empty())
        return {x::telem::MultiSeries{}, false};
    x::telem::MultiSeries ms;
    for (const auto &s: it->second)
        ms.series.push_back(s->deep_copy());
    return {std::move(ms), true};
}

static void append_to_write_buffer(Series &dest, const Series &src) {
    if (src == nullptr || src->empty()) return;
    if (dest == nullptr) {
        dest = x::mem::make_local_shared<x::telem::Series>(src->deep_copy());
        return;
    }
    if (dest->data_type() != src->data_type())
        throw std::runtime_error("cannot append series with mismatched data types");
    if (dest->data_type().is_variable()) {
        const auto old_time_range = dest->time_range;
        auto merged = dest->strings();
        auto incoming = src->strings();
        merged.insert(merged.end(), incoming.begin(), incoming.end());
        dest = x::mem::make_local_shared<x::telem::Series>(merged, dest->data_type());
        dest->time_range = old_time_range;
    } else {
        const auto required_cap = dest->size() + src->size();
        if (dest->cap() < required_cap) {
            const auto grown_cap = std::max(required_cap, dest->cap() * 2 + 1);
            auto grown = x::mem::make_local_shared<x::telem::Series>(
                dest->data_type(),
                grown_cap
            );
            grown->write(*dest);
            grown->time_range = dest->time_range;
            grown->alignment = dest->alignment;
            dest = std::move(grown);
        }
        dest->write(*src);
    }
    if (dest->time_range == x::telem::TimeRange()) {
        dest->time_range = src->time_range;
    } else {
        if (src->time_range.start < dest->time_range.start)
            dest->time_range.start = src->time_range.start;
        if (src->time_range.end > dest->time_range.end)
            dest->time_range.end = src->time_range.end;
    }
}

void State::write_value(
    const types::ChannelKey key,
    const Series &data,
    const Series &time
) {
    auto &data_buf = this->writes[key];
    if (data_buf == nullptr || data_buf->empty())
        this->active_write_keys.push_back(key);
    append_to_write_buffer(data_buf, data);
    if (const auto idx_iter = this->indexes.find(key);
        idx_iter != this->indexes.end() && idx_iter->second != 0) {
        auto &time_buf = this->writes[idx_iter->second];
        if (time_buf == nullptr || time_buf->empty())
            this->active_write_keys.push_back(idx_iter->second);
        append_to_write_buffer(time_buf, time);
    }
}

#define IMPL_WRITE_CHANNEL(suffix, cpptype, dt_const)                                  \
    void State::write_channel_##suffix(                                                \
        const types::ChannelKey key,                                                   \
        const cpptype value                                                            \
    ) {                                                                                \
        this->write_channel_typed(key, dt_const, value);                               \
    }

IMPL_WRITE_CHANNEL(u8, uint8_t, x::telem::UINT8_T)
IMPL_WRITE_CHANNEL(u16, uint16_t, x::telem::UINT16_T)
IMPL_WRITE_CHANNEL(u32, uint32_t, x::telem::UINT32_T)
IMPL_WRITE_CHANNEL(u64, uint64_t, x::telem::UINT64_T)
IMPL_WRITE_CHANNEL(i8, int8_t, x::telem::INT8_T)
IMPL_WRITE_CHANNEL(i16, int16_t, x::telem::INT16_T)
IMPL_WRITE_CHANNEL(i32, int32_t, x::telem::INT32_T)
IMPL_WRITE_CHANNEL(i64, int64_t, x::telem::INT64_T)
IMPL_WRITE_CHANNEL(f32, float, x::telem::FLOAT32_T)
IMPL_WRITE_CHANNEL(f64, double, x::telem::FLOAT64_T)

#undef IMPL_WRITE_CHANNEL

std::tuple<x::telem::MultiSeries, x::telem::MultiSeries, bool>
State::read_series(const types::ChannelKey key) {
    auto [data, ok] = this->read_value(key);
    if (!ok) return {x::telem::MultiSeries{}, x::telem::MultiSeries{}, false};
    const auto index_it = this->indexes.find(key);
    if (index_it == this->indexes.end() || index_it->second == 0) {
        const bool has_data = !data.series.empty();
        return {std::move(data), x::telem::MultiSeries{}, has_data};
    }
    auto [time, time_ok] = this->read_value(index_it->second);
    if (!time_ok) return {x::telem::MultiSeries{}, x::telem::MultiSeries{}, false};
    const bool has_data = !data.series.empty() && !time.series.empty();
    return {std::move(data), std::move(time), has_data};
}

void State::write_series(
    const types::ChannelKey key,
    const Series &data,
    const Series &time
) {
    this->write_value(key, data, time);
}

void State::flush_into(x::telem::Frame &out) {
    for (auto &series_vec: this->reads | std::views::values) {
        if (series_vec.size() <= 1) continue;
        auto last = std::move(series_vec.back());
        series_vec.clear();
        series_vec.push_back(std::move(last));
    }
    for (const auto key: this->active_write_keys) {
        auto it = this->writes.find(key);
        if (it == this->writes.end() || it->second == nullptr || it->second->empty())
            continue;
        out.emplace(key, it->second->shallow_copy());
        it->second->detach_buffer();
        it->second->time_range = x::telem::TimeRange();
        it->second->alignment = x::telem::Alignment();
    }
    this->active_write_keys.clear();
}

void State::reset() {
    this->reads.clear();
    this->writes.clear();
    this->active_write_keys.clear();
}

}
