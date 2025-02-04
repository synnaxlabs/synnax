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
#include "client/cpp/telem/series.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/framer.pb.h"

using namespace synnax;

Frame::Frame(
    std::unique_ptr<std::vector<ChannelKey> > channels,
    std::unique_ptr<std::vector<synnax::Series> > series
) : channels(std::move(channels)), series(std::move(series)) {
}

Frame::Frame(const size_t size) {
    channels = std::make_unique<std::vector<ChannelKey> >();
    series = std::make_unique<std::vector<synnax::Series> >();
    series->reserve(size);
    channels->reserve(size);
}

Frame::Frame(const ChannelKey &chan, synnax::Series ser) {
    channels = std::make_unique<std::vector<ChannelKey> >();
    series = std::make_unique<std::vector<synnax::Series> >();
    channels->push_back(chan);
    series->push_back(std::move(ser));
}

Frame::Frame(const api::v1::Frame &f) {
    auto key = f.keys();
    channels = std::make_unique<std::vector<ChannelKey> >();
    series = std::make_unique<std::vector<synnax::Series> >();
    series->reserve(f.series_size());
    for (auto &ser: f.series()) series->emplace_back(ser);
    channels->reserve(key.size());
    for (auto &k: key) channels->push_back(k);
}

void Frame::add(const ChannelKey &chan, synnax::Series &ser) const {
    channels->push_back(chan);
    series->push_back(std::move(ser));
}

void Frame::add(const ChannelKey &chan, synnax::Series ser) const {
    channels->push_back(chan);
    series->push_back(std::move(ser));
}

void Frame::toProto(api::v1::Frame *f) const {
    f->mutable_keys()->Add(channels->begin(), channels->end());
    f->mutable_series()->Reserve(series->size());
    for (auto &ser: *series) ser.to_proto(f->add_series());
}

std::ostream &synnax::operator<<(std::ostream &os, const Frame &f) {
    os << "Frame{" << std::endl;
    for (size_t i = 0; i < f.channels->size(); i++) {
        os << " " << f.channels->at(i) << ": " << f.series->at(i) << ", " << std::endl;
    }
    os << "}";
    return os;
}
