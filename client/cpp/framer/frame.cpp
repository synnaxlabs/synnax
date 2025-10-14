// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <vector>

#include "client/cpp/framer/framer.h"
#include "x/cpp/telem/series.h"

#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/framer.pb.h"

namespace synnax {
Frame::Frame(const size_t size):
    channels(std::make_unique<std::vector<ChannelKey>>()),
    series(std::make_unique<std::vector<telem::Series>>()) {
    this->series->reserve(size);
    this->channels->reserve(size);
}

Frame::Frame(const ChannelKey &chan, telem::Series &&ser):
    channels(std::make_unique<std::vector<ChannelKey>>(1, chan)),
    series(std::make_unique<std::vector<telem::Series>>()) {
    this->series->reserve(1);
    this->series->emplace_back(std::move(ser));
}

Frame::Frame(std::unordered_map<ChannelKey, telem::SampleValue> &data, size_t cap):
    channels(std::make_unique<std::vector<ChannelKey>>()),
    series(std::make_unique<std::vector<telem::Series>>()) {
    if (cap < data.size()) cap = data.size();
    this->series->reserve(cap);
    this->channels->reserve(cap);
    for (auto &[key, value]: data) {
        this->channels->push_back(key);
        this->series->emplace_back(telem::Series(value));
    }
}

Frame::Frame(const api::v1::Frame &f):
    channels(
        std::make_unique<std::vector<ChannelKey>>(f.keys().begin(), f.keys().end())
    ),
    series(std::make_unique<std::vector<telem::Series>>()) {
    this->series->reserve(f.series_size());
    for (const auto &ser: f.series())
        this->series->emplace_back(ser);
}

void Frame::add(const ChannelKey &chan, telem::Series &ser) const {
    this->channels->push_back(chan);
    this->series->push_back(std::move(ser));
}

void Frame::to_proto(api::v1::Frame *f) const {
    f->mutable_keys()->Add(this->channels->begin(), this->channels->end());
    f->mutable_series()->Reserve(static_cast<int>(this->series->size()));
    for (auto &ser: *this->series)
        ser.to_proto(f->add_series());
}

void Frame::emplace(const ChannelKey &chan, telem::Series &&ser) const {
    this->channels->push_back(chan);
    this->series->push_back(std::move(ser));
}

bool Frame::empty() const {
    return this->series == nullptr || this->series->empty();
}

telem::SampleValue Frame::at(const ChannelKey &key, const int &index) const {
    for (size_t i = 0; i < this->channels->size(); i++)
        if (this->channels->at(i) == key) return this->series->at(i).at(index);
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
    this->series->reserve(other.series->size());
    for (const auto &ser: *other.series)
        this->series->emplace_back(ser.deep_copy());
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
