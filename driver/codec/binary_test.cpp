// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <bit>
#include <cmath>
#include <cstdint>
#include <limits>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/bench/alloc.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/binary.h"
#include "driver/codec/errors.h"

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

/// @brief returns the error that creating a decoder for one field f bound to a channel
/// of dt gives.
x::errors::Error create_error(const icd::BinaryField &f, const x::telem::DataType &dt) {
    return Binary::create({frame(f, dt)}).second;
}

/// @brief asserts that creating a decoder for f bound to dt fails with text.
void assert_refuses(
    const icd::BinaryField &f,
    const x::telem::DataType &dt,
    const std::string &text
) {
    const auto err = create_error(f, dt);
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "message frame: field " + icd::base(f).name + ": " + text);
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

/// @brief asserts that creating a decoder for m fails with text.
void assert_refuses_message(const BinaryMessage &m, const std::string &text) {
    const auto [dec, err] = Binary::create({m});
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, text);
}

/// @brief returns the one sample that a decoder for f bound to dt decodes from bytes.
template<typename T>
T decode_one(
    const icd::BinaryField &f,
    const x::telem::DataType &dt,
    const std::vector<std::uint8_t> &bytes
) {
    auto dec = ASSERT_NIL_P(Binary::create({frame(f, dt)}));
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(bytes)), 0U);
    const auto &s = dec.samples(0);
    EXPECT_EQ(s.size(), 1);
    EXPECT_EQ(s[0].size(), 1);
    return s[0].at<T>(0);
}

std::vector<std::uint8_t> le32(const std::uint32_t v) {
    return {
        static_cast<std::uint8_t>(v),
        static_cast<std::uint8_t>(v >> 8),
        static_cast<std::uint8_t>(v >> 16),
        static_cast<std::uint8_t>(v >> 24),
    };
}
}

/// @brief it should write an unsigned field to an unsigned channel exactly.
TEST(Binary, DecodeUnsigned) {
    const auto f = field<icd::UnsignedBinaryField>("rpm", 4, 12);
    EXPECT_EQ(decode_one<std::uint16_t>(f, x::telem::UINT16_T, {0xB0, 0xCA}), 0xCAB);
}

/// @brief it should write 64-bit integers exactly, past the precision of a float64.
TEST(Binary, Decode64BitExact) {
    const std::vector<std::uint8_t> ones(8, 0xFF);
    const auto u = field<icd::UnsignedBinaryField>("u", 0, 64);
    EXPECT_EQ(
        decode_one<std::uint64_t>(u, x::telem::UINT64_T, ones),
        std::numeric_limits<std::uint64_t>::max()
    );
    const std::vector<std::uint8_t> big = {0x01, 0, 0, 0, 0, 0, 0x20, 0x80};
    const auto i = field<icd::IntegerBinaryField>("i", 0, 64);
    EXPECT_EQ(
        decode_one<std::int64_t>(i, x::telem::INT64_T, big),
        std::numeric_limits<std::int64_t>::min() + (std::int64_t{1} << 53) + 1
    );
}

/// @brief it should write a big-endian signed field to a signed channel.
TEST(Binary, DecodeBigEndianSigned) {
    auto f = field<icd::IntegerBinaryField>("temp", 7, 16);
    f.byte_order = icd::BYTE_ORDER_BIG_ENDIAN;
    EXPECT_EQ(decode_one<std::int16_t>(f, x::telem::INT16_T, {0xFF, 0x38}), -200);
}

/// @brief it should write an unsigned field to a signed channel wide enough for it.
TEST(Binary, DecodeUnsignedToWiderSigned) {
    const auto f = field<icd::UnsignedBinaryField>("count", 0, 8);
    EXPECT_EQ(decode_one<std::int16_t>(f, x::telem::INT16_T, {0xFF}), 255);
}

/// @brief it should read 32 and 64-bit IEEE 754 floats.
TEST(Binary, DecodeFloats) {
    const auto f32 = field<icd::FloatBinaryField>("f32", 0, 32);
    const auto bytes32 = le32(std::bit_cast<std::uint32_t>(1.5F));
    EXPECT_EQ(decode_one<float>(f32, x::telem::FLOAT32_T, bytes32), 1.5F);
    const auto f64 = field<icd::FloatBinaryField>("f64", 0, 64);
    const auto bits = std::bit_cast<std::uint64_t>(-2.25);
    auto bytes = le32(static_cast<std::uint32_t>(bits));
    const auto high = le32(static_cast<std::uint32_t>(bits >> 32));
    bytes.insert(bytes.end(), high.begin(), high.end());
    EXPECT_EQ(decode_one<double>(f64, x::telem::FLOAT64_T, bytes), -2.25);
}

/// @brief it should apply scale, then offset, to the raw value.
TEST(Binary, DecodeScaled) {
    auto f = field<icd::IntegerBinaryField>("temp", 0, 8);
    f.scale = 0.5;
    f.offset = -40;
    EXPECT_EQ(decode_one<double>(f, x::telem::FLOAT64_T, {0xFE}), -41);
}

/// @brief it should write an unscaled integer field to a float channel.
TEST(Binary, DecodeIntegerToFloat) {
    const auto f = field<icd::IntegerBinaryField>("count", 0, 8);
    EXPECT_EQ(decode_one<float>(f, x::telem::FLOAT32_T, {0x80}), -128);
}

/// @brief it should write the default of a field that a short payload does not hold,
/// with scale and offset.
TEST(Binary, DecodeDefault) {
    auto f = field<icd::UnsignedBinaryField>("rpm", 8, 8);
    f.scale = 2;
    f.default_ = 21;
    EXPECT_EQ(decode_one<double>(f, x::telem::FLOAT64_T, {0x01}), 42);
}

/// @brief it should decode the bound fields of a frame in binding order, and leave the
/// unbound ones out.
TEST(Binary, DecodeBindingOrder) {
    const auto a = field<icd::UnsignedBinaryField>("a", 0, 8);
    const auto b = field<icd::UnsignedBinaryField>("b", 8, 8);
    const auto c = field<icd::UnsignedBinaryField>("c", 16, 8);
    auto dec = ASSERT_NIL_P(
        Binary::create({{
            .name = "frame",
            .payload = {.fields = {a, b, c}},
            .bindings = {
                {.field = c.key, .data_type = x::telem::UINT8_T},
                {.field = a.key, .data_type = x::telem::UINT8_T},
            },
        }})
    );
    const std::vector<std::uint8_t> bytes = {1, 2, 3};
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(bytes)), 0U);
    const auto &s = dec.samples(0);
    ASSERT_EQ(s.size(), 2);
    EXPECT_EQ(s[0].at<std::uint8_t>(0), 3);
    EXPECT_EQ(s[1].at<std::uint8_t>(0), 1);
}

/// @brief it should reject a payload too short for a field with no default.
TEST(Binary, RejectShortPayload) {
    const auto f = field<icd::UnsignedBinaryField>("rpm", 8, 16);
    auto dec = ASSERT_NIL_P(Binary::create({frame(f, x::telem::UINT16_T)}));
    const std::vector<std::uint8_t> bytes = {0x01, 0x02};
    const auto [message, err] = dec.decode(bytes);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "field rpm: the payload has 2 bytes and the field needs 3");
}

/// @brief it should reject a value too large for a float32 channel.
TEST(Binary, RejectFloat32Overflow) {
    const auto f = field<icd::FloatBinaryField>("v", 0, 64);
    auto dec = ASSERT_NIL_P(Binary::create({frame(f, x::telem::FLOAT32_T)}));
    const auto bits = std::bit_cast<std::uint64_t>(1e300);
    auto bytes = le32(static_cast<std::uint32_t>(bits));
    const auto high = le32(static_cast<std::uint32_t>(bits >> 32));
    bytes.insert(bytes.end(), high.begin(), high.end());
    const auto [message, err] = dec.decode(bytes);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "field v: 1e+300 overflows float32");
}

/// @brief it should refuse a float field that is not 32 or 64 bits.
TEST(Binary, RefuseFloatWidth) {
    assert_refuses(
        field<icd::FloatBinaryField>("v", 0, 16),
        x::telem::FLOAT32_T,
        "a float field must be 32 or 64 bits, not 16"
    );
}

/// @brief it should refuse a field kind that does not convert to the channel.
TEST(Binary, RefuseKindConversion) {
    assert_refuses(
        field<icd::FloatBinaryField>("v", 0, 32),
        x::telem::INT32_T,
        "cannot write float values to int32 channels"
    );
    assert_refuses(
        field<icd::IntegerBinaryField>("v", 0, 8),
        x::telem::UINT8_T,
        "cannot write integer values to uint8 channels"
    );
}

/// @brief it should refuse an integer channel that cannot hold every value of the
/// field.
TEST(Binary, RefuseNarrowChannel) {
    assert_refuses(
        field<icd::UnsignedBinaryField>("v", 0, 16),
        x::telem::UINT8_T,
        "cannot write 16-bit unsigned values to uint8 channels"
    );
    assert_refuses(
        field<icd::UnsignedBinaryField>("v", 0, 8),
        x::telem::INT8_T,
        "cannot write 8-bit unsigned values to int8 channels"
    );
}

/// @brief it should refuse a scaled field on a channel that is not a float.
TEST(Binary, RefuseScaledInteger) {
    auto f = field<icd::UnsignedBinaryField>("v", 0, 8);
    f.scale = 0.1;
    assert_refuses(
        f,
        x::telem::INT32_T,
        "a scaled field requires a float channel, not int32"
    );
}

/// @brief it should refuse a default that the bits of its field cannot hold.
TEST(Binary, RefuseDefaultOutOfRange) {
    auto u = field<icd::UnsignedBinaryField>("v", 0, 8);
    u.default_ = 300;
    assert_refuses(u, x::telem::UINT16_T, "default: 300 is out of range for 8 bits");
    auto i = field<icd::IntegerBinaryField>("v", 0, 8);
    i.default_ = -129;
    assert_refuses(i, x::telem::INT16_T, "default: -129 is out of range for 8 bits");
    auto f = field<icd::FloatBinaryField>("v", 0, 32);
    f.default_ = 1e300;
    assert_refuses(f, x::telem::FLOAT64_T, "default: 1e+300 overflows float32");
}

/// @brief it should refuse a default that is not finite.
TEST(Binary, RefuseDefaultNotFinite) {
    auto f = field<icd::FloatBinaryField>("v", 0, 64);
    f.default_ = std::numeric_limits<double>::quiet_NaN();
    assert_refuses(f, x::telem::FLOAT64_T, "default: NaN is not finite");
}

/// @brief it should decode only the fields that the multiplexor of a frame selects.
TEST(Binary, DecodeMultiplexed) {
    const auto mode = field<icd::UnsignedBinaryField>("mode", 0, 8);
    auto a = field<icd::UnsignedBinaryField>("a", 8, 8);
    a.multiplexor = mode.key;
    a.multiplex_values = {1};
    auto b = field<icd::UnsignedBinaryField>("b", 8, 8);
    b.multiplexor = mode.key;
    b.multiplex_values = {2, 3};
    auto dec = ASSERT_NIL_P(
        Binary::create({multiplexed({mode, a, b}, {a.key, b.key})})
    );
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(std::vector<std::uint8_t>{1, 16})), 0U);
    ASSERT_EQ(dec.samples(0)[0].size(), 1);
    EXPECT_EQ(dec.samples(0)[0].at<std::uint8_t>(0), 16);
    EXPECT_EQ(dec.samples(0)[1].size(), 0);
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(std::vector<std::uint8_t>{3, 32})), 0U);
    EXPECT_EQ(dec.samples(0)[0].size(), 0);
    ASSERT_EQ(dec.samples(0)[1].size(), 1);
    EXPECT_EQ(dec.samples(0)[1].at<std::uint8_t>(0), 32);
}

/// @brief it should write the default of a field that the multiplexor does not select.
TEST(Binary, DecodeMultiplexedDefault) {
    const auto mode = field<icd::UnsignedBinaryField>("mode", 0, 8);
    auto a = field<icd::UnsignedBinaryField>("a", 8, 8);
    a.multiplexor = mode.key;
    a.multiplex_values = {1};
    a.default_ = 7;
    auto dec = ASSERT_NIL_P(Binary::create({multiplexed({mode, a}, {a.key})}));
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(std::vector<std::uint8_t>{2, 16})), 0U);
    ASSERT_EQ(dec.samples(0)[0].size(), 1);
    EXPECT_EQ(dec.samples(0)[0].at<std::uint8_t>(0), 7);
}

/// @brief it should select a field only when each multiplexor in its chain holds one of
/// its values.
TEST(Binary, DecodeCascadedMultiplexors) {
    const auto outer = field<icd::UnsignedBinaryField>("outer", 0, 8);
    auto inner = field<icd::IntegerBinaryField>("inner", 8, 8);
    inner.multiplexor = outer.key;
    inner.multiplex_values = {1};
    auto v = field<icd::UnsignedBinaryField>("v", 16, 8);
    v.multiplexor = inner.key;
    v.multiplex_values = {-1};
    auto dec = ASSERT_NIL_P(Binary::create({multiplexed({outer, inner, v}, {v.key})}));
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(std::vector<std::uint8_t>{1, 0xFF, 9})), 0U);
    ASSERT_EQ(dec.samples(0)[0].size(), 1);
    EXPECT_EQ(dec.samples(0)[0].at<std::uint8_t>(0), 9);
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(std::vector<std::uint8_t>{0, 0xFF, 9})), 0U);
    EXPECT_EQ(dec.samples(0)[0].size(), 0);
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(std::vector<std::uint8_t>{1, 0xFE, 9})), 0U);
    EXPECT_EQ(dec.samples(0)[0].size(), 0);
}

/// @brief it should reject a payload too short for its multiplexors.
TEST(Binary, RejectShortForMultiplexors) {
    const auto mode = field<icd::UnsignedBinaryField>("mode", 8, 8);
    auto a = field<icd::UnsignedBinaryField>("a", 0, 8);
    a.multiplexor = mode.key;
    a.multiplex_values = {1};
    a.default_ = 0;
    auto dec = ASSERT_NIL_P(Binary::create({multiplexed({mode, a}, {a.key})}));
    const auto [message, err] = dec.decode(std::vector<std::uint8_t>{1});
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "the payload has 1 bytes and its multiplexors need 2");
}

/// @brief it should refuse a multiplexor that is not a field of the message.
TEST(Binary, RefuseMissingMultiplexor) {
    auto a = field<icd::UnsignedBinaryField>("a", 8, 8);
    const auto key = x::uuid::create();
    a.multiplexor = key;
    a.multiplex_values = {1};
    assert_refuses(
        a,
        x::telem::UINT8_T,
        "multiplexor " + key.to_string() + " does not exist"
    );
}

/// @brief it should refuse a float multiplexor.
TEST(Binary, RefuseFloatMultiplexor) {
    const auto mode = field<icd::FloatBinaryField>("mode", 0, 32);
    auto a = field<icd::UnsignedBinaryField>("a", 32, 8);
    a.multiplexor = mode.key;
    a.multiplex_values = {1};
    assert_refuses_message(
        multiplexed({mode, a}, {a.key}),
        "message frame: field a: multiplexor mode is a float field"
    );
}

/// @brief it should refuse a multiplexed field with no multiplex values.
TEST(Binary, RefuseEmptyMultiplexValues) {
    const auto mode = field<icd::UnsignedBinaryField>("mode", 0, 8);
    auto a = field<icd::UnsignedBinaryField>("a", 8, 8);
    a.multiplexor = mode.key;
    assert_refuses_message(
        multiplexed({mode, a}, {a.key}),
        "message frame: field a: multiplex values are empty"
    );
}

/// @brief it should refuse multiplexors that select each other.
TEST(Binary, RefuseMultiplexorCycle) {
    auto x = field<icd::UnsignedBinaryField>("x", 0, 8);
    auto y = field<icd::UnsignedBinaryField>("y", 8, 8);
    x.multiplexor = y.key;
    x.multiplex_values = {1};
    y.multiplexor = x.key;
    y.multiplex_values = {1};
    auto a = field<icd::UnsignedBinaryField>("a", 16, 8);
    a.multiplexor = x.key;
    a.multiplex_values = {1};
    assert_refuses_message(
        multiplexed({x, y, a}, {a.key}),
        "message frame: field a: multiplexors form a cycle"
    );
}

/// @brief it should refuse a field that ends past the length of its payload.
TEST(Binary, RefuseFieldPastLength) {
    const auto f = field<icd::UnsignedBinaryField>("v", 4, 8);
    auto m = frame(f, x::telem::UINT8_T);
    m.payload.length = 1;
    const auto [dec, err] = Binary::create({m});
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "message frame: field v: the field ends past the 1-byte payload"
    );
}

/// @brief it should refuse a binding to a field that its message does not hold.
TEST(Binary, RefuseUnknownField) {
    auto m = frame(field<icd::UnsignedBinaryField>("v", 0, 8), x::telem::UINT8_T);
    const auto key = x::uuid::create();
    m.bindings[0].field = key;
    const auto [dec, err] = Binary::create({m});
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "message frame: field " + key.to_string() + " does not exist");
}

/// @brief it should refuse two messages that both have no identifier.
TEST(Binary, RefuseTwoFallbacks) {
    auto a = frame(field<icd::UnsignedBinaryField>("v", 0, 8), x::telem::UINT8_T);
    auto b = a;
    b.name = "other";
    const auto [dec, err] = Binary::create({a, b});
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "messages frame and other both have no identifier");
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
}
