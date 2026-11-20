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
#include "x/cpp/telem/telem.h"

namespace x::loop {
/// @brief returns a clock whose time moves forward only when a loop sleeps on it, so a
/// loop paced by it runs as fast as the host allows. Copies share one time, which any
/// thread can read.
inline HostClock fake_clock() {
    const auto start = hs_clock::now();
    const auto elapsed = std::make_shared<std::atomic<int64_t>>(0);
    return HostClock(
        Clock{
            .now =
                [start, elapsed] {
                    return start + std::chrono::nanoseconds(elapsed->load());
                },
            .sleep = [elapsed](
                         const telem::TimeSpan span
                     ) { elapsed->fetch_add(span.nanoseconds()); },
        }
    );
}
}
