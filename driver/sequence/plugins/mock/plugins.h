// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <vector>

#include "x/cpp/telem/control.h"

#include "driver/pipeline/mock/pipeline.h"
#include "driver/sequence/plugins/plugins.h"

namespace plugins::mock {
class FrameSink final : public plugins::FrameSink, public driver::pipeline::mock::Sink {
public:
    std::vector<
        std::pair<std::vector<synnax::ChannelKey>, std::vector<x::telem::Authority>>>
        authority_calls;

    x::errors::Error write(const x::telem::Frame &frame) override {
        return driver::pipeline::mock::Sink::write(frame);
    }

    x::errors::Error set_authority(
        const std::vector<synnax::ChannelKey> &keys,
        const std::vector<x::telem::Authority> &authorities
    ) override {
        this->authority_calls.emplace_back(keys, authorities);
        return x::errors::NIL;
    }
};
}
