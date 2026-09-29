// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/errors/errors.h"
#include "driver/mil1553/backend.h"

namespace driver::mil1553 {
namespace {
void expect_invalid(
    const synnax::mil1553::Properties &props,
    const std::string &message
) {
    const auto err = validate(props);
    ASSERT_OCCURRED_AS(err, errors::CONFIGURATION_ERROR);
    EXPECT_EQ(err.data, message);
}
}

TEST(Validate, AcceptsEachRole) {
    ASSERT_NIL(validate({.role = synnax::mil1553::ROLE_BUS_CONTROLLER}));
    ASSERT_NIL(validate({.role = synnax::mil1553::ROLE_MONITOR}));
    ASSERT_NIL(validate({
        .role = synnax::mil1553::ROLE_REMOTE_TERMINAL,
        .terminals = {0, 30},
    }));
}

TEST(Validate, RejectsAnUnknownRole) {
    expect_invalid({.role = "listener"}, "unknown MIL-STD-1553 role listener");
}

TEST(Validate, RejectsTerminalsOnABusController) {
    expect_invalid(
        {.role = synnax::mil1553::ROLE_BUS_CONTROLLER, .terminals = {1}},
        "only a remote terminal channel owns terminals"
    );
}

TEST(Validate, RejectsARemoteTerminalWithoutTerminals) {
    expect_invalid(
        {.role = synnax::mil1553::ROLE_REMOTE_TERMINAL},
        "a remote terminal channel must own at least one terminal"
    );
}

TEST(Validate, RejectsTheBroadcastAddress) {
    expect_invalid(
        {.role = synnax::mil1553::ROLE_REMOTE_TERMINAL, .terminals = {31}},
        "terminal 31 is not from 0 to 30"
    );
}

TEST(Validate, RejectsARepeatedTerminal) {
    expect_invalid(
        {.role = synnax::mil1553::ROLE_REMOTE_TERMINAL, .terminals = {4, 4}},
        "terminal 4 is listed twice"
    );
}

TEST(Transfer, CountsNoWordsForAnUnansweredTransmitCommand) {
    Transfer t{.command = {.rt = 1, .transmit = true, .subaddress = 1, .count = 4}};
    EXPECT_EQ(t.count(), 0);
    t.answered = true;
    EXPECT_EQ(t.count(), 4);
    t.status.busy = true;
    EXPECT_EQ(t.count(), 0);
}

TEST(Transfer, CountsTheWordsAReceiveCommandSent) {
    const Transfer t{.command = {.rt = 1, .subaddress = 1, .count = 3}};
    EXPECT_EQ(t.count(), 3);
}
}
