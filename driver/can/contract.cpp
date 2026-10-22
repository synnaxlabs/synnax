// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <chrono>
#include <cstdint>
#include <numeric>
#include <optional>
#include <vector>

#include "driver/can/contract.h"

namespace driver::can::contract {
namespace {
const x::telem::TimeSpan WAIT = x::telem::SECOND;

Frame data_frame(
    const std::uint32_t id,
    const std::vector<std::uint8_t> &data,
    const bool extended = false
) {
    Frame f;
    f.id = id;
    f.extended = extended;
    f.length = static_cast<std::uint8_t>(data.size());
    std::ranges::copy(data, f.data.begin());
    return f;
}

/// @returns the first frame with the identifier that arrives within timeout, skipping
/// other traffic and bus errors.
std::optional<Frame> next(
    Bus &bus,
    const std::uint32_t id,
    const bool extended,
    const x::telem::TimeSpan timeout = WAIT
) {
    const auto deadline = x::telem::TimeStamp::now() + timeout;
    Frame frame;
    for (auto now = x::telem::TimeStamp::now(); now < deadline;
         now = x::telem::TimeStamp::now()) {
        const auto [got, err] = bus.receive(frame, deadline - now);
        EXPECT_FALSE(err) << err;
        if (err || !got) return std::nullopt;
        if (frame.type != Type::BUS_ERROR && frame.id == id &&
            frame.extended == extended)
            return frame;
    }
    return std::nullopt;
}

void expect_payload(const Frame &frame, const std::vector<std::uint8_t> &data) {
    EXPECT_EQ(frame.length, data.size());
    EXPECT_TRUE(std::ranges::equal(frame.payload(), data));
}
}

TEST_P(BusContract, ClosesTwice) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    ASSERT_NIL(bus->close());
    ASSERT_NIL(bus->close());
}

TEST_P(BusContract, OpensAgainAfterClose) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    ASSERT_NIL(bus->close());
    bus.reset();
    EXPECT_NE(this->open(), nullptr);
}

TEST_P(BusContract, WaitsTheWholeTimeoutWhenNoFrameArrives) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    const auto timeout = 50 * x::telem::MILLISECOND;
    Frame frame;
    for (int i = 0; i < 100; i++) {
        const auto start = std::chrono::steady_clock::now();
        if (ASSERT_NIL_P(bus->receive(frame, timeout))) continue;
        EXPECT_GE(std::chrono::steady_clock::now() - start, timeout.chrono() / 2);
        return;
    }
    FAIL() << "every receive returned a frame";
}

TEST_P(BusContract, SendsFramesToThePeer) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    auto peer = this->rig->peer(false);
    if (peer == nullptr) GTEST_SKIP() << "the rig has no peer on the wire";
    ASSERT_NIL(bus->send(data_frame(0x123, {0xDE, 0xAD})));
    const std::vector<std::uint8_t> eight = {1, 2, 3, 4, 5, 6, 7, 8};
    ASSERT_NIL(bus->send(data_frame(0x1ABCDEF0, eight, true)));
    const auto standard = next(*peer, 0x123, false);
    ASSERT_TRUE(standard.has_value());
    EXPECT_EQ(standard->type, Type::DATA);
    expect_payload(*standard, {0xDE, 0xAD});
    const auto extended = next(*peer, 0x1ABCDEF0, true);
    ASSERT_TRUE(extended.has_value());
    expect_payload(*extended, eight);
}

TEST_P(BusContract, ReceivesFramesFromThePeer) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    auto peer = this->rig->peer(false);
    if (peer == nullptr) GTEST_SKIP() << "the rig has no peer on the wire";
    const auto before = x::telem::TimeStamp::now();
    ASSERT_NIL(peer->send(data_frame(0x2A1, {7})));
    auto remote = data_frame(0x2A2, {});
    remote.type = Type::REMOTE;
    remote.length = 4;
    ASSERT_NIL(peer->send(remote));
    const auto data = next(*bus, 0x2A1, false);
    ASSERT_TRUE(data.has_value());
    EXPECT_EQ(data->type, Type::DATA);
    expect_payload(*data, {7});
    if (data->clock == Clock::HOST) {
        EXPECT_GE(data->time, before);
        EXPECT_LE(data->time, x::telem::TimeStamp::now());
    }
    const auto request = next(*bus, 0x2A2, false);
    ASSERT_TRUE(request.has_value());
    EXPECT_EQ(request->type, Type::REMOTE);
    EXPECT_EQ(request->length, 4);
    if (request->clock == data->clock) EXPECT_GE(request->time, data->time);
}

TEST_P(BusContract, NeverReceivesItsOwnFrames) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    // Without a peer, nothing acknowledges the frame on a real bus.
    auto peer = this->rig->peer(false);
    if (peer == nullptr) GTEST_SKIP() << "the rig has no peer on the wire";
    ASSERT_NIL(bus->send(data_frame(0x5A5, {0xC0, 0xFF, 0xEE})));
    ASSERT_TRUE(next(*peer, 0x5A5, false).has_value());
    EXPECT_FALSE(next(*bus, 0x5A5, false, 100 * x::telem::MILLISECOND).has_value());
}

TEST_P(BusContract, RefusesToSendWhenListenOnly) {
    auto bus = this->open(false, true);
    ASSERT_NE(bus, nullptr);
    const auto err = bus->send(data_frame(0x1, {}));
    ASSERT_MATCHES(err, LISTEN_ONLY_ERROR);
    EXPECT_EQ(err.data, "channel " + this->props().channel + " is listen only");
}

TEST_P(BusContract, ReceivesWhileListenOnly) {
    auto bus = this->open(false, true);
    ASSERT_NE(bus, nullptr);
    auto peer = this->rig->peer(false);
    if (peer == nullptr) GTEST_SKIP() << "the rig has no peer on the wire";
    ASSERT_NIL(peer->send(data_frame(0x3B3, {1, 2})));
    const auto frame = next(*bus, 0x3B3, false);
    ASSERT_TRUE(frame.has_value());
    expect_payload(*frame, {1, 2});
}

TEST_P(BusContract, CarriesCANFDFrames) {
    if (!this->rig->fd()) SKIP_MISSING_CAPABILITY("the channel does not run CAN FD");
    auto bus = this->open(true);
    ASSERT_NE(bus, nullptr);
    auto peer = this->rig->peer(true);
    if (peer == nullptr) GTEST_SKIP() << "the rig has no peer on the wire";
    std::vector<std::uint8_t> data(MAX_FD_LENGTH);
    std::iota(data.begin(), data.end(), 0);
    auto sent = data_frame(0x18DAF110, data, true);
    sent.fd = true;
    sent.bitrate_switched = true;
    ASSERT_NIL(bus->send(sent));
    const auto got = next(*peer, 0x18DAF110, true);
    ASSERT_TRUE(got.has_value());
    EXPECT_TRUE(got->fd);
    EXPECT_TRUE(got->bitrate_switched);
    expect_payload(*got, data);
    data.resize(12);
    auto back = data_frame(0x4C4, data);
    back.fd = true;
    ASSERT_NIL(peer->send(back));
    const auto received = next(*bus, 0x4C4, false);
    ASSERT_TRUE(received.has_value());
    EXPECT_TRUE(received->fd);
    EXPECT_FALSE(received->bitrate_switched);
    expect_payload(*received, data);
}

TEST_P(BusContract, FailsCallsOnceTheAdapterIsUnplugged) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    if (!this->rig->unplug()) GTEST_SKIP() << "the rig cannot unplug its adapter";
    Frame frame;
    ASSERT_OCCURRED_AS_P(bus->receive(frame, WAIT), UNREACHABLE_ERROR);
    ASSERT_MATCHES(bus->send(data_frame(0x1, {})), UNREACHABLE_ERROR);
    bus.reset();
    ASSERT_OCCURRED_AS_P(this->rig->open(this->props()), UNREACHABLE_ERROR);
}

TEST_P(BusContract, OpensAgainOnceTheAdapterReturns) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    if (!this->rig->unplug()) GTEST_SKIP() << "the rig cannot unplug its adapter";
    bus.reset();
    this->rig->plug();
    bus = this->open();
    ASSERT_NE(bus, nullptr);
    auto peer = this->rig->peer(false);
    if (peer == nullptr) GTEST_SKIP() << "the rig has no peer on the wire";
    ASSERT_NIL(peer->send(data_frame(0x6D6, {9})));
    ASSERT_TRUE(next(*bus, 0x6D6, false).has_value());
}
}
