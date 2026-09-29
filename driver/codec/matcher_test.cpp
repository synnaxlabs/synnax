// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <optional>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/library/types.gen.h"
#include "x/cpp/test/test.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/codec/matcher.h"

namespace driver::codec {
namespace {
namespace library = synnax::library;

library::MessageEntry message(
    std::optional<library::Identifier> identifier,
    const std::string &name = "message",
    std::vector<library::BinaryField> fields = {}
) {
    library::MessageEntry m;
    m.key = x::uuid::create();
    m.name = name;
    m.payload = library::BinaryPayload{
        .identifier = std::move(identifier),
        .fields = std::move(fields),
    };
    return m;
}

library::MessageEntry
can(const std::uint32_t id,
    const bool extended = false,
    const std::optional<std::uint32_t> mask = std::nullopt) {
    return message(
        library::CanIdentifier{.id = id, .extended = extended, .mask = mask}
    );
}

library::MessageEntry
prefixed(const std::string &prefix, const std::string &name = "message") {
    library::MessageEntry m;
    m.key = x::uuid::create();
    m.name = name;
    m.payload = library::TextPayload{.prefix = prefix};
    return m;
}

library::MessageEntry header(
    const std::int32_t value,
    const std::string &name = "message",
    const std::uint8_t bit_length = 8
) {
    library::BinaryField type;
    type.key = x::uuid::create();
    type.name = "type";
    type.start_bit = 0;
    type.bit_length = bit_length;
    return message(
        library::FieldIdentifier{.field = type.key, .value = value},
        name,
        {type}
    );
}

std::span<const std::uint8_t> bytes(const std::string &s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}
}

TEST(Matcher, MatchesStandardAndExtendedIdsSeparately) {
    const std::vector messages = {can(0x100), can(0x100, true)};
    const auto m = ASSERT_NIL_P(Matcher::compile(messages));
    EXPECT_EQ(m.match(0x100, false), 0);
    EXPECT_EQ(m.match(0x100, true), 1);
    EXPECT_EQ(m.match(0x101, false), std::nullopt);
}

TEST(Matcher, MatchesAJ1939MessageFromAnySourceAddressOrPriority) {
    const std::vector messages = {can(0x18FEF100, true, 0x03FFFF00)};
    const auto m = ASSERT_NIL_P(Matcher::compile(messages));
    EXPECT_EQ(m.match(0x0CFEF1AB, true), 0);
    EXPECT_EQ(m.match(0x18FEF200, true), std::nullopt);
    EXPECT_EQ(m.match(0x0CFEF1AB & 0x7FF, false), std::nullopt);
}

TEST(Matcher, PrefersAnExactIdOverAMask) {
    const std::vector messages = {can(0x100, false, 0x700), can(0x123)};
    const auto m = ASSERT_NIL_P(Matcher::compile(messages));
    EXPECT_EQ(m.match(0x123, false), 1);
    EXPECT_EQ(m.match(0x1FF, false), 0);
}

TEST(Matcher, TreatsAFullMaskAsExact) {
    const std::vector messages = {can(0x100, false, 0xFFFFFFFF), can(0x100)};
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), CONFIG_ERROR);
}

TEST(Matcher, FallsBackToTheMessageWithNoIdentifier) {
    const std::vector messages = {can(0x100), message(std::nullopt)};
    const auto m = ASSERT_NIL_P(Matcher::compile(messages));
    EXPECT_EQ(m.match(0x100, false), 0);
    EXPECT_EQ(m.match(0x200, false), 1);
    EXPECT_EQ(m.match(bytes("anything")), 1);
}

TEST(Matcher, RejectsADuplicateExactId) {
    const std::vector messages = {can(0x100), can(0x100)};
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), CONFIG_ERROR);
}

TEST(Matcher, RejectsTwoMessagesWithNoIdentifier) {
    const std::vector messages = {message(std::nullopt), message(std::nullopt)};
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), CONFIG_ERROR);
}

TEST(Matcher, MatchesTheLongestPrefix) {
    const std::vector messages = {prefixed(""), prefixed("$GP"), prefixed("$GPGGA")};
    const auto m = ASSERT_NIL_P(Matcher::compile(messages));
    EXPECT_EQ(m.match(bytes("$GPGGA,1,2")), 2);
    EXPECT_EQ(m.match(bytes("$GPRMC,1,2")), 1);
    EXPECT_EQ(m.match(bytes("$GL")), 0);
}

TEST(Matcher, RejectsADuplicatePrefix) {
    const std::vector messages = {prefixed("A", "a"), prefixed("A", "b")};
    const auto [_, err] = Matcher::compile(messages);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "messages a and b have the same prefix A");
}

TEST(Matcher, TreatsAnEmptyPrefixAsNoIdentifier) {
    const std::vector messages = {message(std::nullopt, "a"), prefixed("", "b")};
    const auto [_, err] = Matcher::compile(messages);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "messages a and b both have no identifier");
}

TEST(Matcher, MatchesAFrameByAHeaderField) {
    const std::vector messages = {header(1), header(2)};
    const auto m = ASSERT_NIL_P(Matcher::compile(messages));
    const std::vector<std::uint8_t> two = {2, 0xAA};
    EXPECT_EQ(m.match(two), 1);
    const std::vector<std::uint8_t> three = {3, 0xAA};
    EXPECT_EQ(m.match(three), std::nullopt);
    EXPECT_EQ(m.match(std::span<const std::uint8_t>{}), std::nullopt);
}

TEST(Matcher, MatchesHeadersOfDifferentWidths) {
    const std::vector messages = {header(1), header(2), header(0x0305, "wide", 16)};
    const auto m = ASSERT_NIL_P(Matcher::compile(messages));
    const std::vector<std::uint8_t> two = {2, 0};
    EXPECT_EQ(m.match(two), 1);
    const std::vector<std::uint8_t> wide = {0x05, 0x03};
    EXPECT_EQ(m.match(wide), 2);
    const std::vector<std::uint8_t> unknown = {0x05, 0x04};
    EXPECT_EQ(m.match(unknown), std::nullopt);
}

TEST(Matcher, RejectsTwoMessagesWithTheSameHeaderValue) {
    const std::vector messages = {header(1, "a"), header(2, "b"), header(1, "c")};
    const auto [_, err] = Matcher::compile(messages);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "messages a and c have the same identifier value 1");
}

TEST(Matcher, RejectsTwoMessagesWithTheSameExactId) {
    const std::vector messages = {
        message(library::CanIdentifier{.id = 0x100}, "a"),
        message(library::CanIdentifier{.id = 0x100}, "b"),
    };
    const auto [_, err] = Matcher::compile(messages);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "messages a and b have the same CAN identifier");
}

TEST(Matcher, RejectsTwoMessagesWithTheSameIdAndMask) {
    const std::vector messages = {
        message(library::CanIdentifier{.id = 0x100, .mask = 0x700}, "a"),
        message(library::CanIdentifier{.id = 0x1FF, .mask = 0x700}, "b"),
    };
    const auto [_, err] = Matcher::compile(messages);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "messages a and b have the same CAN identifier and mask");
}

TEST(Matcher, NamesBothMessagesWithNoIdentifier) {
    const std::vector messages = {
        message(std::nullopt, "a"),
        message(std::nullopt, "b")
    };
    const auto [_, err] = Matcher::compile(messages);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "messages a and b both have no identifier");
}

TEST(Matcher, RejectsAFloatHeaderField) {
    auto m = header(1);
    std::get<library::BinaryPayload>(m.payload).fields[0].float_ = true;
    const std::vector messages = {m};
    const auto [_, err] = Matcher::compile(messages);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "message message: identifier field must be an integer");
}
}
