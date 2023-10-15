// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "v1/framer.pb.h"
#include "synnax/framer/framer.h"

using namespace Synnax::Framer;

Frame::Frame(std::vector<Channel::Key> *channels, std::vector<Telem::Series> *series) {
    this->columns = channels;
    this->series = series;
}

Frame::Frame(size_t size) {
    columns = new std::vector<Channel::Key>();
    series = new std::vector<Telem::Series>();
    series->reserve(size);
    columns->reserve(size);
}

Frame::Frame(const api::v1::Frame &f) {
    auto key = f.keys();
    columns = new std::vector<Channel::Key>(key.begin(), key.end());
    series = new std::vector<Telem::Series>();
    series->reserve(f.series_size());
    for (auto &ser: f.series()) series->push_back(Telem::Series(ser));
}

void Frame::push_back(Channel::Key col, Telem::Series ser) {
    columns->push_back(col);
    series->push_back(ser);
}

void Frame::to_proto(api::v1::Frame *f) const {
    f->mutable_keys()->Add(columns->begin(), columns->end());
    f->mutable_series()->Reserve(series->size());
    for (auto &ser: *series) ser.to_proto(f->add_series());
}
