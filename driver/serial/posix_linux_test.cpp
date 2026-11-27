// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <span>
#include <system_error>
#include <vector>

#include "gtest/gtest.h"

#include "driver/serial/native.h"

namespace driver::serial::native {
/// @brief it should keep the stripped bytes when the overrun count fails.
TEST(Receive, KeepsTheBytesWhenTheOverrunCountFails) {
    Line line;
    std::vector<std::uint8_t> chunk{1, 0xFF, 0xFF, 2};
    const auto [received, ec] = receive(-1, line, chunk);
    EXPECT_EQ(ec, std::errc::bad_file_descriptor);
    EXPECT_EQ(received.size, 3);
    EXPECT_EQ(std::span(chunk).first(received.size)[1], 0xFF);
}
}
