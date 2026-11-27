// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>
#include <functional>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/serial/line.h"
#include "driver/transport/errors.h"

namespace driver::serial::line {
/// @brief it should accept the default properties.
TEST(Check, AcceptsTheDefaults) {
    ASSERT_NIL(check(synnax::serial::Properties{}));
}

/// @brief it should refuse each value outside the serial schema with a message that
/// names it.
TEST(Check, RefusesValuesOutsideTheSchema) {
    using P = synnax::serial::Properties;
    const std::vector<std::pair<std::function<void(P &)>, std::string>> cases{
        {[](P &p) { p.baud_rate = 0; }, "baud rate must be greater than zero"},
        {[](P &p) { p.data_bits = 4; }, "data bits must be 5, 6, 7, or 8, got 4"},
        {[](P &p) { p.data_bits = 9; }, "data bits must be 5, 6, 7, or 8, got 9"},
        {[](P &p) { p.parity = "Odd"; }, "unknown parity 'Odd'"},
        {[](P &p) { p.stop_bits = "3"; }, "unknown stop bits '3'"},
        {[](P &p) { p.flow_control = "rts"; }, "unknown flow control 'rts'"},
    };
    for (const auto &[edit, message]: cases) {
        SCOPED_TRACE(message);
        P p;
        edit(p);
        const auto err = check(p);
        ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
        EXPECT_EQ(err.data, message);
    }
}

/// @brief it should refuse properties that fail check before it writes any setting.
TEST(Line, RefusesPropertiesThatFailCheck) {
    synnax::serial::Properties p;
    p.data_bits = 9;
    const Settings before{};
    Settings s{};
    const auto err = apply(p, s);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    EXPECT_EQ(err.data, "data bits must be 5, 6, 7, or 8, got 9");
    EXPECT_EQ(std::memcmp(&s, &before, sizeof(Settings)), 0);
}
}
