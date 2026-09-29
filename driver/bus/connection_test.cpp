// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <memory>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/connection.h"
#include "driver/bus/testutil/testutil.h"

namespace driver::bus {
using namespace testutil;

namespace {
int opens(Wire &wire) {
    std::lock_guard lock(wire.mu);
    return wire.opens;
}

std::pair<std::shared_ptr<Connection>, x::errors::Error> acquire(
    Connections &connections,
    const std::string &key,
    const x::json::json &settings,
    const std::shared_ptr<Wire> &wire
) {
    return connections.acquire(key, settings, [&] {
        return std::make_shared<Connection>(opener(wire));
    });
}

int closes(Wire &wire) {
    std::lock_guard lock(wire.mu);
    return wire.closes;
}
}

TEST(Connections, SharesOneTransportAmongTheTasksOfADevice) {
    const auto wire = std::make_shared<Wire>();
    Connections connections;
    const auto settings = x::json::json{{"port", "/dev/ttyUSB0"}};
    const auto a = ASSERT_NIL_P(acquire(connections, "dev", settings, wire));
    const auto b = ASSERT_NIL_P(acquire(connections, "dev", settings, wire));
    EXPECT_EQ(a, b);
    ASSERT_NIL(a->lock().transport().second);
    ASSERT_NIL(b->lock().transport().second);
    EXPECT_EQ(opens(*wire), 1);
}

TEST(Connections, ClosesTheTransportWhenTheLastTaskReleasesIt) {
    const auto wire = std::make_shared<Wire>();
    Connections connections;
    auto a = ASSERT_NIL_P(acquire(connections, "dev", {}, wire));
    auto b = ASSERT_NIL_P(acquire(connections, "dev", {}, wire));
    ASSERT_NIL(a->lock().transport().second);
    a.reset();
    EXPECT_EQ(closes(*wire), 0);
    b.reset();
    EXPECT_EQ(closes(*wire), 1);
    const auto c = ASSERT_NIL_P(acquire(connections, "dev", {}, wire));
    ASSERT_NIL(c->lock().transport().second);
    EXPECT_EQ(opens(*wire), 2);
}

TEST(Connections, ForgetsADeviceWhenTheLastTaskReleasesIt) {
    const auto wire = std::make_shared<Wire>();
    Connections connections;
    auto a = ASSERT_NIL_P(acquire(connections, "dev", {}, wire));
    ASSERT_NIL(a->lock().transport().second);
    EXPECT_EQ(connections.find("dev"), a);
    a.reset();
    EXPECT_EQ(connections.find("dev"), nullptr);
    EXPECT_EQ(closes(*wire), 1);
}

TEST(Connections, ClosesAConnectionReleasedAfterTheRegistry) {
    const auto wire = std::make_shared<Wire>();
    auto connections = std::make_unique<Connections>();
    auto a = ASSERT_NIL_P(acquire(*connections, "dev", {}, wire));
    ASSERT_NIL(a->lock().transport().second);
    connections.reset();
    EXPECT_EQ(closes(*wire), 0);
    a.reset();
    EXPECT_EQ(closes(*wire), 1);
}

TEST(Connections, RejectsOtherSettingsForAnOpenDevice) {
    const auto wire = std::make_shared<Wire>();
    Connections connections;
    auto a = ASSERT_NIL_P(acquire(connections, "dev", {{"baud_rate", 9600}}, wire));
    ASSERT_OCCURRED_AS_P(
        acquire(connections, "dev", {{"baud_rate", 115200}}, wire),
        transport::CONFIG_ERROR
    );
    a.reset();
    ASSERT_NIL_P(acquire(connections, "dev", {{"baud_rate", 115200}}, wire));
}

TEST(Connections, OpensADeviceConnectionPerDevice) {
    const auto wire = std::make_shared<Wire>();
    Connections connections;
    const auto a = ASSERT_NIL_P(acquire(connections, "a", {}, wire));
    const auto b = ASSERT_NIL_P(acquire(connections, "b", {}, wire));
    EXPECT_NE(a, b);
}

TEST(Connection, ReopensTheTransportAfterAClose) {
    const auto wire = std::make_shared<Wire>();
    Connection conn(opener(wire));
    {
        auto guard = conn.lock();
        ASSERT_NIL(guard.transport().second);
        EXPECT_EQ(guard.opens(), 1);
        guard.close();
    }
    auto guard = conn.lock();
    ASSERT_NIL(guard.transport().second);
    EXPECT_EQ(guard.opens(), 2);
    EXPECT_EQ(closes(*wire), 1);
}

TEST(Connection, ReturnsTheOpenError) {
    const auto wire = std::make_shared<Wire>();
    wire->open_errs.push_back(x::errors::Error(transport::UNREACHABLE_ERROR, "gone"));
    Connection conn(opener(wire));
    ASSERT_OCCURRED_AS_P(conn.lock().transport(), transport::UNREACHABLE_ERROR);
    ASSERT_NIL(conn.lock().transport().second);
}

TEST(Connection, ReportsTheErrorOfTheLastOpen) {
    const auto wire = std::make_shared<Wire>();
    wire->open_errs.push_back(x::errors::Error(transport::UNREACHABLE_ERROR, "gone"));
    Connection conn(opener(wire));
    ASSERT_NIL(conn.error());
    ASSERT_OCCURRED_AS_P(conn.lock().transport(), transport::UNREACHABLE_ERROR);
    const auto err = conn.error();
    ASSERT_MATCHES(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "gone");
    ASSERT_NIL(conn.lock().transport().second);
    ASSERT_NIL(conn.error());
}

TEST(Connection, GrantsTheLockInTheOrderItWasAsked) {
    Connection conn(opener(std::make_shared<Wire>()));
    std::atomic running = true;
    std::thread spinner([&] {
        while (running)
            auto guard = conn.lock();
    });
    const x::telem::Stopwatch sw;
    for (int i = 0; i < 1000; i++)
        auto guard = conn.lock();
    running = false;
    spinner.join();
    EXPECT_LT(sw.elapsed(), x::telem::SECOND);
}
}
