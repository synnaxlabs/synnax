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

library::MessageEntry message(std::optional<library::Identifier> identifier) {
    library::MessageEntry m;
    m.key = x::uuid::create();
    m.name = "message";
    m.identifier = std::move(identifier);
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

library::MessageEntry token(const std::string &prefix) {
    return message(library::TokenIdentifier{.prefix = prefix});
}

library::MessageEntry header(const std::uint8_t value) {
    library::BinaryField type;
    type.key = x::uuid::create();
    type.name = "type";
    type.start_bit = 0;
    type.bit_length = 8;
    auto m = message(library::FieldIdentifier{.field = type.key, .value = value});
    m.fields.emplace_back(type);
    return m;
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
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), LAYOUT_ERROR);
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
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), LAYOUT_ERROR);
}

TEST(Matcher, RejectsAStandardIdOver11Bits) {
    const std::vector messages = {can(0x800)};
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), LAYOUT_ERROR);
}

TEST(Matcher, RejectsAnExtendedIdOver29Bits) {
    const std::vector messages = {can(0x20000000, true)};
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), LAYOUT_ERROR);
}

TEST(Matcher, RejectsTwoMessagesWithNoIdentifier) {
    const std::vector messages = {message(std::nullopt), message(std::nullopt)};
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), LAYOUT_ERROR);
}

TEST(Matcher, RejectsAnUnsupportedIdentifier) {
    const std::vector messages = {message(library::Arinc429Identifier{.label = 0x10})};
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), LAYOUT_ERROR);
}

TEST(Matcher, MatchesTheLongestTokenPrefix) {
    const std::vector messages = {token(""), token("$GP"), token("$GPGGA")};
    const auto m = ASSERT_NIL_P(Matcher::compile(messages));
    EXPECT_EQ(m.match(bytes("$GPGGA,1,2")), 2);
    EXPECT_EQ(m.match(bytes("$GPRMC,1,2")), 1);
    EXPECT_EQ(m.match(bytes("$GL")), 0);
}

TEST(Matcher, RejectsADuplicateToken) {
    const std::vector messages = {token("A"), token("A")};
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), LAYOUT_ERROR);
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

TEST(Matcher, RejectsAHeaderFieldThatIsNotInTheMessage) {
    auto m = header(1);
    m.fields.clear();
    const std::vector messages = {m};
    ASSERT_OCCURRED_AS_P(Matcher::compile(messages), LAYOUT_ERROR);
}
}
