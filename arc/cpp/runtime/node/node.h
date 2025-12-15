// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <string>

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

namespace arc::runtime::node {
struct Context {
    telem::TimeSpan elapsed;
    std::function<void(const std::string &output_param)> mark_changed;
    std::function<void(const xerrors::Error &)> report_error;
    /// Activates the stage that the given node belongs to.
    std::function<void(const std::string &node_key)> activate_stage;
};

class Node {
public:
    virtual ~Node() = default;

    virtual xerrors::Error next(Context &ctx) = 0;

    /// Reset is called when a stage containing this node is activated.
    /// Nodes can override to reset their internal state (e.g., timers, counters).
    /// Default implementation does nothing.
    virtual void reset() {}
};
}
