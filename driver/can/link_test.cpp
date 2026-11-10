// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <utility>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/can/link.h"
#include "driver/can/loopback/loopback.h"

namespace driver::can {
namespace {
/// @brief a link over a loopback channel and a clock the test moves by hand.
class LinkTest : public ::testing::Test {
protected:
    loopback::Backend backend;
    synnax::can::Properties props;
    x::telem::TimeStamp now = x::telem::TimeStamp(x::telem::SECOND);
    std::unique_ptr<Link> link;

    void SetUp() override {
        this->props.backend = loopback::BACKEND;
        this->props.channel = "can0";
        this->link = std::make_unique<Link>(
            [this] { return this->backend.open(this->props); },
            [this] { return this->now; }
        );
    }

    std::size_t attempts() { return this->backend.attempts("can0"); }
};
}

TEST_F(LinkTest, OpensTheBusOnFirstUseAndSharesIt) {
    const auto first = ASSERT_NIL_P(this->link->bus());
    const auto second = ASSERT_NIL_P(this->link->bus());
    EXPECT_EQ(first.bus, second.bus);
    EXPECT_EQ(first.opens, 1);
    EXPECT_EQ(this->attempts(), 1);
}

TEST_F(LinkTest, WaitsForAChannelThatWasNeverOpen) {
    this->backend.unplug("can0");
    const auto err = this->link->bus().second;
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "channel can0 is not attached");
}

TEST_F(LinkTest, DoublesTheWaitAfterEachFailedOpenUpToTheLongestWait) {
    this->backend.unplug("can0");
    std::size_t expected = 0;
    auto wait = MIN_REOPEN_WAIT;
    for (int i = 0; i < 8; i++) {
        ASSERT_OCCURRED_AS_P(this->link->bus(), UNREACHABLE_ERROR);
        EXPECT_EQ(this->attempts(), ++expected) << i;
        this->now = this->now + wait - x::telem::MILLISECOND;
        ASSERT_OCCURRED_AS_P(this->link->bus(), UNREACHABLE_ERROR);
        EXPECT_EQ(this->attempts(), expected) << i;
        this->now = this->now + x::telem::MILLISECOND;
        wait = std::min(wait * 2, MAX_REOPEN_WAIT);
    }
    EXPECT_EQ(wait, MAX_REOPEN_WAIT);
}

TEST_F(LinkTest, OpensAgainAtOnceAfterAnUnreachableFault) {
    const auto first = ASSERT_NIL_P(this->link->bus());
    const auto err = this->link->report(
        first,
        {UNREACHABLE_ERROR, "the adapter stopped answering"}
    );
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "the adapter stopped answering");
    const auto second = ASSERT_NIL_P(this->link->bus());
    EXPECT_NE(second.bus, first.bus);
    EXPECT_EQ(second.opens, 2);
}

TEST_F(LinkTest, ReturnsTheErrorOfAFailedReopen) {
    const auto first = ASSERT_NIL_P(this->link->bus());
    this->backend.unplug("can0");
    this->link->report(first, {UNREACHABLE_ERROR, "gone"});
    const auto err = this->link->bus().second;
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "channel can0 is not attached");
}

TEST_F(LinkTest, DropsTheBusOnACriticalFault) {
    const auto first = ASSERT_NIL_P(this->link->bus());
    const auto err = this->link->report(first, {CRITICAL_HARDWARE_ERROR, "failed"});
    ASSERT_MATCHES(err, CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "failed");
    EXPECT_EQ(ASSERT_NIL_P(this->link->bus()).opens, 2);
}

TEST_F(LinkTest, KeepsTheBusWhenAFrameCannotBeQueued) {
    const auto first = ASSERT_NIL_P(this->link->bus());
    const auto err = this->link->report(
        first,
        {TEMPORARY_HARDWARE_ERROR, "transmit queue is full"}
    );
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "transmit queue is full");
    EXPECT_EQ(ASSERT_NIL_P(this->link->bus()).bus, first.bus);
}

TEST_F(LinkTest, IgnoresAFaultOfABusItAlreadyReplaced) {
    const auto first = ASSERT_NIL_P(this->link->bus());
    this->link->report(first, {UNREACHABLE_ERROR, "gone"});
    const auto second = ASSERT_NIL_P(this->link->bus());
    this->link->report(first, {UNREACHABLE_ERROR, "gone"});
    EXPECT_EQ(ASSERT_NIL_P(this->link->bus()).bus, second.bus);
}

TEST_F(LinkTest, StartsTheWaitOverAfterAnOpenSucceeds) {
    this->backend.unplug("can0");
    for (int i = 0; i < 3; i++) {
        ASSERT_OCCURRED_AS_P(this->link->bus(), UNREACHABLE_ERROR);
        this->now = this->now + MAX_REOPEN_WAIT;
    }
    this->backend.plug("can0");
    const auto opened = ASSERT_NIL_P(this->link->bus());
    this->backend.unplug("can0");
    this->link->report(opened, {UNREACHABLE_ERROR, "gone"});
    ASSERT_OCCURRED_AS_P(this->link->bus(), UNREACHABLE_ERROR);
    const auto before = this->attempts();
    this->now = this->now + MIN_REOPEN_WAIT;
    ASSERT_OCCURRED_AS_P(this->link->bus(), UNREACHABLE_ERROR);
    EXPECT_EQ(this->attempts(), before + 1);
}

TEST_F(LinkTest, SpacesFailedOpensOfEveryKind) {
    this->backend.fail(loopback::Call::OPEN, {CONFIG_ERROR, "bad bitrate"}, 2);
    const auto err = this->link->bus().second;
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "bad bitrate");
    ASSERT_OCCURRED_AS_P(this->link->bus(), CONFIG_ERROR);
    EXPECT_EQ(this->attempts(), 1);
}
}
