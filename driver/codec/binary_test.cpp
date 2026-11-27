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
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/bench/alloc.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/binary.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;

/// @brief returns a little-endian field of type F named name.
template<typename F>
F field(const std::string &name, const std::uint16_t start, const std::uint8_t length) {
    F f;
    f.key = x::uuid::create();
    f.name = name;
    f.start_bit = start;
    f.bit_length = length;
    return f;
}

/// @brief returns a message named frame with one field f, bound to a channel of dt.
BinaryMessage frame(const icd::BinaryField &f, const x::telem::DataType &dt) {
    return {
        .name = "frame",
        .payload = {.fields = {f}},
        .bindings = {{.field = icd::base(f).key, .data_type = dt}},
    };
}

/// @brief returns a message named frame with fields, which binds the fields of keys to
/// uint8 channels.
BinaryMessage multiplexed(
    const std::vector<icd::BinaryField> &fields,
    const std::vector<icd::FieldKey> &keys
) {
    BinaryMessage m{.name = "frame", .payload = {.fields = fields}};
    for (const auto &k: keys)
        m.bindings.push_back({.field = k, .data_type = x::telem::UINT8_T});
    return m;
}

/// @brief returns a message named name with one uint8 field at byte 1, bound to a uint8
/// channel, and the identifier id.
BinaryMessage
identified(const std::string &name, const std::optional<icd::Identifier> &id) {
    auto m = frame(field<icd::UnsignedBinaryField>("v", 8, 8), x::telem::UINT8_T);
    m.name = name;
    m.payload.identifier = id;
    return m;
}

icd::Identifier
can(const std::uint32_t id,
    const bool extended = false,
    const std::optional<std::uint32_t> mask = std::nullopt) {
    icd::CanIdentifier c;
    c.id = id;
    c.extended = extended;
    c.mask = mask;
    return c;
}

/// @brief returns a message named name whose field kind, at byte 0, holds value.
BinaryMessage tagged(const std::string &name, const std::int32_t value) {
    auto m = identified(name, std::nullopt);
    const auto kind = field<icd::UnsignedBinaryField>("kind", 0, 8);
    m.payload.fields.emplace_back(kind);
    icd::FieldIdentifier id;
    id.field = kind.key;
    id.value = value;
    m.payload.identifier = id;
    return m;
}

const std::vector<std::uint8_t> PAYLOAD = {1, 42};
}

/// @brief it should decode a frame without allocating once a first decode is done.
TEST(Binary, DecodeWithoutAllocating) {
    auto f = field<icd::IntegerBinaryField>("temp", 0, 16);
    f.scale = 0.1;
    auto dec = ASSERT_NIL_P(Binary::create({frame(f, x::telem::FLOAT64_T)}));
    const std::vector<std::uint8_t> bytes = {0x10, 0x27};
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(bytes)), 0U);
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++)
        ASSERT_EQ(ASSERT_NIL_P(dec.decode(bytes)), 0U);
    EXPECT_EQ(x::bench::alloc_count.load(std::memory_order_relaxed) - before, 0);
    EXPECT_NEAR(dec.samples(0)[0].at<double>(0), 1000, 1e-9);
}

/// @brief it should route and decode frames without allocating once a first decode is
/// done.
TEST(Binary, MatchWithoutAllocating) {
    auto dec = ASSERT_NIL_P(
        Binary::create({
            identified("exact", can(0x100)),
            identified("masked", can(0x200, false, 0x700)),
            tagged("field", 1),
        })
    );
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(0x100, false, PAYLOAD)), 0U);
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(0x2AB, false, PAYLOAD)), 1U);
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(PAYLOAD)), 2U);
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++) {
        ASSERT_EQ(ASSERT_NIL_P(dec.decode(0x100, false, PAYLOAD)), 0U);
        ASSERT_EQ(ASSERT_NIL_P(dec.decode(0x2AB, false, PAYLOAD)), 1U);
        ASSERT_EQ(ASSERT_NIL_P(dec.decode(PAYLOAD)), 2U);
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

/// @brief returns an empty series of dt, which takes the default of its field.
x::telem::Series none(const x::telem::DataType &dt = x::telem::UINT8_T) {
    return x::telem::Series(dt, 0);
}

/// @brief returns a uint8 field named name in the byte at offset that the field mux
/// selects with values.
icd::UnsignedBinaryField selected_by(
    const std::string &name,
    const std::uint16_t offset,
    const icd::FieldKey &mux,
    const std::vector<std::int32_t> &values
) {
    auto f = field<icd::UnsignedBinaryField>(name, offset * 8, 8);
    f.multiplexor = mux;
    f.multiplex_values = values;
    return f;
}
}

/// @brief it should encode a scaled sample that a decoder of the same message reads
/// back.
TEST(BinaryEncoder, RoundTripScaled) {
    auto f = field<icd::IntegerBinaryField>("v", 0, 16);
    f.scale = 0.1;
    f.offset = -40;
    const auto m = frame(f, x::telem::FLOAT64_T);
    auto enc = ASSERT_NIL_P(BinaryEncoder::create(m));
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(enc.encode(series_of(x::telem::Series(21.5)), payload));
    auto dec = ASSERT_NIL_P(Binary::create({m}));
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(payload)), 0U);
    // A platform that fuses the multiply and add of the decode changes the last bit.
    EXPECT_NEAR(dec.samples(0)[0].at<double>(0), 21.5, 1e-9);
}

/// @brief it should reject a field with no sample and no default.
TEST(BinaryEncoder, RejectMissingValue) {
    const auto m = frame(field<icd::UnsignedBinaryField>("v", 0, 8), x::telem::UINT8_T);
    auto enc = ASSERT_NIL_P(BinaryEncoder::create(m));
    std::vector<std::uint8_t> payload;
    const auto err = enc.encode(series_of(none()), payload);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data(), "message frame: field v: no value and no default");
}

/// @brief it should encode without allocating once its payload has room.
TEST(BinaryEncoder, EncodeWithoutAllocating) {
    auto f = field<icd::IntegerBinaryField>("temp", 0, 16);
    f.scale = 0.1;
    auto enc = ASSERT_NIL_P(BinaryEncoder::create(frame(f, x::telem::FLOAT64_T)));
    const auto samples = series_of(x::telem::Series(0.1 + 0.2));
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(enc.encode(samples, payload));
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++)
        ASSERT_NIL(enc.encode(samples, payload));
    EXPECT_EQ(x::bench::alloc_count.load(std::memory_order_relaxed) - before, 0);
    EXPECT_EQ(payload, (std::vector<std::uint8_t>{0x03, 0x00}));
}

/// @brief it should refuse a sample outside the min and max of its field, and accept a
/// float32 sample that equals a limit at float32 precision.
TEST(BinaryEncoder, CheckSampleLimits) {
    auto v = field<icd::FloatBinaryField>("v", 0, 32);
    v.min = -1;
    v.max = 0.1;
    const auto enc = ASSERT_NIL_P(BinaryEncoder::create(frame(v, x::telem::FLOAT32_T)));
    ASSERT_NIL(enc.check(0, x::telem::Series(0.1F), 0));
    auto err = enc.check(0, x::telem::Series(0.5F), 0);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data(), "message frame: field v: 0.5 is above the max 0.1");
    err = enc.check(0, x::telem::Series(-2.0F), 0);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data(), "message frame: field v: -2 is below the min -1");
}

/// @brief it should check the value that a sample rounds to, so a sample under the max
/// that rounds to a raw value past it is refused.
TEST(BinaryEncoder, CheckSampleLimitsAfterRounding) {
    auto v = field<icd::IntegerBinaryField>("v", 0, 8);
    v.scale = 5;
    v.max = 13;
    const auto enc = ASSERT_NIL_P(BinaryEncoder::create(frame(v, x::telem::FLOAT64_T)));
    ASSERT_NIL(enc.check(0, x::telem::Series(12.0), 0));
    const auto err = enc.check(0, x::telem::Series(13.0), 0);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data(),
        "message frame: field v: 3 decodes to 15, which is above the max 13"
    );
}

/// @brief it should check the float32 value of a float64 sample for a float32 field,
/// and accept one that equals a limit at float32 precision.
TEST(BinaryEncoder, CheckFloat32FieldLimits) {
    auto v = field<icd::FloatBinaryField>("v", 0, 32);
    v.max = 0.1;
    const auto enc = ASSERT_NIL_P(BinaryEncoder::create(frame(v, x::telem::FLOAT64_T)));
    ASSERT_NIL(enc.check(0, x::telem::Series(0.1), 0));
    const auto err = enc.check(0, x::telem::Series(0.10000001), 0);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data(),
        "message frame: field v: 0.10000000894069672 is above the max 0.1"
    );
}

/// @brief it should be ready when each field has a sample or a default.
TEST(BinaryEncoder, Ready) {
    const auto v = field<icd::UnsignedBinaryField>("v", 0, 8);
    auto w = field<icd::UnsignedBinaryField>("w", 8, 8);
    w.default_ = 1;
    auto enc = ASSERT_NIL_P(BinaryEncoder::create(multiplexed({v, w}, {v.key, w.key})));
    EXPECT_FALSE(enc.ready(series_of(none(), none())));
    EXPECT_TRUE(enc.ready(series_of(x::telem::Series(std::uint8_t{1}), none())));
}

/// @brief it should need samples only for the multiplexor and the fields that its value
/// selects.
TEST(BinaryEncoder, ReadyForSelectedFields) {
    const auto mode = field<icd::UnsignedBinaryField>("mode", 0, 8);
    const auto a = selected_by("a", 1, mode.key, {0});
    const auto b = selected_by("b", 2, mode.key, {1});
    auto enc = ASSERT_NIL_P(
        BinaryEncoder::create(multiplexed({mode, a, b}, {mode.key, a.key, b.key}))
    );
    const auto five = [] { return x::telem::Series(std::uint8_t{5}); };
    EXPECT_FALSE(enc.ready(series_of(none(), five(), five())));
    EXPECT_FALSE(
        enc.ready(series_of(x::telem::Series(std::uint8_t{0}), none(), five()))
    );
    EXPECT_TRUE(
        enc.ready(series_of(x::telem::Series(std::uint8_t{0}), five(), none()))
    );
    EXPECT_FALSE(
        enc.ready(series_of(x::telem::Series(std::uint8_t{1}), five(), none()))
    );
}

/// @brief it should refuse an infinite or NaN sample for a float field with a min and a
/// max.
TEST(BinaryEncoder, CheckNonFiniteSampleLimits) {
    for (const std::uint8_t bits: {32, 64}) {
        auto v = field<icd::FloatBinaryField>("v", 0, bits);
        v.min = -10;
        v.max = 10;
        const auto enc = ASSERT_NIL_P(
            BinaryEncoder::create(frame(v, x::telem::FLOAT64_T))
        );
        constexpr double inf = std::numeric_limits<double>::infinity();
        auto err = enc.check(0, x::telem::Series(inf), 0);
        ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
        EXPECT_EQ(err.data(), "message frame: field v: Infinity is above the max 10");
        err = enc.check(0, x::telem::Series(-inf), 0);
        ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
        EXPECT_EQ(err.data(), "message frame: field v: -Infinity is below the min -10");
        err = enc.check(0, x::telem::Series(std::nan("")), 0);
        ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
        EXPECT_EQ(err.data(), "message frame: field v: NaN is outside the limits");
    }
}

/// @brief it should accept a NaN sample for a float field with no min and no max.
TEST(BinaryEncoder, CheckNaNSampleWithoutLimits) {
    const auto enc = ASSERT_NIL_P(
        BinaryEncoder::create(
            frame(field<icd::FloatBinaryField>("v", 0, 64), x::telem::FLOAT64_T)
        )
    );
    ASSERT_NIL(enc.check(0, x::telem::Series(std::nan("")), 0));
}
}
