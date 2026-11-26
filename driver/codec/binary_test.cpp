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
#include <optional>
#include <string>
#include <utility>
#include <variant>
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

/// @brief asserts that creating a decoder for messages fails with text.
void assert_refuses_all(
    const std::vector<BinaryMessage> &messages,
    const std::string &text
) {
    const auto [dec, err] = Binary::create(messages);
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, text);
}

/// @brief asserts that creating a decoder for m fails with text.
void assert_refuses_message(const BinaryMessage &m, const std::string &text) {
    assert_refuses_all({m}, text);
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

/// @brief it should refuse a field with a zero scale.
TEST(Binary, RefuseZeroScale) {
    auto f = field<icd::UnsignedBinaryField>("v", 0, 8);
    f.scale = 0;
    assert_refuses(f, x::telem::FLOAT64_T, "scale must be non-zero");
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

/// @brief it should write a NaN default for a field that a short payload does not hold.
TEST(Binary, DecodeNaNDefault) {
    auto f = field<icd::FloatBinaryField>("v", 8, 32);
    f.default_ = std::numeric_limits<double>::quiet_NaN();
    EXPECT_TRUE(std::isnan(decode_one<double>(f, x::telem::FLOAT64_T, {0x01})));
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

/// @brief it should reject a payload too short for a multiplexor, and name the first
/// multiplexor that does not fit.
TEST(Binary, RejectShortForMultiplexors) {
    const auto bank = field<icd::UnsignedBinaryField>("bank", 0, 8);
    const auto mode = field<icd::UnsignedBinaryField>("mode", 16, 8);
    auto a = field<icd::UnsignedBinaryField>("a", 8, 8);
    a.multiplexor = bank.key;
    a.multiplex_values = {1};
    a.default_ = 0;
    auto b = field<icd::UnsignedBinaryField>("b", 8, 8);
    b.multiplexor = mode.key;
    b.multiplex_values = {1};
    b.default_ = 0;
    auto dec = ASSERT_NIL_P(
        Binary::create({multiplexed({bank, mode, a, b}, {a.key, b.key})})
    );
    const auto [message, err] = dec.decode(std::vector<std::uint8_t>{1, 0});
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "field mode: the payload has 2 bytes and the field needs 3");
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

namespace {
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
BinaryMessage tagged(
    const std::string &name,
    const std::int32_t value,
    const icd::BinaryField &kind = field<icd::UnsignedBinaryField>("kind", 0, 8)
) {
    auto m = identified(name, std::nullopt);
    m.payload.fields.push_back(kind);
    icd::FieldIdentifier id;
    id.field = icd::base(kind).key;
    id.value = value;
    m.payload.identifier = id;
    return m;
}

const std::vector<std::uint8_t> PAYLOAD = {1, 42};
}

/// @brief it should match a CAN frame by its exact standard or extended identifier.
TEST(Binary, MatchCANExact) {
    auto dec = ASSERT_NIL_P(
        Binary::create({
            identified("a", can(0x100)),
            identified("b", can(0x200)),
            identified("c", can(0x100, true)),
        })
    );
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(0x100, false, PAYLOAD)), 0U);
    EXPECT_EQ(dec.samples(0)[0].at<std::uint8_t>(0), 42);
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(0x200, false, PAYLOAD)), 1U);
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(0x100, true, PAYLOAD)), 2U);
    EXPECT_FALSE(ASSERT_NIL_P(dec.decode(0x300, false, PAYLOAD)).has_value());
}

/// @brief it should match a CAN frame by the bits of its identifier under a mask, and
/// give other frames to the message with no identifier.
TEST(Binary, MatchCANMasked) {
    auto dec = ASSERT_NIL_P(
        Binary::create({
            identified("pgn", can(0x18FEF100, true, 0x03FFFF00)),
            identified("other", std::nullopt),
        })
    );
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(0x1CFEF1AB, true, PAYLOAD)), 0U);
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(0x18FEF2AB, true, PAYLOAD)), 1U);
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(0x0F1, false, PAYLOAD)), 1U);
}

/// @brief it should match a frame by the value of its identifier field.
TEST(Binary, MatchFieldValue) {
    auto dec = ASSERT_NIL_P(Binary::create({tagged("a", 1), tagged("b", 2)}));
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(PAYLOAD)), 0U);
    EXPECT_EQ(dec.samples(0)[0].at<std::uint8_t>(0), 42);
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(std::vector<std::uint8_t>{2, 7})), 1U);
    EXPECT_FALSE(ASSERT_NIL_P(dec.decode(std::vector<std::uint8_t>{3, 7})).has_value());
    EXPECT_FALSE(ASSERT_NIL_P(dec.decode(std::vector<std::uint8_t>{})).has_value());
}

/// @brief it should match a signed identifier field by its two's complement value.
TEST(Binary, MatchSignedFieldValue) {
    const auto kind = field<icd::IntegerBinaryField>("kind", 0, 8);
    auto dec = ASSERT_NIL_P(Binary::create({tagged("a", -1, kind)}));
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(std::vector<std::uint8_t>{0xFF, 7})), 0U);
}

/// @brief it should match a CAN identifier only on a CAN frame, and a field identifier
/// only on a frame from a stream or a datagram.
TEST(Binary, MatchBySource) {
    auto dec = ASSERT_NIL_P(
        Binary::create({
            identified("can", can(0x001)),
            tagged("field", 1),
        })
    );
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(PAYLOAD)), 1U);
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(0x001, false, PAYLOAD)), 0U);
    EXPECT_FALSE(ASSERT_NIL_P(dec.decode(0x002, false, PAYLOAD)).has_value());
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

/// @brief it should reject a frame that meets the field identifiers of two messages.
TEST(Binary, RejectTwoFieldMatches) {
    auto dec = ASSERT_NIL_P(
        Binary::create({
            tagged("a", 1),
            tagged("b", 42, field<icd::UnsignedBinaryField>("kind", 8, 8)),
        })
    );
    const auto [message, err] = dec.decode(PAYLOAD);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "frame matches messages a and b");
}

/// @brief it should refuse two messages with the same CAN identifier.
TEST(Binary, RefuseSameCANIdentifier) {
    assert_refuses_all(
        {identified("a", can(0x100)), identified("b", can(0x100))},
        "messages a and b both match CAN identifier 0x100"
    );
}

/// @brief it should refuse a masked CAN identifier that meets another identifier.
TEST(Binary, RefuseOverlappingMasks) {
    assert_refuses_all(
        {
            identified("a", can(0x18FEF1AB, true)),
            identified("b", can(0x18FEF100, true, 0x00FFFF00)),
        },
        "messages a and b both match extended CAN identifier 0x18FEF1AB"
    );
}

/// @brief it should refuse a CAN identifier too large for its format.
TEST(Binary, RefuseCANIdentifierPastMax) {
    assert_refuses_all(
        {identified("a", can(0x800))},
        "message a: identifier: id 0x800 exceeds the maximum 0x7FF"
    );
}

/// @brief it should refuse two messages whose identifier fields hold the same value.
TEST(Binary, RefuseSameFieldValue) {
    assert_refuses_all(
        {tagged("a", 1), tagged("b", 1)},
        "messages a and b match the same identifier value 1"
    );
}

/// @brief it should refuse signed and unsigned identifier fields on the same bits whose
/// values have the same raw bits.
TEST(Binary, RefuseSameFieldBitsAcrossSigns) {
    assert_refuses_all(
        {
            tagged("a", -1, field<icd::IntegerBinaryField>("kind", 0, 8)),
            tagged("b", 255),
        },
        "messages a and b match the same identifier value 255"
    );
}

/// @brief it should refuse an identifier field that ends past the fixed payload length.
TEST(Binary, RefuseIdentifierPastLength) {
    auto m = tagged("a", 1, field<icd::UnsignedBinaryField>("kind", 16, 8));
    m.payload.length = 2;
    assert_refuses_all(
        {m},
        "message a: identifier: field kind: the field ends past the 2-byte payload"
    );
}

/// @brief it should refuse an identifier value that its field cannot hold.
TEST(Binary, RefuseFieldValueOutOfRange) {
    assert_refuses_all(
        {tagged("a", 300)},
        "message a: identifier: 300 is out of range for field kind"
    );
    assert_refuses_all(
        {tagged("a", -1)},
        "message a: identifier: -1 is out of range for field kind"
    );
}

/// @brief it should refuse an identifier field that is a float or does not exist.
TEST(Binary, RefuseInvalidIdentifierField) {
    assert_refuses_all(
        {tagged("a", 1, field<icd::FloatBinaryField>("kind", 0, 32))},
        "message a: identifier: field kind is a float field"
    );
    auto m = tagged("a", 1);
    const auto key = x::uuid::create();
    std::get<icd::FieldIdentifier>(m.payload.identifier.value()).field = key;
    assert_refuses_all(
        {m},
        "message a: identifier: field " + key.to_string() + " does not exist"
    );
}

namespace {
using Bytes = std::vector<std::uint8_t>;

/// @brief returns a vector that holds each series of s.
template<typename... S>
std::vector<x::telem::Series> series_of(S &&...s) {
    std::vector<x::telem::Series> out;
    out.reserve(sizeof...(s));
    (out.push_back(std::forward<S>(s)), ...);
    return out;
}

/// @brief returns the payload that an encoder of m encodes from samples.
Bytes encode(const BinaryMessage &m, const std::vector<x::telem::Series> &samples) {
    auto enc = ASSERT_NIL_P(BinaryEncoder::create(m));
    Bytes payload;
    EXPECT_NIL(enc.encode(samples, payload));
    return payload;
}

/// @brief asserts that encoding samples for m fails with text.
void assert_rejects_samples(
    const BinaryMessage &m,
    const std::vector<x::telem::Series> &samples,
    const std::string &text
) {
    auto enc = ASSERT_NIL_P(BinaryEncoder::create(m));
    Bytes payload;
    const auto err = enc.encode(samples, payload);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, text);
}

/// @brief asserts that creating an encoder for m fails with text.
void assert_encoder_refuses(const BinaryMessage &m, const std::string &text) {
    const auto err = BinaryEncoder::create(m).second;
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, text);
}

/// @brief returns the sample that a decoder reads back from the payload that an encoder
/// encodes from v, for one field f bound to a channel of dt.
template<typename T>
T round_trip(const icd::BinaryField &f, const x::telem::DataType &dt, const T v) {
    const auto m = frame(f, dt);
    const auto payload = encode(m, series_of(x::telem::Series(v, dt)));
    auto dec = ASSERT_NIL_P(Binary::create({m}));
    EXPECT_EQ(ASSERT_NIL_P(dec.decode(payload)), 0U);
    return dec.samples(0)[0].at<T>(0);
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

/// @brief it should write a sample into the bits of its field.
TEST(BinaryEncoder, EncodeUnsigned) {
    const auto m = frame(field<icd::UnsignedBinaryField>("v", 8, 8), x::telem::UINT8_T);
    EXPECT_EQ(encode(m, series_of(x::telem::Series(std::uint8_t{42}))), (Bytes{0, 42}));
}

/// @brief it should encode samples that a decoder of the same message reads back.
TEST(BinaryEncoder, RoundTrip) {
    auto big = field<icd::IntegerBinaryField>("v", 7, 16);
    big.byte_order = icd::BYTE_ORDER_BIG_ENDIAN;
    EXPECT_EQ(round_trip(big, x::telem::INT16_T, std::int16_t{-1234}), -1234);
    EXPECT_EQ(
        round_trip(
            field<icd::IntegerBinaryField>("v", 3, 5),
            x::telem::INT8_T,
            std::int8_t{-16}
        ),
        -16
    );
    EXPECT_EQ(
        round_trip(field<icd::FloatBinaryField>("v", 0, 32), x::telem::FLOAT32_T, 1.5F),
        1.5F
    );
    EXPECT_EQ(
        round_trip(
            field<icd::FloatBinaryField>("v", 0, 64),
            x::telem::FLOAT64_T,
            -2.25
        ),
        -2.25
    );
    constexpr auto max = std::numeric_limits<std::uint64_t>::max();
    EXPECT_EQ(
        round_trip(
            field<icd::UnsignedBinaryField>("v", 0, 64),
            x::telem::UINT64_T,
            max
        ),
        max
    );
    auto scaled = field<icd::IntegerBinaryField>("v", 0, 16);
    scaled.scale = 0.1;
    scaled.offset = -40;
    EXPECT_NEAR(round_trip(scaled, x::telem::FLOAT64_T, 21.5), 21.5, 1e-9);
}

/// @brief it should write the default of a field with no binding or no sample, and zero
/// the bits that no field holds.
TEST(BinaryEncoder, EncodeDefaults) {
    auto v = field<icd::UnsignedBinaryField>("v", 0, 8);
    v.default_ = 7;
    auto w = field<icd::UnsignedBinaryField>("w", 16, 8);
    w.default_ = 9;
    const BinaryMessage m{
        .name = "frame",
        .payload = {.fields = {v, w}},
        .bindings = {{.field = v.key, .data_type = x::telem::UINT8_T}},
    };
    EXPECT_EQ(encode(m, series_of(none())), (Bytes{7, 0, 9}));
    EXPECT_EQ(
        encode(m, series_of(x::telem::Series(std::uint8_t{3}))),
        (Bytes{3, 0, 9})
    );
}

/// @brief it should fill a payload to the length of its message.
TEST(BinaryEncoder, EncodeLength) {
    auto m = frame(field<icd::UnsignedBinaryField>("v", 0, 8), x::telem::UINT8_T);
    m.payload.length = 4;
    EXPECT_EQ(
        encode(m, series_of(x::telem::Series(std::uint8_t{5}))),
        (Bytes{5, 0, 0, 0})
    );
}

/// @brief it should remove the scale and offset from a sample, and round it to the
/// nearest integer.
TEST(BinaryEncoder, EncodeScaledAndRounded) {
    auto scaled = field<icd::UnsignedBinaryField>("v", 0, 8);
    scaled.scale = 0.5;
    scaled.offset = 10;
    EXPECT_EQ(
        encode(frame(scaled, x::telem::FLOAT64_T), series_of(x::telem::Series(20.0))),
        Bytes{20}
    );
    const auto u = frame(
        field<icd::UnsignedBinaryField>("v", 0, 8),
        x::telem::FLOAT64_T
    );
    EXPECT_EQ(encode(u, series_of(x::telem::Series(2.5))), Bytes{3});
    const auto i = frame(
        field<icd::IntegerBinaryField>("v", 0, 8),
        x::telem::FLOAT32_T
    );
    EXPECT_EQ(encode(i, series_of(x::telem::Series(-2.5F))), Bytes{0xFD});
}

/// @brief it should reject a field with no sample and no default.
TEST(BinaryEncoder, RejectMissingValue) {
    const auto m = frame(field<icd::UnsignedBinaryField>("v", 0, 8), x::telem::UINT8_T);
    assert_rejects_samples(m, series_of(none()), "field v: no value and no default");
}

/// @brief it should reject a sample that does not fit its field.
TEST(BinaryEncoder, RejectOutOfRange) {
    const auto u8 = field<icd::UnsignedBinaryField>("v", 0, 8);
    const auto i8 = field<icd::IntegerBinaryField>("v", 0, 8);
    assert_rejects_samples(
        frame(u8, x::telem::INT16_T),
        series_of(x::telem::Series(std::int16_t{256})),
        "field v: 256 is out of range for 8 bits"
    );
    assert_rejects_samples(
        frame(u8, x::telem::INT16_T),
        series_of(x::telem::Series(std::int16_t{-1})),
        "field v: -1 is out of range for 8 bits"
    );
    assert_rejects_samples(
        frame(i8, x::telem::UINT8_T),
        series_of(x::telem::Series(std::uint8_t{128})),
        "field v: 128 is out of range for 8 bits"
    );
    assert_rejects_samples(
        frame(i8, x::telem::INT16_T),
        series_of(x::telem::Series(std::int16_t{-129})),
        "field v: -129 is out of range for 8 bits"
    );
    assert_rejects_samples(
        frame(u8, x::telem::FLOAT64_T),
        series_of(x::telem::Series(255.5)),
        "field v: 255.5 is out of range for 8 bits"
    );
    assert_rejects_samples(
        frame(u8, x::telem::FLOAT64_T),
        series_of(x::telem::Series(std::numeric_limits<double>::quiet_NaN())),
        "field v: NaN is out of range for 8 bits"
    );
    assert_rejects_samples(
        frame(field<icd::IntegerBinaryField>("v", 0, 64), x::telem::UINT64_T),
        series_of(x::telem::Series(std::uint64_t{1} << 63)),
        "field v: 9223372036854775808 is out of range for 64 bits"
    );
    assert_rejects_samples(
        frame(field<icd::FloatBinaryField>("v", 0, 32), x::telem::FLOAT64_T),
        series_of(x::telem::Series(1e300)),
        "field v: 1e+300 overflows float32"
    );
}

/// @brief it should write the value of the identifier into the identifier field.
TEST(BinaryEncoder, EncodeIdentifier) {
    EXPECT_EQ(
        encode(tagged("status", 5), series_of(x::telem::Series(std::uint8_t{42}))),
        (Bytes{5, 42})
    );
    const auto negative = tagged(
        "status",
        -2,
        field<icd::IntegerBinaryField>("kind", 0, 8)
    );
    EXPECT_EQ(
        encode(negative, series_of(x::telem::Series(std::uint8_t{42}))),
        (Bytes{0xFE, 42})
    );
}

/// @brief it should refuse a binding on the identifier field.
TEST(BinaryEncoder, RefuseBoundIdentifier) {
    auto m = tagged("status", 5);
    m.bindings.push_back(
        {.field = icd::base(m.payload.fields[1]).key, .data_type = x::telem::UINT8_T}
    );
    assert_encoder_refuses(
        m,
        "message status: field kind holds the identifier, so it cannot take a channel"
    );
}

/// @brief it should refuse a field with no binding and no default.
TEST(BinaryEncoder, RefuseUnboundFieldWithoutDefault) {
    auto m = frame(field<icd::UnsignedBinaryField>("v", 0, 8), x::telem::UINT8_T);
    m.payload.fields.emplace_back(field<icd::UnsignedBinaryField>("w", 8, 8));
    assert_encoder_refuses(m, "message frame: field w: needs a channel or a default");
}

/// @brief it should refuse a binding to a field that does not exist, and two bindings
/// of one field.
TEST(BinaryEncoder, RefuseInvalidBindings) {
    const auto v = field<icd::UnsignedBinaryField>("v", 0, 8);
    auto unknown = frame(v, x::telem::UINT8_T);
    const auto key = x::uuid::create();
    unknown.bindings[0].field = key;
    assert_encoder_refuses(
        unknown,
        "message frame: field " + key.to_string() + " does not exist"
    );
    auto twice = frame(v, x::telem::UINT8_T);
    twice.bindings.push_back(twice.bindings[0]);
    assert_encoder_refuses(twice, "message frame: field v has more than one binding");
}

/// @brief it should refuse a channel that does not hold numbers, and a scaled field
/// whose channel does not hold floats.
TEST(BinaryEncoder, RefuseChannelDataType) {
    assert_encoder_refuses(
        frame(field<icd::UnsignedBinaryField>("v", 0, 8), x::telem::STRING_T),
        "message frame: field v: cannot write string channels to unsigned fields"
    );
    auto scaled = field<icd::IntegerBinaryField>("v", 0, 16);
    scaled.scale = 2;
    assert_encoder_refuses(
        frame(scaled, x::telem::INT32_T),
        "message frame: field v: a scaled field requires a float channel, not int32"
    );
}

/// @brief it should refuse a bound field with a zero scale, and write the default of an
/// unbound one.
TEST(BinaryEncoder, RefuseZeroScale) {
    auto zero = field<icd::IntegerBinaryField>("v", 0, 16);
    zero.scale = 0;
    assert_encoder_refuses(
        frame(zero, x::telem::FLOAT64_T),
        "message frame: field v: scale must be non-zero"
    );
    auto unbound = field<icd::UnsignedBinaryField>("w", 8, 8);
    unbound.scale = 0;
    unbound.default_ = 7;
    auto m = frame(field<icd::UnsignedBinaryField>("v", 0, 8), x::telem::UINT8_T);
    m.payload.fields.emplace_back(unbound);
    EXPECT_EQ(encode(m, series_of(x::telem::Series(std::uint8_t{3}))), (Bytes{3, 7}));
}

/// @brief it should write only the fields that the multiplexor values select.
TEST(BinaryEncoder, EncodeMultiplexed) {
    const auto mode = field<icd::UnsignedBinaryField>("mode", 0, 8);
    const auto a = selected_by("a", 1, mode.key, {1});
    const auto b = selected_by("b", 2, mode.key, {2, 3});
    const auto m = multiplexed({mode, a, b}, {mode.key, a.key, b.key});
    auto enc = ASSERT_NIL_P(BinaryEncoder::create(m));
    const auto samples = [](const std::uint8_t v) {
        return series_of(
            x::telem::Series(v),
            x::telem::Series(std::uint8_t{16}),
            x::telem::Series(std::uint8_t{32})
        );
    };
    Bytes payload;
    ASSERT_NIL(enc.encode(samples(3), payload));
    EXPECT_EQ(payload, (Bytes{3, 0, 32}));
    ASSERT_NIL(enc.encode(samples(1), payload));
    EXPECT_EQ(payload, (Bytes{1, 16, 0}));
    auto dec = ASSERT_NIL_P(Binary::create({m}));
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(payload)), 0U);
    EXPECT_EQ(dec.samples(0)[1].at<std::uint8_t>(0), 16);
    EXPECT_EQ(dec.samples(0)[2].size(), 0);
}

/// @brief it should need no value for a field that the multiplexors do not select.
TEST(BinaryEncoder, EncodeUnselectedFieldWithoutValue) {
    const auto mode = field<icd::UnsignedBinaryField>("mode", 0, 8);
    const auto a = selected_by("a", 1, mode.key, {1});
    const auto m = multiplexed({mode, a}, {mode.key, a.key});
    EXPECT_EQ(
        encode(m, series_of(x::telem::Series(std::uint8_t{2}), none())),
        (Bytes{2, 0})
    );
}

/// @brief it should select a field only when each multiplexor in its chain holds one of
/// its values.
TEST(BinaryEncoder, EncodeCascadedMultiplexors) {
    const auto outer = field<icd::UnsignedBinaryField>("outer", 0, 8);
    const auto inner = selected_by("inner", 1, outer.key, {1});
    const auto v = selected_by("v", 2, inner.key, {2});
    const auto m = multiplexed({outer, inner, v}, {outer.key, inner.key, v.key});
    const auto samples = [](const std::uint8_t o, const std::uint8_t i) {
        return series_of(
            x::telem::Series(o),
            x::telem::Series(i),
            x::telem::Series(std::uint8_t{9})
        );
    };
    EXPECT_EQ(encode(m, samples(1, 2)), (Bytes{1, 2, 9}));
    EXPECT_EQ(encode(m, samples(1, 3)), (Bytes{1, 3, 0}));
    EXPECT_EQ(encode(m, samples(0, 2)), (Bytes{0, 0, 0}));
}

/// @brief it should need nothing for a field that a fixed multiplexor value never
/// selects, and leave it out.
TEST(BinaryEncoder, EncodeFixedMultiplexor) {
    auto mode = field<icd::UnsignedBinaryField>("mode", 0, 8);
    mode.default_ = 1;
    const auto a = selected_by("a", 1, mode.key, {1});
    const auto b = selected_by("b", 2, mode.key, {2});
    const auto m = multiplexed({mode, a, b}, {a.key});
    EXPECT_EQ(
        encode(m, series_of(x::telem::Series(std::uint8_t{16}))),
        (Bytes{1, 16, 0})
    );
}

/// @brief it should refuse a binding on a field that a fixed multiplexor value never
/// selects.
TEST(BinaryEncoder, RefuseFieldThatFixedMultiplexorNeverSelects) {
    auto mode = field<icd::UnsignedBinaryField>("mode", 0, 8);
    mode.default_ = 1;
    const auto b = selected_by("b", 1, mode.key, {2});
    assert_encoder_refuses(
        multiplexed({mode, b}, {b.key}),
        "message frame: field b takes a channel, but multiplexor mode never selects it"
    );
}

/// @brief it should refuse a field with no binding and no default that a multiplexor
/// bound to a channel can select.
TEST(BinaryEncoder, RefuseUnboundFieldUnderBoundMultiplexor) {
    const auto mode = field<icd::UnsignedBinaryField>("mode", 0, 8);
    const auto b = selected_by("b", 1, mode.key, {2});
    assert_encoder_refuses(
        multiplexed({mode, b}, {mode.key}),
        "message frame: field b: needs a channel or a default"
    );
}

/// @brief it should encode without allocating once its payload has room.
TEST(BinaryEncoder, EncodeWithoutAllocating) {
    auto f = field<icd::IntegerBinaryField>("temp", 0, 16);
    f.scale = 0.1;
    auto enc = ASSERT_NIL_P(BinaryEncoder::create(frame(f, x::telem::FLOAT64_T)));
    const auto samples = series_of(x::telem::Series(0.1 + 0.2));
    Bytes payload;
    ASSERT_NIL(enc.encode(samples, payload));
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++)
        ASSERT_NIL(enc.encode(samples, payload));
    EXPECT_EQ(x::bench::alloc_count.load(std::memory_order_relaxed) - before, 0);
    EXPECT_EQ(payload, (Bytes{0x03, 0x00}));
}

/// @brief it should check a sample of a binding as encode does, and report which fields
/// have a default.
TEST(BinaryEncoder, CheckSample) {
    const auto v = field<icd::UnsignedBinaryField>("v", 0, 8);
    auto w = field<icd::UnsignedBinaryField>("w", 8, 4);
    w.default_ = 1;
    const BinaryMessage m{
        .name = "frame",
        .payload = {.fields = {v, w}},
        .bindings = {
            {.field = w.key, .data_type = x::telem::UINT8_T},
            {.field = v.key, .data_type = x::telem::UINT8_T},
        },
    };
    const auto enc = ASSERT_NIL_P(BinaryEncoder::create(m));
    ASSERT_NIL(enc.check(0, x::telem::SampleValue(std::uint8_t{15})));
    ASSERT_NIL(enc.check(1, x::telem::SampleValue(std::uint8_t{16})));
    const auto err = enc.check(0, x::telem::SampleValue(std::uint8_t{16}));
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "field w: 16 is out of range for 4 bits");
    EXPECT_TRUE(enc.has_default(0));
    EXPECT_FALSE(enc.has_default(1));
}
}
