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

#include "x/cpp/errors/errors.h"

#include "arc/cpp/errors/errors.h"

namespace arc::runtime::errors {
/// @brief Callback invoked when a runtime error or warning occurs.
using Handler = std::function<void(const x::errors::Error &)>;
/// @brief No-op error handler for testing.
inline Handler noop_handler = [](const x::errors::Error &) {};
/// @brief Base error type for all arc runtime errors.
const auto BASE = arc::errors::BASE.sub("runtime");
/// @brief Queue capacity exceeded, data will be lost.
const auto QUEUE_FULL = BASE.sub("queue_full");
/// @brief Input queue capacity exceeded.
const auto QUEUE_FULL_INPUT = QUEUE_FULL.sub("input");
/// @brief Output queue capacity exceeded.
const auto QUEUE_FULL_OUTPUT = QUEUE_FULL.sub("output");
/// @brief WASM code called panic(), execution cannot continue.
const auto WASM_PANIC = BASE.sub("wasm_panic");
/// @brief Non-fatal warning base type - execution continues.
const auto WARNING = BASE.sub("warning");
/// @brief Multiple frames arrived for same channel in one cycle, older data discarded.
const auto DATA_DROPPED = WARNING.sub("data_dropped");
}
