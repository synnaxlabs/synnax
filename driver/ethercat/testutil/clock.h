// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <chrono>
#include <memory>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/telem.h"

namespace driver::ethercat::testutil {
/// @brief a clock whose time moves forward only when a loop sleeps on it, so a bus
/// cycle paced by it runs as fast as the host allows. Copies share one time, which any
/// thread can read.
class SimClock {
    struct State {
        std::atomic<int64_t> elapsed{0};
        hs_clock::time_point steady = hs_clock::now();
        x::telem::TimeStamp host = x::telem::TimeStamp::now();
    };
    std::shared_ptr<State> state = std::make_shared<State>();

public:
    /// @brief returns the clock for an x::loop::Timer to wait on.
    [[nodiscard]] x::loop::Clock loop() const {
        return {
            .now =
                [s = this->state] {
                    return s->steady + std::chrono::nanoseconds(s->elapsed.load());
                },
            .sleep = [s = this->state](
                         const x::telem::TimeSpan span
                     ) { s->elapsed.fetch_add(span.nanoseconds()); },
        };
    }

    /// @brief returns the host time on this clock.
    [[nodiscard]] x::telem::NowFunc now() const {
        return [s = this->state] {
            return s->host + x::telem::TimeSpan(s->elapsed.load());
        };
    }
};
}
