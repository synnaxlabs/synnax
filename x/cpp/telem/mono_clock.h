// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>

#include "x/cpp/telem/telem.h"

namespace x::telem {

using NowFunc = std::function<TimeStamp()>;

/// @brief provides monotonically increasing timestamps. On platforms with
/// coarse clock resolution (e.g. Windows), consecutive now() calls can return
/// the same value. MonoClock guarantees each call returns a strictly greater
/// timestamp than the previous one by bumping by 1 nanosecond when necessary.
class MonoClock {
public:
    /// @brief constructs a MonoClock with an optional custom time source.
    /// If source is null or empty, defaults to TimeStamp::now. The runtime
    /// nullptr guard (rather than a default argument) mirrors Go's
    /// MonoClock.Now nil check, so callers that forward an explicit
    /// NowFunc{nullptr} (e.g. via their own defaulted parameter) still get
    /// a valid clock instead of std::bad_function_call on the first now().
    explicit MonoClock(NowFunc source = nullptr):
        source(source ? std::move(source) : NowFunc(TimeStamp::now)) {}

    /// @brief returns a timestamp strictly greater than any previous call.
    TimeStamp now() {
        auto ts = source();
        if (ts <= last) ts = last + 1;
        last = ts;
        return ts;
    }

private:
    /// @brief the underlying time source.
    NowFunc source;
    /// @brief the last timestamp returned by now().
    TimeStamp last{0};
};

}
