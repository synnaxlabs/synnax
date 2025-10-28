// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/nodes/interval/node.h"

namespace arc {
namespace interval {

Node::Node(std::string id,
           State* state,
           ChannelKey output_ch,
           uint64_t period_ns)
    : id_(std::move(id)),
      state_(*state),
      output_ch_(output_ch),
      period_ns_(period_ns),
      last_execution_(telem::TimeStamp::now()) {}

xerrors::Error Node::execute(NodeContext& ctx) {
    auto now = telem::TimeStamp::now();
    auto elapsed = now - last_execution_;

    // Self-check: has the configured period elapsed?
    if (elapsed.nanoseconds() >= static_cast<int64_t>(period_ns_)) {
        // Emit tick signal (u8 value = 1)
        uint8_t tick = 1;
        auto err = state_.write_channel(output_ch_, tick);
        if (err) {
            return err;
        }

        // Mark output changed to trigger downstream propagation
        ctx.mark_changed("output");

        // Update last execution time
        last_execution_ = now;
    }

    // If period hasn't elapsed, do nothing (return early)
    return xerrors::NIL;
}

}  // namespace interval
}  // namespace arc
