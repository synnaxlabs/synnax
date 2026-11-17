// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <filesystem>
#include <memory>
#include <stdexcept>

#include "client/cpp/ranger/ranger.h"
#include "client/cpp/status/status.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/outbox/outbox.h"

namespace driver::outbox::testutil {
/// @brief opens an outbox in a new temporary directory, which it removes when the
/// outbox ends.
/// @param statuses sends the status sets. A default client suits a test that queues
/// no sets.
/// @param ranges creates and ends the ranges. A default client suits a test that
/// queues no range calls.
/// @throws std::runtime_error when the outbox fails to open.
inline std::shared_ptr<Outbox> open(
    const synnax::status::Client &statuses,
    const synnax::ranger::Client &ranges = {}
) {
    const auto dir = std::filesystem::temp_directory_path() /
                     ("outbox_" + x::uuid::create().to_string());
    auto [outbox, err] = Outbox::open(
        {.dir = dir, .statuses = statuses, .ranges = ranges}
    );
    if (err) throw std::runtime_error(err.message());
    return {outbox.release(), [dir](const Outbox *o) {
                delete o;
                std::error_code ec;
                std::filesystem::remove_all(dir, ec);
            }};
}
}
