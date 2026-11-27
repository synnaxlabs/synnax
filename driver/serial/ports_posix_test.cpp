// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/task/task.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/serial/ports.h"
#include "driver/serial/pty.h"
#include "driver/transport/connection.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
/// @returns a device with name on the port at path.
Device device(const std::string &name, const std::string &path) {
    Device dev{.key = name + "_key", .name = name};
    dev.props.port = path;
    return dev;
}

/// @returns the error that refuses a port to a device because the device with name
/// has it open through path.
std::string refused(const std::string &name, const std::string &path) {
    return "The device '" + name + "' has serial port " + path +
           " open. Two devices cannot share one port. Change the port of one of them.";
}

/// @brief waits for the backoff of conn to end, then opens it.
/// @returns the error of the open.
x::errors::Error reopen(transport::Connection &conn) {
    std::this_thread::sleep_until(conn.retry_at());
    return conn.lock().transport().second;
}
}

/// @brief it should refuse an open port to a second device through any path to the
/// port, and give it to the second device once the port closes.
TEST_F(PtyTest, RefusesOpenPortToSecondDevice) {
    Ports ports;
    auto oven = ASSERT_NIL_P(ports.acquire(device("oven", this->path())));
    const Link link(this->path());
    const auto [none, err] = ports.acquire(device("press", link.path.string()));
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    ASSERT_EQ(err.data(), refused("oven", link.path.string()));
    oven.reset();
    const auto press = ASSERT_NIL_P(ports.acquire(device("press", link.path.string())));
}

/// @brief it should claim the port that a link names at each open, so the claim
/// follows the open port when the link moves.
TEST_F(PtyTest, FollowsOpenPortWhenLinkMoves) {
    Pty other;
    other.open();
    Ports ports;
    const Link link(this->path());
    const auto oven = ASSERT_NIL_P(ports.acquire(device("oven", link.path.string())));
    link.point(other.path());
    oven->lock().fail(x::errors::Error(transport::UNREACHABLE_ERROR, "hung up"));
    ASSERT_NIL(reopen(*oven));
    const auto press = ASSERT_NIL_P(ports.acquire(device("press", this->path())));
    const auto [none, err] = ports.acquire(device("mixer", other.path()));
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    ASSERT_EQ(err.data(), refused("oven", other.path()));
}

/// @brief it should configure and start a device on a missing port, and claim the port
/// only when it opens.
TEST_F(PtyTest, ClaimsPortOnlyWhenItOpens) {
    Ports ports;
    const Link link;
    synnax::task::Task t;
    t.key = x::uuid::create();
    t.name = "oven";
    const auto claim = ASSERT_NIL_P(ports.claim_read("oven_key", t));
    const auto oven = ASSERT_NIL_P(ports.acquire(device("oven", link.path.string())));
    const auto press = ASSERT_NIL_P(ports.acquire(device("press", this->path())));
    link.point(this->path());
    const auto err = reopen(*oven);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    ASSERT_EQ(err.data(), refused("press", link.path.string()));
}

/// @brief it should release the port when the open fails after the claim.
TEST_F(PtyTest, ReleasesPortWhenOpenFails) {
    Ports ports;
    auto bad = device("oven", this->path());
    bad.props.data_bits = 9;
    ASSERT_OCCURRED_AS_P(ports.acquire(bad), transport::CONFIG_ERROR);
    const auto press = ASSERT_NIL_P(ports.acquire(device("press", this->path())));
}
}
