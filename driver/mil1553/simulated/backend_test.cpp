// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/errors/errors.h"
#include "driver/mil1553/simulated/backend.h"

namespace driver::mil1553::simulated {
namespace {
using codec::mil1553::Command;

synnax::mil1553::Properties create_bus_controller() {
    return {.role = synnax::mil1553::ROLE_BUS_CONTROLLER};
}

synnax::mil1553::Properties create_monitor() {
    return {.role = synnax::mil1553::ROLE_MONITOR};
}

synnax::mil1553::Properties
create_remote_terminal(const std::vector<std::uint8_t> &terminals) {
    return {.role = synnax::mil1553::ROLE_REMOTE_TERMINAL, .terminals = terminals};
}

Command receive(const std::uint8_t rt, const std::uint8_t sa, const std::uint8_t n) {
    return {.rt = rt, .transmit = false, .subaddress = sa, .count = n};
}

Command transmit(const std::uint8_t rt, const std::uint8_t sa, const std::uint8_t n) {
    return {.rt = rt, .transmit = true, .subaddress = sa, .count = n};
}

std::vector<std::uint16_t> words(const Transfer &t) {
    return {t.words.begin(), t.words.begin() + static_cast<long>(t.count())};
}

std::vector<Transfer> read_all(Channel &ch, const x::telem::TimeSpan timeout) {
    std::vector<Transfer> out(8);
    const auto batch = ASSERT_NIL_P(ch.read(out, timeout));
    out.resize(batch.count);
    return out;
}

class SimulatedBackendTest : public ::testing::Test {
protected:
    Backend backend;
    std::unique_ptr<Channel> bc;

    void SetUp() override {
        this->bc = ASSERT_NIL_P(backend.open(create_bus_controller()));
    }
};
}

TEST_F(SimulatedBackendTest, WrapsDataAroundOnAnUnownedTerminal) {
    const std::array<std::uint16_t, 2> sent{0x1234, 0xABCD};
    const auto r = ASSERT_NIL_P(this->bc->transact(receive(5, 3, 2), sent));
    EXPECT_TRUE(r.answered);
    EXPECT_EQ(r.status, codec::mil1553::Status{.rt = 5});
    const auto t = ASSERT_NIL_P(this->bc->transact(transmit(5, 3, 3), {}));
    EXPECT_TRUE(t.answered);
    EXPECT_EQ(words(t), (std::vector<std::uint16_t>{0x1234, 0xABCD, 0}));
}

TEST_F(SimulatedBackendTest, AnswersZerosBeforeAnyDataArrives) {
    const auto t = ASSERT_NIL_P(this->bc->transact(transmit(1, 1, 2), {}));
    EXPECT_EQ(words(t), (std::vector<std::uint16_t>{0, 0}));
}

TEST_F(SimulatedBackendTest, ShowsEveryTransferToAMonitor) {
    auto mon = ASSERT_NIL_P(this->backend.open(create_monitor()));
    const std::array<std::uint16_t, 1> sent{7};
    ASSERT_NIL_P(this->bc->transact(receive(2, 4, 1), sent));
    ASSERT_NIL_P(this->bc->transact(transmit(9, 1, 1), {}));
    const auto seen = read_all(*mon, x::telem::SECOND);
    ASSERT_EQ(seen.size(), 2);
    EXPECT_EQ(seen[0].command, receive(2, 4, 1));
    EXPECT_EQ(words(seen[0]), (std::vector<std::uint16_t>{7}));
    EXPECT_EQ(seen[1].command, transmit(9, 1, 1));
}

TEST_F(SimulatedBackendTest, ShowsARemoteTerminalOnlyItsOwnTransfers) {
    auto rt = ASSERT_NIL_P(this->backend.open(create_remote_terminal({3})));
    const std::array<std::uint16_t, 1> sent{1};
    ASSERT_NIL_P(this->bc->transact(receive(4, 1, 1), sent));
    ASSERT_NIL_P(this->bc->transact(receive(3, 1, 1), sent));
    const auto seen = read_all(*rt, x::telem::SECOND);
    ASSERT_EQ(seen.size(), 1);
    EXPECT_EQ(seen[0].command.rt, 3);
}

TEST_F(SimulatedBackendTest, AnswersBusyFromAnOwnedTerminalWithoutAResponse) {
    auto rt = ASSERT_NIL_P(this->backend.open(create_remote_terminal({3})));
    const auto t = ASSERT_NIL_P(this->bc->transact(transmit(3, 2, 2), {}));
    EXPECT_TRUE(t.answered);
    EXPECT_TRUE(t.status.busy);
    EXPECT_EQ(t.count(), 0);
}

TEST_F(SimulatedBackendTest, AnswersWithTheResponseARemoteTerminalSet) {
    auto rt = ASSERT_NIL_P(this->backend.open(create_remote_terminal({3})));
    const std::array<std::uint16_t, 3> response{10, 20, 30};
    ASSERT_NIL(rt->respond(3, 2, response));
    const auto two = ASSERT_NIL_P(this->bc->transact(transmit(3, 2, 2), {}));
    EXPECT_FALSE(two.status.busy);
    EXPECT_EQ(words(two), (std::vector<std::uint16_t>{10, 20}));
    const auto four = ASSERT_NIL_P(this->bc->transact(transmit(3, 2, 4), {}));
    EXPECT_EQ(words(four), (std::vector<std::uint16_t>{10, 20, 30, 0}));
}

TEST_F(SimulatedBackendTest, WrapsDataAroundAgainOnceTheRemoteTerminalCloses) {
    auto rt = ASSERT_NIL_P(this->backend.open(create_remote_terminal({3})));
    const std::array<std::uint16_t, 1> response{10};
    ASSERT_NIL(rt->respond(3, 2, response));
    rt.reset();
    const auto t = ASSERT_NIL_P(this->bc->transact(transmit(3, 2, 1), {}));
    EXPECT_FALSE(t.status.busy);
    EXPECT_EQ(words(t), (std::vector<std::uint16_t>{0}));
    ASSERT_NIL_P(this->backend.open(create_remote_terminal({3})));
}

TEST_F(SimulatedBackendTest, RejectsATerminalAnotherChannelOwns) {
    auto rt = ASSERT_NIL_P(this->backend.open(create_remote_terminal({3, 4})));
    const auto [other, err] = this->backend.open(create_remote_terminal({5, 4}));
    ASSERT_OCCURRED_AS(err, errors::CONFIGURATION_ERROR);
    EXPECT_EQ(err.data, "terminal 4 is already owned by another channel");
    ASSERT_NIL_P(this->backend.open(create_remote_terminal({5})));
}

TEST_F(SimulatedBackendTest, KeepsEachChannelOnItsOwnBus) {
    auto props = create_monitor();
    props.channel = 1;
    auto mon = ASSERT_NIL_P(this->backend.open(props));
    const std::array<std::uint16_t, 1> sent{1};
    ASSERT_NIL_P(this->bc->transact(receive(1, 1, 1), sent));
    EXPECT_TRUE(read_all(*mon, x::telem::MILLISECOND).empty());
}

TEST_F(SimulatedBackendTest, RejectsInvalidProperties) {
    ASSERT_OCCURRED_AS_P(
        this->backend.open({.role = "listener"}),
        errors::CONFIGURATION_ERROR
    );
}

TEST_F(SimulatedBackendTest, RejectsAModeCode) {
    const auto [t, err] = this->bc->transact(transmit(1, 0, 1), {});
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "the simulated bus does not carry mode codes or broadcasts");
}

TEST_F(SimulatedBackendTest, RejectsABroadcast) {
    const std::array<std::uint16_t, 1> sent{1};
    ASSERT_OCCURRED_AS_P(
        this->bc->transact(receive(31, 1, 1), sent),
        errors::CRITICAL_HARDWARE_ERROR
    );
}

TEST_F(SimulatedBackendTest, RejectsAReceiveCommandWithTheWrongWordCount) {
    const std::array<std::uint16_t, 1> sent{1};
    const auto [t, err] = this->bc->transact(receive(1, 1, 2), sent);
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "a receive command with 2 words got 1");
}

TEST_F(SimulatedBackendTest, RejectsATransferFromAMonitor) {
    auto mon = ASSERT_NIL_P(this->backend.open(create_monitor()));
    const auto [t, err] = mon->transact(transmit(1, 1, 1), {});
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "only a bus controller channel makes transfers");
}

TEST_F(SimulatedBackendTest, RejectsAReadFromABusController) {
    std::array<Transfer, 1> out;
    const auto [b, err] = this->bc->read(out, x::telem::MILLISECOND);
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "a bus controller channel does not read transfers");
}

TEST_F(SimulatedBackendTest, RejectsAResponseForATerminalTheChannelDoesNotOwn) {
    auto rt = ASSERT_NIL_P(this->backend.open(create_remote_terminal({3})));
    const std::array<std::uint16_t, 1> response{1};
    const auto err = rt->respond(4, 1, response);
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "the channel does not own terminal 4");
    ASSERT_OCCURRED_AS(
        this->bc->respond(3, 1, response),
        errors::CRITICAL_HARDWARE_ERROR
    );
}

TEST_F(SimulatedBackendTest, RejectsAResponseOnAModeCodeSubaddress) {
    auto rt = ASSERT_NIL_P(this->backend.open(create_remote_terminal({3})));
    const std::array<std::uint16_t, 1> response{1};
    const auto err = rt->respond(3, 31, response);
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "subaddress 31 is not from 1 to 30");
}

TEST_F(SimulatedBackendTest, RejectsAnEmptyResponse) {
    auto rt = ASSERT_NIL_P(this->backend.open(create_remote_terminal({3})));
    const auto err = rt->respond(3, 1, {});
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "a response carries from 1 to 32 words");
}

TEST_F(SimulatedBackendTest, ListsOneChannel) {
    const auto infos = ASSERT_NIL_P(this->backend.list());
    ASSERT_EQ(infos.size(), 1);
    EXPECT_EQ(infos[0].name, "Simulated MIL-STD-1553");
}
}
