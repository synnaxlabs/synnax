// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <string>

#include "gtest/gtest.h"

#include "client/cpp/task/task.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/errors/errors.h"
#include "driver/serial/ports.h"
#include "driver/serial/pty.h"

namespace driver::serial {
/// @brief it should refuse a read task on a symlink to the port of another read task.
TEST_F(PtyTest, RefusesReadTaskOnSymlinkToClaimedPort) {
    Ports ports;
    synnax::task::Task oven;
    oven.key = x::uuid::create();
    oven.name = "oven";
    const auto first = ASSERT_NIL_P(ports.claim_read(this->path(), oven));
    const auto link = std::filesystem::temp_directory_path() /
                      ("synnax-port-" + x::uuid::create().to_string());
    std::filesystem::create_symlink(this->path(), link);
    x::defer::defer remove([&link] { std::filesystem::remove(link); });
    synnax::task::Task press;
    press.key = x::uuid::create();
    press.name = "press";
    const auto [refused, err] = ports.claim_read(link.string(), press);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "A serial port runs one read task at a time, and the task 'oven' reads from " +
            this->path() + ". Add these entries to that task."
    );
}
}
