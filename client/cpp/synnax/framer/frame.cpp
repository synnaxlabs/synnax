// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <vector>
#include <memory>

/// api protos
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/framer.pb.h"

/// internal
#include "client/cpp/synnax/framer/framer.h"
#include "client/cpp/synnax/telem/series.h"

using namespace synnax;

Frame::Frame(
    std::unique_ptr<std::vector<ChannelKey>> columns,
    std::unique_ptr<std::vector<synnax::Series>> series
) : columns(std::move(columns)), series(std::move(series)) {
}


Frame::Frame(size_t size) {
    columns = std::make_unique<std::vector<ChannelKey>>();
    series = std::make_unique<std::vector<synnax::Series>>();
    series->reserve(size);
    columns->reserve(size);
}

Frame::Frame(const api::v1::Frame& f) {
    auto key = f.keys();
    columns = std::make_unique<std::vector<ChannelKey>>();
    series = std::make_unique<std::vector<synnax::Series>>();
    series->reserve(f.series_size());
    for (auto& ser: f.series()) series->emplace_back(ser);
    columns->reserve(key.size());
    for (auto& k: key) columns->push_back(k);
}

void Frame::add(ChannelKey col, synnax::Series ser) const {
    columns->push_back(col);
    series->push_back(std::move(ser));
}

void Frame::toProto(api::v1::Frame* f) const {
    f->mutable_keys()->Add(columns->begin(), columns->end());
    f->mutable_series()->Reserve(series->size());
    for (auto& ser: *series) ser.to_proto(f->add_series());
}
