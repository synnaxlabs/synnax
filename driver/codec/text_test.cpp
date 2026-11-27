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
#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/bench/alloc.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

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

/// @brief returns an empty series, which takes the default of its field.
x::telem::Series none() {
    return x::telem::Series(x::telem::FLOAT64_T, 0);
}
}

/// @brief it should reject a field with no sample and no default.
TEST(TextEncoder, RejectMissingSample) {
    auto enc = ASSERT_NIL_P(TextEncoder::create(message("cmd", {at("a", 0)})));
    std::vector<std::uint8_t> payload;
    const auto err = enc.encode(series_of(none()), payload);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "message cmd: field a: no value and no default");
}

/// @brief it should accept a field at the last position, 65535.
TEST(TextEncoder, AcceptLastPosition) {
    ASSERT_NIL_P(TextEncoder::create(message("cmd", {at("a", 65535)})));
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

/// @brief it should check a sample of a binding as encode does.
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
}

/// @brief it should be ready when each field has a sample or a default.
TEST(TextEncoder, Ready) {
    auto b = at("b", 1);
    b.default_ = 1;
    const auto enc = ASSERT_NIL_P(TextEncoder::create(message("cmd", {at("a", 0), b})));
    EXPECT_FALSE(enc.ready(series_of(none(), none())));
    EXPECT_TRUE(enc.ready(series_of(x::telem::Series(2.5), none())));
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
