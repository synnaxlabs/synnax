// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <future>

#include "gtest/gtest.h"

#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/sync/list.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/bypass/pipeline/forwarder.h"
#include "driver/pipeline/testutil/pipeline.h"

namespace driver::bypass::pipeline {
namespace {
x::telem::Frame frame(const float value) {
    x::telem::Frame fr;
    fr.emplace(1, x::telem::Series(value));
    return fr;
}

std::shared_ptr<::driver::pipeline::testutil::WriterFactory>
factory(std::vector<x::errors::Error> open_errors = {}) {
    return std::make_shared<::driver::pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::move(open_errors)
    );
}
}

/// @brief it should count the memory each queued frame holds, so a stream of
/// one-sample frames fills its queue cap at the expected count while the Core is
/// unreachable.
TEST(ForwarderTest, DropsFramesPastTheQueueCap) {
    const auto server = factory(
        std::vector<x::errors::Error>(1000, freighter::UNREACHABLE)
    );
    constexpr size_t frame_size = FRAME_OVERHEAD + SERIES_OVERHEAD + sizeof(float);
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.max_queue_bytes = 10 * frame_size,
         .breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    for (int i = 0; i < 15; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_EQ(forwarder.dropped(), 5);
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(forwarder.dropped(), 15);
    ASSERT_EQ(server->writes->size(), 0);
}

/// @brief it should accept a frame larger than its cap when the queue is empty.
TEST(ForwarderTest, AcceptsALargeFrameIntoAnEmptyQueue) {
    const auto server = factory();
    Forwarder forwarder(server, {.channels = {1}}, {.max_queue_bytes = 2});
    ASSERT_NIL(forwarder.write(frame(1)));
    ASSERT_EVENTUALLY_EQ(server->writes->size(), 1);
    ASSERT_EQ(forwarder.dropped(), 0);
    ASSERT_NIL(forwarder.close());
}

/// @brief it should send what it holds to a reachable Core before close returns.
TEST(ForwarderTest, SendsItsQueueOnClose) {
    const auto server = factory();
    std::promise<void> gate;
    server->open_gate = gate.get_future().share();
    Forwarder forwarder(server, {.channels = {1}});
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_NIL(forwarder.set_authority({.keys = {1}, .authorities = {200}}));
    gate.set_value();
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(server->writes->size(), 3);
    ASSERT_EQ(server->authority_changes->size(), 1);
}

/// @brief it should stop at once on a license error, so its owner retries it, and
/// return the error from later calls.
TEST(ForwarderTest, StopsOnALicenseError) {
    const auto server = factory({synnax::errors::LICENSE_MISSING});
    Forwarder forwarder(server, {.channels = {1}});
    ASSERT_OCCURRED_AS(forwarder.opened(), synnax::errors::LICENSE_MISSING);
    ASSERT_EQ(server->writer_opens.load(), 1);
    ASSERT_OCCURRED_AS(forwarder.write(frame(1)), synnax::errors::LICENSE_MISSING);
    ASSERT_OCCURRED_AS(
        forwarder.set_authority({.keys = {1}, .authorities = {200}}),
        synnax::errors::LICENSE_MISSING
    );
    ASSERT_OCCURRED_AS(forwarder.close(), synnax::errors::LICENSE_MISSING);
}

/// @brief it should count the frames it holds when a permanent error stops it.
TEST(ForwarderTest, CountsHeldFramesOnAPermanentError) {
    const auto server = factory();
    std::promise<void> gate;
    server->open_gate = gate.get_future().share();
    server->open_errors = {x::errors::VALIDATION};
    Forwarder forwarder(server, {.channels = {1}});
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    gate.set_value();
    ASSERT_OCCURRED_AS(forwarder.opened(), x::errors::VALIDATION);
    ASSERT_EQ(forwarder.dropped(), 3);
    ASSERT_OCCURRED_AS(forwarder.close(), x::errors::VALIDATION);
}

/// @brief it should send frames and authority changes in queue order, on one writer
/// and on the writer it reopens after the Core becomes unreachable.
TEST(ForwarderTest, SendsOpsInQueueOrderAcrossAReopen) {
    const auto server = factory();
    std::promise<void> gate;
    server->open_gate = gate.get_future().share();
    server->return_false_ok_on = {1};
    server->close_errors = {freighter::UNREACHABLE};
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    ASSERT_NIL(forwarder.write(frame(0)));
    ASSERT_NIL(forwarder.set_authority({.keys = {1}, .authorities = {100}}));
    ASSERT_NIL(forwarder.write(frame(1)));
    ASSERT_NIL(forwarder.set_authority({.keys = {1}, .authorities = {200}}));
    ASSERT_NIL(forwarder.write(frame(2)));
    gate.set_value();
    ASSERT_EVENTUALLY_EQ(server->writes->size(), 3);
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(server->writer_opens.load(), 2);
    using ::driver::pipeline::testutil::OpType;
    const std::vector expected_ops = {
        OpType::Write,
        OpType::SetAuthority,
        OpType::Write,
        OpType::Write,
        OpType::SetAuthority,
        OpType::Write,
    };
    ASSERT_EQ(server->ops->size(), expected_ops.size());
    for (size_t i = 0; i < expected_ops.size(); i++)
        EXPECT_EQ(server->ops->at(i), expected_ops[i]);
    ASSERT_EQ(server->writes->size(), 3);
    for (size_t i = 0; i < 3; i++)
        EXPECT_EQ(server->writes->at(i).at<float>(1, 0), static_cast<float>(i));
    ASSERT_EQ(server->authority_changes->size(), 2);
    EXPECT_EQ(server->authority_changes->at(0).authorities[0], 100);
    EXPECT_EQ(server->authority_changes->at(1).authorities[0], 200);
    EXPECT_EQ(server->config.authorities, std::vector<x::control::Authority>{100});
}
}
