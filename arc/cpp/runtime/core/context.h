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

#include "x/cpp/xerrors/errors.h"

namespace arc {

/// @brief Context passed to nodes during execution.
///
/// Provides callback mechanisms for nodes to mark their outputs as changed
/// and report errors. This matches the Go runtime's node.Context pattern.
struct NodeContext {
    /// @brief Callback to mark a specific output parameter as changed.
    ///
    /// When a node produces new output, it calls this callback with the
    /// output parameter name. The scheduler then marks downstream nodes
    /// that depend on this output for re-execution.
    ///
    /// @param output_param Name of the output parameter that changed.
    std::function<void(const std::string& output_param)> mark_changed;

    /// @brief Callback to report errors during node execution.
    /// @param err Error to report.
    std::function<void(const xerrors::Error&)> report_error;
};

}  // namespace arc
