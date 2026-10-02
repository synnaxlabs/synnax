// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstring>
#include <numeric>
#include <ranges>

#include "arc/cpp/stl/channels/state.h"

namespace arc::stl::channels {

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

/// @brief returns true if the buffer for key in bufs holds samples.
static bool has_data(
    const std::unordered_map<types::ChannelKey, Series> &bufs,
    types::ChannelKey key
) {
    const auto it = bufs.find(key);
    return it != bufs.end() && it->second != nullptr && !it->second->empty();
}

types::ChannelKey State::index_of(const types::ChannelKey key) const {
    const auto it = this->indexes.find(key);
    if (it == this->indexes.end() || it->second == key) return 0;
    return it->second;
}

std::unordered_map<types::ChannelKey, Series> &
State::body_writes(const types::ChannelKey key) {
    if (this->index_of(key) != 0) return this->unstamped;
    return this->writes;
}

void State::activate(const types::ChannelKey key) {
    if (!has_data(this->writes, key) && !has_data(this->unstamped, key))
        this->active_write_keys.push_back(key);
}

void State::write_value(
    const types::ChannelKey key,
    const Series &data,
    const Series &time
) {
    this->append_write(this->writes, key, data);
    if (const auto idx = this->index_of(key); idx != 0)
        this->append_write(this->writes, idx, time);
}

void State::write_value(const types::ChannelKey key, const Series &data) {
    this->append_write(this->body_writes(key), key, data);
}

void State::append_write(
    std::unordered_map<types::ChannelKey, Series> &bufs,
    const types::ChannelKey key,
    const Series &data
) {
    auto &buf = bufs[key];
    if (buf == nullptr || buf->empty()) this->activate(key);
    append_to_write_buffer(buf, data);
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
IMPL_WRITE_CHANNEL(bool, bool, x::telem::BOOLEAN_T)

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

/// @brief reorders the samples of data and times together so that times does not
/// decrease. Keeps the order of samples with equal timestamps.
static void sort_by_time(Series &data, Series &times) {
    if (times == nullptr) return;
    const auto n = times->size();
    bool sorted = true;
    for (size_t i = 1; i < n && sorted; i++)
        sorted = times->at<int64_t>(static_cast<int>(i - 1)) <=
                 times->at<int64_t>(static_cast<int>(i));
    if (sorted) return;
    std::vector<size_t> order(n);
    std::iota(order.begin(), order.end(), 0);
    std::stable_sort(order.begin(), order.end(), [&times](size_t a, size_t b) {
        return times->at<int64_t>(static_cast<int>(a)) <
               times->at<int64_t>(static_cast<int>(b));
    });
    auto sorted_times = x::mem::make_local_shared<x::telem::Series>(times->deep_copy());
    for (size_t i = 0; i < n; i++)
        std::memcpy(
            sorted_times->data() + i * sizeof(int64_t),
            times->data() + order[i] * sizeof(int64_t),
            sizeof(int64_t)
        );
    times = std::move(sorted_times);
    if (data->data_type().is_variable()) {
        const auto strings = data->strings();
        std::vector<std::string> sorted_strings;
        sorted_strings.reserve(n);
        for (const auto i: order)
            sorted_strings.push_back(strings[i]);
        auto sorted_data = x::mem::make_local_shared<x::telem::Series>(
            sorted_strings,
            data->data_type()
        );
        sorted_data->time_range = data->time_range;
        sorted_data->alignment = data->alignment;
        data = std::move(sorted_data);
        return;
    }
    const auto density = data->data_type().density();
    auto sorted_data = x::mem::make_local_shared<x::telem::Series>(data->deep_copy());
    for (size_t i = 0; i < n; i++)
        std::memcpy(
            sorted_data->data() + i * density,
            data->data() + order[i] * density,
            density
        );
    data = std::move(sorted_data);
}

x::telem::TimeStamp State::stamp_indexes(const x::telem::TimeStamp now) {
    this->writers.clear();
    for (const auto key: this->active_write_keys)
        if (const auto idx = this->index_of(key); idx != 0) this->writers[idx]++;
    auto highest = x::telem::TimeStamp(0);
    // A later iteration can grow active_write_keys, so index by position.
    const auto count = this->active_write_keys.size();
    for (size_t k = 0; k < count; k++) {
        const auto key = this->active_write_keys[k];
        const auto idx = this->index_of(key);
        if (idx == 0) continue;
        const auto last = this->writers[idx] == 1 ? this->stamp_alone(key, idx, now)
                                                  : this->stamp_group(key, idx, now);
        if (last > highest) highest = last;
    }
    return highest;
}

x::telem::TimeStamp State::stamp_alone(
    const types::ChannelKey key,
    const types::ChannelKey idx,
    const x::telem::TimeStamp now
) {
    auto &data = this->writes[key];
    auto &times = this->writes[idx];
    sort_by_time(data, times);
    auto &body = this->unstamped[key];
    auto last = x::telem::TimeStamp(0);
    if (body != nullptr && !body->empty()) {
        auto start = std::max(now.nanoseconds(), this->last_stamps[idx] + 1);
        if (times != nullptr && !times->empty())
            start = std::max(start, times->at<int64_t>(-1) + 1);
        else
            this->active_write_keys.push_back(idx);
        const auto n = body->size();
        last = x::telem::TimeStamp(start + static_cast<int64_t>(n) - 1);
        auto stamps = x::mem::make_local_shared<x::telem::Series>(
            x::telem::TIMESTAMP_T,
            n
        );
        stamps->write_linspace(x::telem::TimeStamp(start), last, n, true);
        stamps->alignment = body->alignment;
        stamps->time_range = body->time_range;
        append_to_write_buffer(times, stamps);
        append_to_write_buffer(data, body);
        body = Series{};
    }
    if (times != nullptr && !times->empty())
        this->last_stamps[idx] = times->at<int64_t>(-1);
    return last;
}

x::telem::TimeStamp State::stamp_group(
    const types::ChannelKey key,
    const types::ChannelKey idx,
    const x::telem::TimeStamp now
) {
    auto &data = this->writes[key];
    if (auto &body = this->unstamped[key]; body != nullptr && !body->empty()) {
        append_to_write_buffer(data, body);
        body = Series{};
    }
    if (has_data(this->writes, idx) || data == nullptr || data->empty())
        return x::telem::TimeStamp(0);
    const auto n = data->size();
    const auto last = x::telem::TimeStamp(
        now.nanoseconds() + static_cast<int64_t>(n) - 1
    );
    auto stamps = x::mem::make_local_shared<x::telem::Series>(x::telem::TIMESTAMP_T, n);
    stamps->write_linspace(now, last, n, true);
    stamps->alignment = data->alignment;
    stamps->time_range = data->time_range;
    this->writes[idx] = std::move(stamps);
    this->active_write_keys.push_back(idx);
    this->last_stamps[idx] = last.nanoseconds();
    return last;
}

x::telem::TimeStamp
State::flush_into(x::telem::Frame &out, const x::telem::TimeStamp now) {
    for (auto &series_vec: this->reads | std::views::values) {
        if (series_vec.size() <= 1) continue;
        auto last = std::move(series_vec.back());
        series_vec.clear();
        series_vec.push_back(std::move(last));
    }
    const auto highest = this->stamp_indexes(now);
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
    return highest;
}

void State::reset() {
    this->reads.clear();
    this->writes.clear();
    this->unstamped.clear();
    this->active_write_keys.clear();
    this->last_stamps.clear();
}

}
