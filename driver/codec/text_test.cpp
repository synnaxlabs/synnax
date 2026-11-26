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
#include <cstddef>
#include <cstdint>
#include <limits>
#include <optional>
#include <string>
#include <string_view>
#include <utility>
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

/// @brief it should give each tagged item to the field with the longest tag that starts
/// it, bound or not.
TEST(Text, DecodeLongestTag) {
    const auto t = tagged("t", "T");
    const auto te = tagged("te", "TE");
    auto both = ASSERT_NIL_P(Text::create({message("line", {t, te})}));
    ASSERT_NIL_P(both.decode("TE5,T3"));
    EXPECT_EQ(value(both, 0, 0), 3);
    EXPECT_EQ(value(both, 0, 1), 5);
    auto m = message("line", {t, te});
    m.bindings.pop_back();
    auto one = ASSERT_NIL_P(Text::create({m}));
    ASSERT_NIL_P(one.decode("TE5,T3"));
    EXPECT_EQ(value(one, 0, 0), 3);
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

/// @brief it should refuse a field with a zero scale.
TEST(Text, RefuseZeroScale) {
    auto f = at("f", 0);
    f.scale = 0;
    assert_refuses({message("a", {f})}, "message a: field f: scale must be non-zero");
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

namespace {
/// @brief returns a vector that holds each series of s.
template<typename... S>
std::vector<x::telem::Series> series_of(S &&...s) {
    std::vector<x::telem::Series> out;
    out.reserve(sizeof...(s));
    (out.push_back(std::forward<S>(s)), ...);
    return out;
}

/// @brief returns the line that an encoder of m encodes from samples.
std::string encode(const TextMessage &m, const std::vector<x::telem::Series> &samples) {
    auto enc = ASSERT_NIL_P(TextEncoder::create(m));
    std::vector<std::uint8_t> payload;
    EXPECT_NIL(enc.encode(samples, payload));
    return {payload.begin(), payload.end()};
}

/// @brief asserts that encoding samples for m fails with text.
void assert_rejects_samples(
    const TextMessage &m,
    const std::vector<x::telem::Series> &samples,
    const std::string &text
) {
    auto enc = ASSERT_NIL_P(TextEncoder::create(m));
    std::vector<std::uint8_t> payload;
    const auto err = enc.encode(samples, payload);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, text);
}

/// @brief asserts that creating an encoder for m fails with text.
void assert_encoder_refuses(const TextMessage &m, const std::string &text) {
    const auto err = TextEncoder::create(m).second;
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, text);
}

/// @brief asserts that a decoder of m reads values back from line.
void assert_decodes(
    const TextMessage &m,
    const std::string &line,
    const std::vector<double> &values
) {
    auto dec = ASSERT_NIL_P(Text::create({m}));
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(line)), 0U);
    for (std::size_t i = 0; i < values.size(); i++)
        EXPECT_EQ(value(dec, 0, i), values[i]);
}

/// @brief returns an empty series, which takes the default of its field.
x::telem::Series none() {
    return x::telem::Series(x::telem::FLOAT64_T, 0);
}
}

/// @brief it should write each delimited field at its position, with an empty item
/// where no field is.
TEST(TextEncoder, EncodeDelimited) {
    const auto m = message("cmd", {at("a", 1), at("b", 3)});
    const auto line = encode(
        m,
        series_of(x::telem::Series(1.5), x::telem::Series(-2.0))
    );
    EXPECT_EQ(line, ",1.5,,-2");
    assert_decodes(m, line, {1.5, -2});
}

/// @brief it should start a line with its prefix, and fill the first items with it.
TEST(TextEncoder, EncodePrefix) {
    const auto m = message("cmd", {at("a", 1), at("b", 2)}, "$SET");
    const auto line = encode(
        m,
        series_of(x::telem::Series(1.0), x::telem::Series(2.0))
    );
    EXPECT_EQ(line, "$SET,1,2");
    assert_decodes(m, line, {1, 2});
    const auto split = message("cmd", {at("a", 2)}, "$SET,X,");
    EXPECT_EQ(encode(split, series_of(x::telem::Series(3.0))), "$SET,X,3");
    const auto bare = message("cmd", {tagged("t", "T=")}, "$SET,");
    EXPECT_EQ(encode(bare, series_of(x::telem::Series(4.0))), "$SET,T=4");
    EXPECT_EQ(encode(message("cmd", {}, "$SET,"), {}), "$SET,");
}

/// @brief it should write each tagged field as its tag and value after the delimited
/// fields.
TEST(TextEncoder, EncodeTagged) {
    const auto m = message("cmd", {tagged("t", "T="), at("a", 0), tagged("h", "H=")});
    const auto line = encode(
        m,
        series_of(x::telem::Series(21.5), x::telem::Series(1.0), x::telem::Series(40.0))
    );
    EXPECT_EQ(line, "1,T=21.5,H=40");
    assert_decodes(m, line, {21.5, 1, 40});
}

/// @brief it should write a tag that starts another tag, and read both back.
TEST(TextEncoder, EncodeOverlappingTags) {
    const auto m = message("cmd", {tagged("te", "TE"), tagged("t", "T")});
    const auto line = encode(
        m,
        series_of(x::telem::Series(5.0), x::telem::Series(3.0))
    );
    EXPECT_EQ(line, "TE5,T3");
    assert_decodes(m, line, {5, 3});
}

/// @brief it should write the shortest text that reads back as each sample, at the
/// precision of its channel.
TEST(TextEncoder, EncodeNumbers) {
    const auto f = at("a", 0);
    EXPECT_EQ(
        encode(
            message("cmd", {f}, "", x::telem::INT32_T),
            series_of(x::telem::Series(-5))
        ),
        "-5"
    );
    constexpr std::int64_t odd = (std::int64_t{1} << 53) + 1;
    EXPECT_EQ(
        encode(
            message("cmd", {f}, "", x::telem::INT64_T),
            series_of(x::telem::Series(odd))
        ),
        "9007199254740993"
    );
    constexpr auto max = std::numeric_limits<std::uint64_t>::max();
    EXPECT_EQ(
        encode(
            message("cmd", {f}, "", x::telem::UINT64_T),
            series_of(x::telem::Series(max))
        ),
        "18446744073709551615"
    );
    EXPECT_EQ(
        encode(
            message("cmd", {f}, "", x::telem::FLOAT32_T),
            series_of(x::telem::Series(1.1F))
        ),
        "1.1"
    );
    const auto m = message("cmd", {f});
    EXPECT_EQ(encode(m, series_of(x::telem::Series(1e300))), "1e+300");
    EXPECT_EQ(encode(m, series_of(x::telem::Series(1e-5))), "1e-05");
    EXPECT_EQ(encode(m, series_of(x::telem::Series(1234567.0))), "1234567");
}

/// @brief it should remove the scale and offset from a sample, and scale a float32
/// sample from its shortest decimal text at float64 precision.
TEST(TextEncoder, EncodeScaled) {
    auto f = at("a", 0);
    f.scale = 0.5;
    f.offset = 10;
    const auto m = message("cmd", {f});
    const auto line = encode(m, series_of(x::telem::Series(20.0)));
    EXPECT_EQ(line, "20");
    assert_decodes(m, line, {20});
    auto doubled = f;
    doubled.scale = 2;
    doubled.offset = 0;
    EXPECT_EQ(
        encode(
            message("cmd", {doubled}, "", x::telem::FLOAT32_T),
            series_of(x::telem::Series(2.2F))
        ),
        "1.1"
    );
    auto shifted = f;
    shifted.scale = 1;
    shifted.offset = -1e6;
    const auto narrow = message("cmd", {shifted}, "", x::telem::FLOAT32_T);
    EXPECT_EQ(encode(narrow, series_of(x::telem::Series(0.001F))), "1000000.001");
}

/// @brief it should write the default of a field with no binding or no sample.
TEST(TextEncoder, EncodeDefaults) {
    auto a = at("a", 0);
    a.default_ = 7;
    auto b = at("b", 1);
    b.default_ = 2.5;
    TextMessage m = message("cmd", {a});
    m.payload.fields.emplace_back(b);
    EXPECT_EQ(encode(m, series_of(none())), "7,2.5");
    EXPECT_EQ(encode(m, series_of(x::telem::Series(3.0))), "3,2.5");
}

/// @brief it should reject a field with no sample and no default, and a value that is
/// not finite.
TEST(TextEncoder, RejectInvalidValues) {
    const auto m = message("cmd", {at("a", 0)});
    assert_rejects_samples(
        m,
        series_of(none()),
        "message cmd: field a: no value and no default"
    );
    assert_rejects_samples(
        m,
        series_of(x::telem::Series(std::numeric_limits<double>::infinity())),
        "message cmd: field a: Infinity is not finite"
    );
    assert_rejects_samples(
        m,
        series_of(x::telem::Series(std::numeric_limits<double>::quiet_NaN())),
        "message cmd: field a: NaN is not finite"
    );
    assert_rejects_samples(
        message("cmd", {at("a", 0)}, "", x::telem::FLOAT32_T),
        series_of(x::telem::Series(std::numeric_limits<float>::infinity())),
        "message cmd: field a: Infinity is not finite"
    );
}

/// @brief it should refuse two fields at one position, a position past 65535, and a
/// field at a position that the prefix fills.
TEST(TextEncoder, RefuseConflictingFields) {
    assert_encoder_refuses(
        message("cmd", {at("a", 1), at("b", 1)}),
        "message cmd: fields a and b both have position 1"
    );
    assert_encoder_refuses(
        message("cmd", {at("a", 4000000000)}),
        "message cmd: field a: position 4000000000 is past the last position 65535"
    );
    ASSERT_NIL_P(TextEncoder::create(message("cmd", {at("a", 65535)})));
    assert_encoder_refuses(
        message("cmd", {at("a", 0)}, "$SET"),
        "message cmd: field a: the prefix fills position 0"
    );
}

/// @brief it should refuse an empty delimiter, an invalid tag, a binding to a field
/// that does not exist, and two bindings of one field.
TEST(TextEncoder, RefuseInvalidMessage) {
    auto empty = message("cmd", {at("a", 0)});
    empty.payload.delimiter = "";
    assert_encoder_refuses(empty, "message cmd: the delimiter is empty");
    assert_encoder_refuses(
        message("cmd", {tagged("t", "")}),
        "message cmd: field t: the tag is empty"
    );
    auto unknown = message("cmd", {at("a", 0)});
    const auto key = x::uuid::create();
    unknown.bindings[0].field = key;
    assert_encoder_refuses(
        unknown,
        "message cmd: field " + key.to_string() + " does not exist"
    );
    auto twice = message("cmd", {at("a", 0)});
    twice.bindings.push_back(twice.bindings[0]);
    assert_encoder_refuses(twice, "message cmd: field a has more than one binding");
}

/// @brief it should refuse a field with no binding and no default, a default that is
/// not finite, and a channel of the wrong data type.
TEST(TextEncoder, RefuseInvalidFields) {
    auto unbound = message("cmd", {at("a", 0)});
    unbound.payload.fields.emplace_back(at("b", 1));
    assert_encoder_refuses(
        unbound,
        "message cmd: field b: needs a channel or a default"
    );
    auto nan = at("a", 0);
    nan.default_ = std::numeric_limits<double>::quiet_NaN();
    assert_encoder_refuses(
        message("cmd", {nan}),
        "message cmd: field a: default: NaN is not finite"
    );
    assert_encoder_refuses(
        message("cmd", {at("a", 0)}, "", x::telem::STRING_T),
        "message cmd: field a: cannot write string channels to float fields"
    );
    auto scaled = at("a", 0);
    scaled.scale = 2;
    assert_encoder_refuses(
        message("cmd", {scaled}, "", x::telem::INT32_T),
        "message cmd: field a: a scaled field requires a float channel, not int32"
    );
}

/// @brief it should refuse a bound field with a zero scale, and write the default of an
/// unbound one.
TEST(TextEncoder, RefuseZeroScale) {
    auto zero = at("a", 0);
    zero.scale = 0;
    assert_encoder_refuses(
        message("cmd", {zero}),
        "message cmd: field a: scale must be non-zero"
    );
    auto unbound = at("b", 1);
    unbound.scale = 0;
    unbound.default_ = 7;
    auto m = message("cmd", {at("a", 0)});
    m.payload.fields.emplace_back(unbound);
    EXPECT_EQ(encode(m, series_of(x::telem::Series(3.0))), "3,7");
}

/// @brief it should encode without allocating once its scratch space has grown.
TEST(TextEncoder, EncodeWithoutAllocating) {
    auto t = tagged("t", "T=");
    t.default_ = 2.5;
    TextMessage m = message("cmd", {at("a", 1)}, "$SET", x::telem::INT64_T);
    m.payload.fields.emplace_back(t);
    auto enc = ASSERT_NIL_P(TextEncoder::create(m));
    const auto samples = series_of(x::telem::Series(std::int64_t{-123456789}));
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(enc.encode(samples, payload));
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++)
        ASSERT_NIL(enc.encode(samples, payload));
    EXPECT_EQ(x::bench::alloc_count.load(std::memory_order_relaxed) - before, 0);
    EXPECT_EQ(std::string(payload.begin(), payload.end()), "$SET,-123456789,T=2.5");
}

/// @brief it should check a sample of a binding as encode does, and report which fields
/// have a default.
TEST(TextEncoder, CheckSample) {
    auto b = at("b", 1);
    b.default_ = 1;
    auto m = message("cmd", {at("a", 0), b});
    auto enc = ASSERT_NIL_P(TextEncoder::create(m));
    const x::telem::Series s(std::vector<double>{2.5, std::nan("")});
    ASSERT_NIL(enc.check(0, s, 0));
    ASSERT_NIL(enc.check(1, s, 0));
    const auto err = enc.check(1, s, 1);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "message cmd: field b: NaN is not finite");
    EXPECT_FALSE(enc.has_default(0));
    EXPECT_TRUE(enc.has_default(1));
}

namespace {
/// @brief asserts that enc refuses sample 0 of s for binding with text.
void assert_check_refuses(
    TextEncoder &enc,
    const std::size_t binding,
    const x::telem::Series &s,
    const std::string &text
) {
    const auto err = enc.check(binding, s, 0);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, text);
}
}

/// @brief it should refuse a sample outside the min and max of its field and accept one
/// at a limit, before the scale and offset come off. A sample that the field cannot
/// hold fails as encode fails.
TEST(TextEncoder, CheckLimits) {
    auto a = at("a", 0);
    a.min = 0;
    a.max = 10;
    auto s = at("s", 1);
    s.scale = 0.1;
    s.max = 10;
    auto enc = ASSERT_NIL_P(TextEncoder::create(message("cmd", {a, s})));
    ASSERT_NIL(enc.check(0, x::telem::Series(0.0), 0));
    ASSERT_NIL(enc.check(0, x::telem::Series(10.0), 0));
    ASSERT_NIL(enc.check(1, x::telem::Series(5.0), 0));
    assert_check_refuses(
        enc,
        0,
        x::telem::Series(10.5),
        "message cmd: field a: 10.5 is above the max 10"
    );
    assert_check_refuses(
        enc,
        0,
        x::telem::Series(-1.0),
        "message cmd: field a: -1 is below the min 0"
    );
    assert_check_refuses(
        enc,
        0,
        x::telem::Series(std::numeric_limits<double>::infinity()),
        "message cmd: field a: Infinity is not finite"
    );
    auto ints = ASSERT_NIL_P(
        TextEncoder::create(message("cmd", {a}, "", x::telem::INT64_T))
    );
    assert_check_refuses(
        ints,
        0,
        x::telem::Series(std::int64_t{11}),
        "message cmd: field a: 11 is above the max 10"
    );
}

/// @brief it should accept a float32 sample that equals a limit at float32 precision,
/// and refuse one past it with its exact value.
TEST(TextEncoder, CheckFloat32Limits) {
    auto a = at("a", 0);
    a.max = 0.1;
    auto b = at("b", 1);
    b.min = 3.3;
    auto enc = ASSERT_NIL_P(
        TextEncoder::create(message("cmd", {a, b}, "", x::telem::FLOAT32_T))
    );
    ASSERT_NIL(enc.check(0, x::telem::Series(0.1F), 0));
    ASSERT_NIL(enc.check(1, x::telem::Series(3.3F), 0));
    assert_check_refuses(
        enc,
        0,
        x::telem::Series(std::nextafter(0.1F, 1.0F)),
        "message cmd: field a: 0.10000000894069672 is above the max 0.1"
    );
    assert_check_refuses(
        enc,
        1,
        x::telem::Series(std::nextafter(3.3F, 0.0F)),
        "message cmd: field b: 3.299999713897705 is below the min 3.3"
    );
}
}
