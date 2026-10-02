// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "x/cpp/telem/telem.h"

namespace x::loop {
/// @brief Blocks the calling thread for short spans. On Windows 10 1803 and later, a
/// sleep waits on a high-resolution timer, so it does not round up to the system tick.
/// Not safe for concurrent use.
class Sleeper {
public:
    Sleeper();
    ~Sleeper();
    Sleeper(Sleeper &&other) noexcept;
    Sleeper &operator=(Sleeper &&other) noexcept;

    /// @brief blocks the calling thread for at least dur.
    void sleep(const telem::TimeSpan &dur);

private:
    /// @brief the sleep of each platform, defined in its own source file.
    struct Impl;
    std::unique_ptr<Impl> impl_;
};
}
