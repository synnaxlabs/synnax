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

#include "driver/arinc429/simulated/backend.h"
#include "driver/errors/errors.h"

namespace driver::arinc429::simulated {
namespace {
using codec::arinc429::Word;

synnax::arinc429::Properties create_properties(const std::uint16_t channel = 0) {
    return {.card = 0, .channel = channel};
}

std::vector<Word> read_all(Channel &ch, const std::size_t n) {
    std::vector<Received> out(n);
    const auto batch = ASSERT_NIL_P(ch.read(out, x::telem::SECOND));
    std::vector<Word> words;
    for (std::size_t i = 0; i < batch.count; i++)
        words.push_back(out[i].word);
    return words;
}
}

TEST(SimulatedBackend, DeliversWordsInOrderToEveryReceiver) {
    Backend b;
    auto rx1 = ASSERT_NIL_P(b.open(create_properties(), Direction::RECEIVE));
    auto rx2 = ASSERT_NIL_P(b.open(create_properties(), Direction::RECEIVE));
    auto tx = ASSERT_NIL_P(b.open(create_properties(), Direction::TRANSMIT));
    const std::vector words{Word(0x918C440D), Word(0x6445C0C1)};
    ASSERT_NIL(tx->write(words));
    EXPECT_EQ(read_all(*rx1, 4), words);
    EXPECT_EQ(read_all(*rx2, 4), words);
}

TEST(SimulatedBackend, StampsEachWordWithItsArrivalTime) {
    Backend b;
    auto rx = ASSERT_NIL_P(b.open(create_properties(), Direction::RECEIVE));
    auto tx = ASSERT_NIL_P(b.open(create_properties(), Direction::TRANSMIT));
    const auto before = x::telem::TimeStamp::now();
    ASSERT_NIL(tx->write(std::array{Word(1)}));
    std::array<Received, 1> out;
    const auto batch = ASSERT_NIL_P(rx->read(out, x::telem::SECOND));
    ASSERT_EQ(batch.count, 1);
    EXPECT_GE(out[0].time, before);
    EXPECT_LE(out[0].time, x::telem::TimeStamp::now());
}

TEST(SimulatedBackend, KeepsEachChannelOnItsOwnWire) {
    Backend b;
    auto rx = ASSERT_NIL_P(b.open(create_properties(1), Direction::RECEIVE));
    auto tx = ASSERT_NIL_P(b.open(create_properties(0), Direction::TRANSMIT));
    ASSERT_NIL(tx->write(std::array{Word(1)}));
    std::array<Received, 1> out;
    const auto batch = ASSERT_NIL_P(rx->read(out, x::telem::MILLISECOND));
    EXPECT_EQ(batch.count, 0);
}

TEST(SimulatedBackend, DoesNotDeliverWordsWrittenBeforeAReceiverOpened) {
    Backend b;
    auto tx = ASSERT_NIL_P(b.open(create_properties(), Direction::TRANSMIT));
    ASSERT_NIL(tx->write(std::array{Word(1)}));
    auto rx = ASSERT_NIL_P(b.open(create_properties(), Direction::RECEIVE));
    std::array<Received, 1> out;
    EXPECT_EQ(ASSERT_NIL_P(rx->read(out, x::telem::MILLISECOND)).count, 0);
}

TEST(SimulatedBackend, AllowsOneTransmitterPerWire) {
    Backend b;
    auto tx = ASSERT_NIL_P(b.open(create_properties(), Direction::TRANSMIT));
    const auto [second, err] = b.open(create_properties(), Direction::TRANSMIT);
    ASSERT_OCCURRED_AS(err, errors::CONFIGURATION_ERROR);
    EXPECT_EQ(err.data, "simulated card 0 channel 0 already has a transmitter");
    tx.reset();
    ASSERT_NIL_P(b.open(create_properties(), Direction::TRANSMIT));
}

TEST(SimulatedBackend, RejectsReadingFromATransmitter) {
    Backend b;
    auto tx = ASSERT_NIL_P(b.open(create_properties(), Direction::TRANSMIT));
    std::array<Received, 1> out;
    const auto [batch, err] = tx->read(out, x::telem::MILLISECOND);
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "cannot read from a transmit channel");
}

TEST(SimulatedBackend, RejectsWritingToAReceiver) {
    Backend b;
    auto rx = ASSERT_NIL_P(b.open(create_properties(), Direction::RECEIVE));
    const auto err = rx->write(std::array{Word(1)});
    ASSERT_OCCURRED_AS(err, errors::CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(err.data, "cannot write to a receive channel");
}

TEST(SimulatedBackend, ListsOneChannel) {
    Backend b;
    const auto infos = ASSERT_NIL_P(b.list());
    ASSERT_EQ(infos.size(), 1);
    EXPECT_EQ(infos[0].card, 0);
    EXPECT_EQ(infos[0].channel, 0);
    EXPECT_EQ(infos[0].name, "Simulated ARINC 429");
}
}
