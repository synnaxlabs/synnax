// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <cstdint>
#include <iostream>
#include <random>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/library/types.gen.h"
#include "x/cpp/test/test.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/codec/mil1553.h"
#include "driver/codec/plan.h"

namespace driver::codec::mil1553 {
namespace {
namespace library = synnax::library;

library::BinaryField field(
    const std::string &name,
    const std::uint16_t start_bit,
    const std::uint8_t bit_length,
    const bool signed_ = false
) {
    library::BinaryField f;
    f.key = x::uuid::create();
    f.name = name;
    f.start_bit = start_bit;
    f.bit_length = bit_length;
    f.byte_order = library::BYTE_ORDER_BIG_ENDIAN;
    f.signed_ = signed_;
    return f;
}

library::MessageEntry message(
    const std::uint8_t rt,
    const std::uint8_t sa,
    const std::uint8_t word_count,
    std::vector<library::BinaryField> fields = {}
) {
    library::MessageEntry m;
    m.key = x::uuid::create();
    m.name = "transfer";
    m.identifier = library::Mil1553Identifier{
        .rt = rt,
        .subaddress = sa,
        .direction = library::DIRECTION_TRANSMIT,
        .word_count = word_count,
    };
    for (auto &f: fields)
        m.fields.emplace_back(std::move(f));
    return m;
}
}

TEST(Command, EncodesTheAddressDirectionSubaddressAndCount) {
    const Command c{.rt = 5, .transmit = true, .subaddress = 1, .count = 4};
    EXPECT_EQ(c.encode(), 0x2C24);
    EXPECT_EQ(Command::decode(0x2C24), c);
}

TEST(Command, EncodesACountOf32AsZero) {
    const Command c{.rt = 1, .transmit = false, .subaddress = 2, .count = 32};
    EXPECT_EQ(c.encode(), 0x0840);
    EXPECT_EQ(Command::decode(0x0840).count, 32);
}

TEST(Command, FlagsModeCodeSubaddresses) {
    EXPECT_TRUE(Command{.subaddress = 0}.mode_code());
    EXPECT_TRUE(Command{.subaddress = 31}.mode_code());
    EXPECT_FALSE(Command{.subaddress = 1}.mode_code());
}

TEST(Command, BuildsFromAnIdentifier) {
    const auto c = Command::from(
        library::Mil1553Identifier{
            .rt = 7,
            .subaddress = 3,
            .direction = library::DIRECTION_RECEIVE,
            .word_count = 2,
        }
    );
    EXPECT_EQ(c, (Command{.rt = 7, .transmit = false, .subaddress = 3, .count = 2}));
}

TEST(Command, RoundTripsEveryCommandWord) {
    for (int word = 0; word <= 0xFFFF; word++) {
        const auto w = static_cast<std::uint16_t>(word);
        ASSERT_EQ(Command::decode(w).encode(), w);
    }
}

TEST(Status, EncodesEachBitInItsPosition) {
    EXPECT_EQ(Status{.rt = 5}.encode(), 0x2800);
    EXPECT_EQ(Status{.message_error = true}.encode(), 1 << 10);
    EXPECT_EQ(Status{.instrumentation = true}.encode(), 1 << 9);
    EXPECT_EQ(Status{.service_request = true}.encode(), 1 << 8);
    EXPECT_EQ(Status{.broadcast_received = true}.encode(), 1 << 4);
    EXPECT_EQ(Status{.busy = true}.encode(), 1 << 3);
    EXPECT_EQ(Status{.subsystem_flag = true}.encode(), 1 << 2);
    EXPECT_EQ(Status{.dynamic_bus_control = true}.encode(), 1 << 1);
    EXPECT_EQ(Status{.terminal_flag = true}.encode(), 1);
}

TEST(Status, RoundTripsEveryWordWithZeroReservedBits) {
    for (int word = 0; word <= 0xFFFF; word++) {
        const auto w = static_cast<std::uint16_t>(word & ~0x00E0);
        ASSERT_EQ(Status::decode(w).encode(), w);
    }
}

TEST(Payload, WritesEachWordBigEndian) {
    const std::array<std::uint16_t, 2> words{0x1234, 0xABCD};
    std::vector<std::uint8_t> payload;
    to_payload(words, payload);
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0x12, 0x34, 0xAB, 0xCD}));
    std::array<std::uint16_t, 2> back{};
    EXPECT_EQ(from_payload(payload, back), 2);
    EXPECT_EQ(back, words);
}

TEST(Payload, DecodesABigEndianFieldPerDataWord) {
    const auto m = message(
        5,
        1,
        3,
        {field("first", 7, 16), field("second", 23, 16, true), field("flag", 47, 1)}
    );
    const auto plan = ASSERT_NIL_P(Plan::compile(m));
    auto values = plan.values();
    const std::array<std::uint16_t, 3> words{0x0102, 0xFFFE, 0x0080};
    std::vector<std::uint8_t> payload;
    to_payload(words, payload);
    ASSERT_NIL(plan.decode(payload, values));
    EXPECT_EQ(values.get(0), 0x0102);
    EXPECT_EQ(values.get(1), -2);
    EXPECT_EQ(values.get(2), 1);
}

TEST(RoundTrip, PreservesRandomDataWords) {
    std::random_device rd;
    const auto start = rd();
    std::cout << "generator start " << start << std::endl;
    std::mt19937 gen(start);
    std::uniform_int_distribution<int> count_dist(1, 32);
    std::uniform_int_distribution<int> word_dist(0, 0xFFFF);
    for (int i = 0; i < 500; i++) {
        const auto count = count_dist(gen);
        std::vector<library::BinaryField> fields;
        for (int w = 0; w < count; w++)
            fields.push_back(field(
                "w" + std::to_string(w),
                static_cast<std::uint16_t>(16 * w + 7),
                16,
                w % 2 == 1
            ));
        const auto m = message(5, 1, static_cast<std::uint8_t>(count), fields);
        ASSERT_NIL(validate(m));
        const auto plan = ASSERT_NIL_P(Plan::compile(m));
        auto values = plan.values();
        std::vector<std::uint16_t> words(count);
        for (auto &w: words)
            w = static_cast<std::uint16_t>(word_dist(gen));
        std::vector<std::uint8_t> payload;
        to_payload(words, payload);
        ASSERT_NIL(plan.decode(payload, values));
        std::vector<std::uint8_t> encoded;
        ASSERT_NIL(plan.encode(values, encoded));
        ASSERT_EQ(encoded, payload) << "generator start " << start;
    }
}

TEST(Validate, AcceptsFieldsInsideTheDataWords) {
    ASSERT_NIL(validate(message(5, 1, 2, {field("x", 23, 16)})));
}

TEST(Validate, RejectsAFieldBeyondTheDataWords) {
    ASSERT_OCCURRED_AS(validate(message(5, 1, 1, {field("x", 23, 16)})), LAYOUT_ERROR);
}

TEST(Validate, RejectsModeCodesAndBroadcast) {
    ASSERT_OCCURRED_AS(validate(message(5, 0, 1)), LAYOUT_ERROR);
    ASSERT_OCCURRED_AS(validate(message(5, 31, 1)), LAYOUT_ERROR);
    ASSERT_OCCURRED_AS(validate(message(31, 1, 1)), LAYOUT_ERROR);
}

TEST(Validate, RejectsAWordCountOutOfRange) {
    ASSERT_OCCURRED_AS(validate(message(5, 1, 0)), LAYOUT_ERROR);
    ASSERT_OCCURRED_AS(validate(message(5, 1, 33)), LAYOUT_ERROR);
}

TEST(Validate, RejectsALengthThatDisagreesWithTheWordCount) {
    auto m = message(5, 1, 2);
    m.length = 2;
    ASSERT_OCCURRED_AS(validate(m), LAYOUT_ERROR);
    m.length = 4;
    ASSERT_NIL(validate(m));
}

TEST(Validate, RejectsAnotherIdentifier) {
    auto m = message(5, 1, 1);
    m.identifier = library::Arinc429Identifier{.label = 1};
    ASSERT_OCCURRED_AS(validate(m), LAYOUT_ERROR);
}
}
