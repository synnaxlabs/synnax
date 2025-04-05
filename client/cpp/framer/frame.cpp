// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <memory>
#include <vector>

/// protos
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/framer.pb.h"

/// internal
#include "client/cpp/framer/framer.h"
#include "x/cpp/telem/series.h"

namespace synnax {
Frame::Frame(const size_t size):
    channels(std::make_unique<std::vector<ChannelKey>>()),
    series(std::make_unique<std::vector<telem::Series>>()) {
    series->reserve(size);
    channels->reserve(size);
}

Frame::Frame(const ChannelKey &chan, telem::Series &&ser):
    channels(std::make_unique<std::vector<ChannelKey>>(1, chan)),
    series(std::make_unique<std::vector<telem::Series>>()) {
    series->reserve(1);
    series->emplace_back(std::move(ser));
}

Frame::Frame(std::unordered_map<ChannelKey, telem::SampleValue> &data, size_t cap):
    channels(std::make_unique<std::vector<ChannelKey>>()),
    series(std::make_unique<std::vector<telem::Series>>()) {
    if (cap < data.size()) cap = data.size();
    series->reserve(cap);
    channels->reserve(cap);
    for (auto &[key, value]: data) {
        channels->push_back(key);
        series->emplace_back(telem::Series(value));
    }
}

Frame::Frame(const api::v1::Frame &f):
    channels(std::make_unique<std::vector<ChannelKey>>(f.keys().begin(), f.keys().end())
    ),
    series(std::make_unique<std::vector<telem::Series>>()) {
    series->reserve(f.series_size());
    for (const auto &ser: f.series())
        series->emplace_back(ser);
}

void Frame::add(const ChannelKey &chan, telem::Series &ser) const {
    channels->push_back(chan);
    series->push_back(std::move(ser));
}

void Frame::to_proto(api::v1::Frame *f) const {
    f->mutable_keys()->Add(channels->begin(), channels->end());
    f->mutable_series()->Reserve(static_cast<int>(series->size()));
    for (auto &ser: *series)
        ser.to_proto(f->add_series());
}

void Frame::emplace(const ChannelKey &chan, telem::Series &&ser) const {
    channels->push_back(chan);
    series->push_back(std::move(ser));
}

bool Frame::empty() const {
    return series == nullptr || series->empty();
}

telem::SampleValue Frame::at(const ChannelKey &key, const int &index) const {
    for (size_t i = 0; i < channels->size(); i++)
        if (channels->at(i) == key) return series->at(i).at(index);
    throw std::runtime_error("channel not found");
}

void Frame::clear() const {
    this->channels->clear();
    this->series->clear();
}

void Frame::reserve(const size_t &size) {
    if (this->channels == nullptr)
        this->channels = std::make_unique<std::vector<ChannelKey>>();
    if (this->series == nullptr)
        this->series = std::make_unique<std::vector<telem::Series>>();
    this->channels->reserve(size);
    this->series->reserve(size);
}

Frame Frame::deep_copy() const {
    return Frame(*this);
}

Frame::Frame(const Frame &other):
    channels(std::make_unique<std::vector<ChannelKey>>(*other.channels)),
    series(std::make_unique<std::vector<telem::Series>>()) {
    series->reserve(other.series->size());
    for (const auto &ser: *other.series)
        series->emplace_back(ser.deep_copy());
}

Frame::Frame(Frame &&other) noexcept:
    channels(std::move(other.channels)), series(std::move(other.series)) {
    other.channels = nullptr;
    other.series = nullptr;
}

std::ostream &operator<<(std::ostream &os, const Frame &f) {
    os << "Frame{" << std::endl;
    for (size_t i = 0; i < f.channels->size(); i++)
        os << " " << f.channels->at(i) << ": " << f.series->at(i) << ", " << std::endl;
    os << "}";
    return os;
}
}
