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
#include <string>

namespace driver::arc {

/// @brief Reporter surfaces a stdlib-originated failure as a task-level status,
/// mirroring the Go-side taskreporter.Reporter so failures land as warnings rather
/// than silent log lines. Shared across the Arc driver stdlib modules.
using Reporter = std::function<
    void(const std::string &variant, const std::string &message)>;

}
