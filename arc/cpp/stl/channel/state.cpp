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

void State::write_value(
    const types::ChannelKey key,
    const Series &data,
    const Series &time
) {
    this->writes[key] = data;
    if (const auto idx_iter = this->indexes.find(key);
        idx_iter != this->indexes.end() && idx_iter->second != 0)
        this->writes[idx_iter->second] = time;
}

std::tuple<x::telem::MultiSeries, x::telem::MultiSeries, bool>
State::read_series(const types::ChannelKey key) {
    auto [data, ok] = this->read_value(key);
    if (!ok) return {x::telem::MultiSeries{}, x::telem::MultiSeries{}, false};
    const auto index_it = this->indexes.find(key);
    if (index_it == this->indexes.end() || index_it->second == 0)
        return {std::move(data), x::telem::MultiSeries{}, !data.series.empty()};
    auto [time, time_ok] = this->read_value(index_it->second);
    if (!time_ok) return {x::telem::MultiSeries{}, x::telem::MultiSeries{}, false};
    return {
        std::move(data),
        std::move(time),
        !data.series.empty() && !time.series.empty()
    };
}

void State::write_series(
    const types::ChannelKey key,
    const Series &data,
    const Series &time
) {
    this->write_value(key, data, time);
}

std::vector<std::pair<types::ChannelKey, Series>> State::flush() {
    for (auto &series_vec: this->reads | std::views::values) {
        if (series_vec.size() <= 1) continue;
        auto last = std::move(series_vec.back());
        series_vec.clear();
        series_vec.push_back(std::move(last));
    }

    std::vector<std::pair<types::ChannelKey, Series>> result;
    result.reserve(this->writes.size());
    for (const auto &[key, data]: this->writes)
        result.push_back({key, data});
    this->writes.clear();
    return result;
}

void State::reset() {
    this->reads.clear();
    this->writes.clear();
}

}
