// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/payload/json.gen.h"
#include "x/cpp/payload/proto.gen.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"

namespace x::payload {
namespace {
/// @brief expects v to come back from protobuf unchanged.
template<typename T>
T expect_round_trips(const T &v) {
    const auto pb = ASSERT_NIL_P(v.to_proto());
    const auto parsed = ASSERT_NIL_P(T::from_proto(pb));
    EXPECT_EQ(parsed.to_json(), v.to_json());
    return parsed;
}
}

/// @brief it should round trip a float read with a data type, enums, and a fill
/// through protobuf.
TEST(FloatRead, RoundTripsProto) {
    FloatRead r;
    r.data_type = x::telem::FLOAT32_T;
    r.enum_values = {{"off", Float{0}}, {"on", Float{1.5}}};
    r.fill = Float{-2.5};
    const auto parsed = expect_round_trips(r);
    EXPECT_EQ(parsed.data_type, x::telem::FLOAT32_T);
    ASSERT_EQ(parsed.enum_values.size(), 2);
    EXPECT_EQ(parsed.enum_values[1].label, "on");
}

/// @brief it should leave the data type and fill of a float read absent through
/// protobuf.
TEST(FloatRead, RoundTripsEmptyProto) {
    const auto parsed = expect_round_trips(FloatRead{});
    EXPECT_FALSE(parsed.data_type.has_value());
    EXPECT_FALSE(parsed.fill.has_value());
    EXPECT_TRUE(parsed.enum_values.empty());
}

/// @brief it should round trip an integer read through protobuf.
TEST(IntegerRead, RoundTripsProto) {
    IntegerRead r;
    r.data_type = x::telem::INT16_T;
    r.enum_values = {{"low", Integer{-1}}, {"high", Integer{1}}};
    r.fill = Integer{-7};
    const auto parsed = expect_round_trips(r);
    EXPECT_EQ(parsed.data_type, x::telem::INT16_T);
    EXPECT_EQ(parsed.fill->value, -7);
}

/// @brief it should round trip an unsigned integer read through protobuf.
TEST(UnsignedRead, RoundTripsProto) {
    UnsignedRead r;
    r.data_type = x::telem::UINT8_T;
    r.enum_values = {{"idle", Unsigned{0}}, {"run", Unsigned{2}}};
    r.fill = Unsigned{9};
    const auto parsed = expect_round_trips(r);
    EXPECT_EQ(parsed.data_type, x::telem::UINT8_T);
    EXPECT_EQ(parsed.fill->value, 9);
}

/// @brief it should round trip a string read through protobuf.
TEST(StringRead, RoundTripsProto) {
    StringRead r;
    r.enum_values = {{"running", "RUN"}};
    r.fill = "idle";
    const auto parsed = expect_round_trips(r);
    EXPECT_EQ(parsed.fill, "idle");
}

/// @brief it should round trip a boolean read through protobuf.
TEST(BooleanRead, RoundTripsProto) {
    BooleanRead r;
    r.enum_values = {{"open", true}};
    r.fill = false;
    const auto parsed = expect_round_trips(r);
    EXPECT_EQ(parsed.fill, false);
}

/// @brief it should round trip a timestamp read through protobuf.
TEST(TimestampRead, RoundTripsProto) {
    TimestampRead r;
    r.time_format = TIME_FORMAT_UNIX_MS;
    r.fill = x::telem::TimeStamp(1700000000000000000);
    const auto parsed = expect_round_trips(r);
    EXPECT_EQ(parsed.time_format, TIME_FORMAT_UNIX_MS);
    EXPECT_EQ(parsed.fill, x::telem::TimeStamp(1700000000000000000));
}

/// @brief it should reject a timestamp read with an unknown time format.
TEST(TimestampRead, RejectsUnknownTimeFormatToProto) {
    TimestampRead r;
    r.time_format = "fortnight";
    const auto [_, err] = r.to_proto();
    ASSERT_TRUE(err);
    EXPECT_EQ(err.type, "unrecognized TimeFormat value: fortnight");
}

/// @brief it should reject a protobuf time format with no value.
TEST(TimeFormat, RejectsOutOfRangeFromProto) {
    const auto [_, err] = time_format_from_pb(static_cast<pb::TimeFormat>(99));
    ASSERT_TRUE(err);
    EXPECT_EQ(err.type, "unrecognized TimeFormat protobuf value");
}

/// @brief it should reject an unknown kind.
TEST(Kind, RejectsUnknownToProto) {
    const auto [_, err] = kind_to_pb("complex");
    ASSERT_TRUE(err);
    EXPECT_EQ(err.type, "unrecognized Kind value: complex");
}

/// @brief it should round trip an enum entry through protobuf.
TEST(EnumEntry, RoundTripsProto) {
    const auto parsed = expect_round_trips(EnumEntry{"on", 1.5});
    EXPECT_EQ(parsed.label, "on");
    EXPECT_EQ(parsed.value, 1.5);
}
}
