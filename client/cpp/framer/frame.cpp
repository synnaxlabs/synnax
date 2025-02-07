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
#include "client/cpp/telem/series.h"

using namespace synnax;

Frame::Frame(
    std::unique_ptr<std::vector<ChannelKey> > channels,
    std::unique_ptr<std::vector<synnax::Series> > series
) : channels(std::move(channels)), series(std::move(series)) {
}

Frame::Frame(const size_t size) :
    channels(std::make_unique<std::vector<ChannelKey> >()),
    series(std::make_unique<std::vector<synnax::Series> >()) {
    series->reserve(size);
    channels->reserve(size);
}

Frame::Frame(const ChannelKey &chan, const synnax::Series &ser) :
    channels(std::make_unique<std::vector<ChannelKey> >(1, chan)),
    series(std::make_unique<std::vector<synnax::Series> >(1, ser)) {
}

Frame::Frame(const api::v1::Frame &f) :
    channels(std::make_unique<std::vector<ChannelKey> >(
            f.keys().begin(),
            f.keys().end())
    ),
    series(std::make_unique<std::vector<synnax::Series> >()) {
    series->reserve(f.series_size());
    for (const auto &ser: f.series()) series->emplace_back(ser);
}

void Frame::add(const ChannelKey &chan, synnax::Series &ser) const {
    channels->push_back(chan);
    series->push_back(std::move(ser));
}

void Frame::to_proto(api::v1::Frame *f) const {
    f->mutable_keys()->Add(channels->begin(), channels->end());
    f->mutable_series()->Reserve(static_cast<int>(series->size()));
    for (auto &ser: *series) ser.to_proto(f->add_series());
}

void Frame::emplace(const ChannelKey &chan, synnax::Series &&ser) const {
    channels->push_back(chan);
    series->push_back(std::move(ser));
}

bool Frame::empty() const { return series->empty(); }

SampleValue Frame::at(const ChannelKey &key, const int &index) const {
    for (size_t i = 0; i < channels->size(); i++)
        if (channels->at(i) == key) return series->at(i).at(index);
    throw std::runtime_error("channel not found");
}

template<typename NumericType>
NumericType Frame::at(const ChannelKey &key, const int &index) const {
    for (size_t i = 0; i < channels->size(); i++)
        if (channels->at(i) == key) return series->at(i).at<NumericType>(index);
    throw std::runtime_error("channel not found");
}

std::ostream &synnax::operator<<(std::ostream &os, const Frame &f) {
    os << "Frame{" << std::endl;
    for (size_t i = 0; i < f.channels->size(); i++)
        os << " " << f.channels->at(i) << ": " << f.series->at(i) << ", " << std::endl;
    os << "}";
    return os;
}
