// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <cmath>
#include <cstdint>
#include <limits>
#include <optional>
#include <string>
#include <string_view>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/bench/alloc.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/text.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/codec/text.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;

/// @brief returns a delimited field named name at position.
icd::DelimitedTextField at(const std::string &name, const std::uint32_t position) {
    icd::DelimitedTextField f;
    f.key = x::uuid::create();
    f.name = name;
    f.position = position;
    return f;
}

/// @brief returns a tagged field named name with tag.
icd::TaggedTextField tagged(const std::string &name, const std::string &tag) {
    icd::TaggedTextField f;
    f.key = x::uuid::create();
    f.name = name;
    f.tag = tag;
    return f;
}

/// @brief returns a message named name with prefix whose fields each bind to a channel
/// of dt.
TextMessage message(
    const std::string &name,
    const std::vector<icd::TextField> &fields,
    const std::string &prefix = "",
    const x::telem::DataType &dt = x::telem::FLOAT64_T
) {
    TextMessage m{.name = name, .payload = {.prefix = prefix, .fields = fields}};
    for (const auto &f: fields)
        m.bindings.push_back({.field = icd::base(f).key, .data_type = dt});
    return m;
}

/// @brief asserts that creating a decoder for messages fails with text.
void assert_refuses(const std::vector<TextMessage> &messages, const std::string &text) {
    const auto [dec, err] = Text::create(messages);
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, text);
}

/// @brief asserts that decoding line with dec fails with text.
void assert_rejects(Text &dec, const std::string_view line, const std::string &text) {
    const auto [message, err] = dec.decode(line);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, text);
}

/// @brief returns the first sample of series s of message m in dec.
double value(const Text &dec, const std::size_t m, const std::size_t s) {
    return dec.samples(m)[s].at<double>(0);
}
}

/// @brief it should read delimited fields at their positions.
TEST(Text, DecodeDelimited) {
    auto dec = ASSERT_NIL_P(Text::create({message("line", {at("a", 0), at("c", 2)})}));
    EXPECT_EQ(ASSERT_NIL_P(dec.decode("1.5,2,-3e2")), 0U);
    EXPECT_EQ(value(dec, 0, 0), 1.5);
    EXPECT_EQ(value(dec, 0, 1), -300);
}

/// @brief it should split items on a delimiter of more than one character.
TEST(Text, DecodeLongDelimiter) {
    auto m = message("line", {at("a", 0), at("b", 1)});
    m.payload.delimiter = "::";
    auto dec = ASSERT_NIL_P(Text::create({m}));
    ASSERT_NIL_P(dec.decode("4::5"));
    EXPECT_EQ(value(dec, 0, 0), 4);
    EXPECT_EQ(value(dec, 0, 1), 5);
    assert_rejects(dec, "4:5", "field a: invalid number \"4:5\"");
}

/// @brief it should read a tagged field from the first item that starts with its tag.
TEST(Text, DecodeTagged) {
    auto dec = ASSERT_NIL_P(
        Text::create({message("line", {tagged("t", "T="), tagged("p", "P=")})})
    );
    ASSERT_NIL_P(dec.decode("P=101.3, T=23.5, T=99"));
    EXPECT_EQ(value(dec, 0, 0), 23.5);
    EXPECT_EQ(value(dec, 0, 1), 101.3);
}

/// @brief it should trim spaces, tabs, and carriage returns around a value.
TEST(Text, DecodeTrimmed) {
    auto dec = ASSERT_NIL_P(Text::create({message("line", {at("a", 0), at("b", 1)})}));
    ASSERT_NIL_P(dec.decode(" \t+7 , .5\r"));
    EXPECT_EQ(value(dec, 0, 0), 7);
    EXPECT_EQ(value(dec, 0, 1), 0.5);
}

/// @brief it should scale and offset a value.
TEST(Text, DecodeScaled) {
    auto f = at("a", 0);
    f.scale = 0.5;
    f.offset = -1;
    auto dec = ASSERT_NIL_P(Text::create({message("line", {f})}));
    ASSERT_NIL_P(dec.decode("10"));
    EXPECT_EQ(value(dec, 0, 0), 4);
}

/// @brief it should write a value to a float32 channel.
TEST(Text, DecodeFloat32) {
    auto dec = ASSERT_NIL_P(
        Text::create({message("line", {at("a", 0)}, "", x::telem::FLOAT32_T)})
    );
    ASSERT_NIL_P(dec.decode("0.1"));
    EXPECT_EQ(dec.samples(0)[0].at<float>(0), 0.1F);
}

/// @brief it should give a line to the message with the longest prefix that starts it,
/// or else to the message with no prefix.
TEST(Text, RouteByPrefix) {
    auto dec = ASSERT_NIL_P(
        Text::create({
            message("any", {at("a", 1)}),
            message("gp", {at("a", 1)}, "$GP"),
            message("gga", {at("a", 1)}, "$GPGGA"),
        })
    );
    EXPECT_EQ(ASSERT_NIL_P(dec.decode("$GPGGA,1")), 2U);
    EXPECT_EQ(ASSERT_NIL_P(dec.decode("$GPRMC,2")), 1U);
    EXPECT_EQ(value(dec, 1, 0), 2);
    EXPECT_EQ(ASSERT_NIL_P(dec.decode("$HC,3")), 0U);
}

/// @brief it should match no message for a blank line, even with a message that has no
/// prefix.
TEST(Text, MatchNoneForBlankLine) {
    auto dec = ASSERT_NIL_P(Text::create({message("any", {at("a", 0)})}));
    EXPECT_FALSE(ASSERT_NIL_P(dec.decode("")).has_value());
    EXPECT_FALSE(ASSERT_NIL_P(dec.decode(" \t\r")).has_value());
}

/// @brief it should match no message for a line that no prefix starts.
TEST(Text, MatchNone) {
    auto dec = ASSERT_NIL_P(Text::create({message("gga", {at("a", 1)}, "$GPGGA")}));
    EXPECT_FALSE(ASSERT_NIL_P(dec.decode("$GPRMC,1")).has_value());
}

/// @brief it should write the default of a field whose item is empty or missing.
TEST(Text, DecodeDefault) {
    auto f = at("b", 1);
    f.default_ = 3;
    f.scale = 2;
    auto dec = ASSERT_NIL_P(Text::create({message("line", {f})}));
    ASSERT_NIL_P(dec.decode("1,,2"));
    EXPECT_EQ(value(dec, 0, 0), 6);
    ASSERT_NIL_P(dec.decode("1"));
    EXPECT_EQ(value(dec, 0, 0), 6);
    ASSERT_NIL_P(dec.decode("1,5"));
    EXPECT_EQ(value(dec, 0, 0), 10);
}

/// @brief it should reject a line with no value for a field with no default.
TEST(Text, RejectMissingValue) {
    auto dec = ASSERT_NIL_P(
        Text::create({message("line", {at("a", 0), at("b", 2), tagged("t", "T=")})})
    );
    assert_rejects(dec, "1,2,T=3", "field b: invalid number \"T=3\"");
    assert_rejects(dec, "1,2", "field b: no value at position 2");
    assert_rejects(dec, "1,2,3", "field t: no value after tag \"T=\"");
    assert_rejects(dec, "1,2,3,T=", "field t: no value after tag \"T=\"");
}

/// @brief it should reject a value that is not a decimal number.
TEST(Text, RejectInvalidNumber) {
    auto dec = ASSERT_NIL_P(Text::create({message("line", {at("a", 0)})}));
    assert_rejects(dec, "nan", "field a: invalid number \"nan\"");
    assert_rejects(dec, "0x10", "field a: invalid number \"0x10\"");
    assert_rejects(dec, "1e999", "field a: invalid number \"1e999\"");
}

/// @brief it should reject a value that overflows a float32 channel.
TEST(Text, RejectFloat32Overflow) {
    auto dec = ASSERT_NIL_P(
        Text::create({message("line", {at("a", 0)}, "", x::telem::FLOAT32_T)})
    );
    assert_rejects(dec, "1e39", "field a: 1e+39 overflows float32");
}

/// @brief it should write infinity for a value that its scale makes infinite.
TEST(Text, DecodeInfiniteScaledValue) {
    auto f = at("a", 0);
    f.scale = 1e10;
    auto dec = ASSERT_NIL_P(Text::create({message("line", {f})}));
    ASSERT_NIL_P(dec.decode("1e300"));
    EXPECT_EQ(value(dec, 0, 0), std::numeric_limits<double>::infinity());
}

/// @brief it should name the message in a decode error when it holds more than one.
TEST(Text, RejectNamesMessage) {
    auto dec = ASSERT_NIL_P(
        Text::create({
            message("gga", {at("a", 1)}, "$GPGGA"),
            message("rmc", {at("a", 1)}, "$GPRMC"),
        })
    );
    assert_rejects(dec, "$GPRMC", "message rmc: field a: no value at position 1");
}

/// @brief it should refuse two messages with the same prefix or no prefix.
TEST(Text, RefuseSharedPrefix) {
    assert_refuses(
        {message("a", {}, "$GP"), message("b", {}, "$GP")},
        "messages a and b have the same prefix \"$GP\""
    );
    assert_refuses(
        {message("a", {}), message("b", {})},
        "messages a and b both have no prefix"
    );
}

/// @brief it should refuse a tag that a trimmed item can never start with.
TEST(Text, RefuseUnmatchableTag) {
    auto m = message("a", {tagged("t", "T: ")});
    m.payload.delimiter = " ";
    assert_refuses({m}, "message a: field t: the tag contains the delimiter \" \"");
    for (const auto *tag: {" T", "\tT", "\rT"}) {
        SCOPED_TRACE(x::json::quote(tag));
        assert_refuses(
            {message("a", {tagged("t", tag)})},
            "message a: field t: the tag starts with whitespace"
        );
    }
}

/// @brief it should refuse an empty delimiter or tag.
TEST(Text, RefuseEmpty) {
    auto m = message("a", {});
    m.payload.delimiter = "";
    assert_refuses({m}, "message a: the delimiter is empty");
    assert_refuses(
        {message("a", {tagged("t", "")})},
        "message a: field t: the tag is empty"
    );
}

/// @brief it should refuse a binding to a field that does not exist.
TEST(Text, RefuseMissingField) {
    auto m = message("a", {});
    const auto key = x::uuid::create();
    m.bindings.push_back({.field = key, .data_type = x::telem::FLOAT64_T});
    assert_refuses({m}, "message a: field " + key.to_string() + " does not exist");
}

/// @brief it should refuse a field bound to a channel that cannot hold a float.
TEST(Text, RefuseNonFloatChannel) {
    assert_refuses(
        {message("a", {at("f", 0)}, "", x::telem::INT32_T)},
        "message a: field f: cannot write float values to int32 channels"
    );
}

/// @brief it should write a NaN default for a line with no value for its field.
TEST(Text, DecodeNaNDefault) {
    auto f = at("b", 1);
    f.default_ = std::numeric_limits<double>::quiet_NaN();
    auto dec = ASSERT_NIL_P(Text::create({message("line", {f})}));
    ASSERT_NIL_P(dec.decode("1"));
    EXPECT_TRUE(std::isnan(value(dec, 0, 0)));
}

/// @brief it should refuse a default that overflows its channel.
TEST(Text, RefuseDefaultThatOverflowsChannel) {
    auto f = at("f", 0);
    f.default_ = 1e300;
    assert_refuses(
        {message("a", {f}, "", x::telem::FLOAT32_T)},
        "message a: field f: default: 1e+300 overflows float32"
    );
}

/// @brief it should route and decode lines without allocating once a first decode is
/// done.
TEST(Text, DecodeWithoutAllocating) {
    auto dec = ASSERT_NIL_P(
        Text::create({
            message("gga", {at("a", 1), tagged("t", "T=")}, "$GPGGA"),
            message("any", {at("a", 0), at("b", 3)}),
        })
    );
    constexpr std::string_view gga = "$GPGGA,-122.41941550000,T=1e3";
    constexpr std::string_view any = "1,2,3,0.000123456789012";
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(gga)), 0U);
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(any)), 1U);
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++) {
        ASSERT_EQ(ASSERT_NIL_P(dec.decode(gga)), 0U);
        ASSERT_EQ(ASSERT_NIL_P(dec.decode(any)), 1U);
    }
    EXPECT_EQ(x::bench::alloc_count.load(std::memory_order_relaxed) - before, 0);
}
}
