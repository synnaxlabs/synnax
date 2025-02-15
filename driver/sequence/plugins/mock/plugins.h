// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <vector>

/// module
#include "x/cpp/telem/control.h"

/// internal
#include "driver/pipeline/mock/pipeline.h"
#include "driver/sequence/plugins/plugins.h"

namespace plugins::mock {
class FrameSink final : public plugins::FrameSink, public pipeline::mock::Sink {
public:
    std::vector<std::pair<
        std::vector<synnax::ChannelKey>,
        std::vector<telem::Authority>
    > > authority_calls;

    xerrors::Error write(const synnax::Frame &frame) override {
        return pipeline::mock::Sink::write(frame);
    }

    xerrors::Error set_authority(
        const std::vector<synnax::ChannelKey> &keys,
        const std::vector<telem::Authority> &authorities
    ) override {
        this->authority_calls.emplace_back(keys, authorities);
        return xerrors::NIL;
    }
};
}
