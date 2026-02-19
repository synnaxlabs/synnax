// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

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

public:
    explicit State(const std::vector<Digest> &digests);

    State() = default;

    void ingest(const x::telem::Frame &frame);

    std::pair<x::telem::MultiSeries, bool> read_value(types::ChannelKey key);

    void write_value(types::ChannelKey key, const Series &data, const Series &time);

    std::tuple<x::telem::MultiSeries, x::telem::MultiSeries, bool>
    read_chan(types::ChannelKey key);

    void write_chan(types::ChannelKey key, const Series &data, const Series &time);

    std::vector<std::pair<types::ChannelKey, Series>> flush();

    void reset();
};

}
