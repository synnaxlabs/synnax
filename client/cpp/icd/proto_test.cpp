// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/icd/json.gen.h"
#include "client/cpp/icd/proto.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

namespace synnax::icd {
namespace {
ICD parse_icd(const x::json::json &j) {
    x::json::Parser parser(j);
    auto ic = ICD::parse(parser);
    EXPECT_NIL(parser.error());
    return ic;
}

const x::json::json BINARY_MESSAGE = {
    {"key", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e01"},
    {"name", "Engine"},
    {"payload",
     {{"format", "binary"},
      {"length", 8},
      {"identifier",
       {{"type", "can"},
        {"id", 256},
        {"extended", true},
        {"fd", true},
        {"bitrate_switched", true},
        {"mask", 2047}}},
      {"fields",
       {{{"key", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e02"},
         {"name", "mode"},
         {"scale", 1},
         {"offset", 0},
         {"units", ""},
         {"start_bit", 0},
         {"bit_length", 4},
         {"byte_order", "little_endian"},
         {"kind", "unsigned"},
         {"multiplex_values", x::json::json::array()}},
        {{"key", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e03"},
         {"name", "rpm"},
         {"scale", 0.25},
         {"offset", -40},
         {"units", "rpm"},
         {"min", 0},
         {"max", 8000},
         {"start_bit", 15},
         {"bit_length", 16},
         {"byte_order", "big_endian"},
         {"kind", "integer"},
         {"multiplexor", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e02"},
         {"multiplex_values", {1, 3}}}}}}},
};

const x::json::json TEXT_MESSAGE = {
    {"key", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e04"},
    {"name", "Line"},
    {"payload",
     {{"format", "text"},
      {"delimiter", ";"},
      {"prefix", "$GP"},
      {"fields",
       {{{"key", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e05"},
         {"name", "speed"},
         {"scale", 1},
         {"offset", 0},
         {"units", "m/s"},
         {"encoding", "delimited"},
         {"position", 2}},
        {{"key", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e06"},
         {"name", "heading"},
         {"scale", 1},
         {"offset", 0},
         {"units", ""},
         {"encoding", "tagged"},
         {"tag", "HDG="}}}}}},
};

x::json::json json_field(
    const std::string &key,
    const std::string &type,
    const x::json::json &extra = x::json::json::object()
) {
    x::json::json f = {
        {"key", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5f" + key},
        {"name", type},
        {"scale", 1},
        {"offset", 0},
        {"units", ""},
        {"pointer", "/" + type},
        {"type", type},
    };
    f.update(extra);
    return f;
}

const x::json::json JSON_MESSAGE = {
    {"key", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e07"},
    {"name", "Status"},
    {"payload",
     {{"format", "json"},
      {"match", {{"type", "string"}, {"pointer", "/kind"}, {"value", "status"}}},
      {"fields",
       {json_field(
            "01",
            "float",
            {{"enum_values", {{{"label", "HIGH"}, {"value", "Infinity"}}}},
             {"default", 1.5}}
        ),
        json_field(
            "02",
            "integer",
            {{"enum_values", {{{"label", "MIN"}, {"value", "-9223372036854775808"}}}},
             {"default", "-3"}}
        ),
        json_field(
            "03",
            "unsigned",
            {{"enum_values", {{{"label", "MAX"}, {"value", "18446744073709551615"}}}},
             {"default", "7"}}
        ),
        json_field(
            "04",
            "string",
            {{"enum_values", {{{"label", "on"}, {"value", "ON"}}}}, {"default", "OFF"}}
        ),
        json_field(
            "05",
            "boolean",
            {{"enum_values", {{{"label", "yes"}, {"value", true}}}}, {"default", false}}
        ),
        json_field(
            "06",
            "timestamp",
            {{"time_format", "unix_ms"}, {"default", 1700000000000000000}}
        ),
        json_field("07", "json"),
        json_field("08", "uuid"),
        json_field("09", "bytes")}}}},
};

const x::json::json FULL_ICD = {
    {"key", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e00"},
    {"name", "Rig"},
    {"messages", {BINARY_MESSAGE, TEXT_MESSAGE, JSON_MESSAGE}},
};

void expect_round_trips(const x::json::json &j) {
    const auto ic = parse_icd(j);
    const auto pb = ASSERT_NIL_P(ic.to_proto());
    const auto parsed = ASSERT_NIL_P(ICD::from_proto(pb));
    EXPECT_EQ(parsed.to_json(), ic.to_json());
}

/// @brief expects the value that j parses to to come back from protobuf as j.
template<typename T>
void expect_request_round_trips(const x::json::json &j) {
    x::json::Parser parser(j);
    const auto v = T::parse(parser);
    ASSERT_NIL(parser.error());
    const auto pb = ASSERT_NIL_P(v.to_proto());
    const auto parsed = ASSERT_NIL_P(T::from_proto(pb));
    EXPECT_EQ(parsed.to_json(), j);
}

const std::string KEY_A = "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5ea1";
const std::string KEY_B = "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5ea2";

x::json::json with_match(const x::json::json &match) {
    auto j = FULL_ICD;
    j["messages"] = {JSON_MESSAGE};
    j["messages"][0]["payload"]["match"] = match;
    return j;
}
}

/// @brief it should round trip an ICD with every payload and field variant through
/// protobuf.
TEST(ICD, RoundTripsProto) {
    expect_round_trips(FULL_ICD);
}

/// @brief it should round trip a float match through protobuf.
TEST(JSONMatch, RoundTripsFloatProto) {
    expect_round_trips(
        with_match({{"type", "float"}, {"pointer", "/a"}, {"value", "NaN"}})
    );
}

/// @brief it should round trip an integer match through protobuf.
TEST(JSONMatch, RoundTripsIntegerProto) {
    expect_round_trips(
        with_match({{"type", "integer"}, {"pointer", "/a"}, {"value", "-1"}})
    );
}

/// @brief it should round trip an unsigned integer match through protobuf.
TEST(JSONMatch, RoundTripsUnsignedProto) {
    expect_round_trips(
        with_match({{"type", "unsigned"}, {"pointer", "/a"}, {"value", "1"}})
    );
}

/// @brief it should round trip a boolean match through protobuf.
TEST(JSONMatch, RoundTripsBooleanProto) {
    expect_round_trips(
        with_match({{"type", "boolean"}, {"pointer", "/a"}, {"value", true}})
    );
}

/// @brief it should round trip a field identifier through protobuf.
TEST(Identifier, RoundTripsFieldVariantProto) {
    auto j = FULL_ICD;
    j["messages"] = {BINARY_MESSAGE};
    j["messages"][0]["payload"]["identifier"] = {
        {"type", "field"},
        {"field", "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e02"},
        {"value", -2},
    };
    expect_round_trips(j);
}

/// @brief it should round trip a filter with no clauses through protobuf.
TEST(Where, RoundTripsEmptyProto) {
    expect_request_round_trips<Where>(x::json::json::object());
}

/// @brief it should round trip a filter of one key and of several keys through
/// protobuf.
TEST(Where, RoundTripsKeysProto) {
    expect_request_round_trips<Where>({{"key", {{"eq", KEY_A}}}});
    expect_request_round_trips<Where>({{"key", {{"in", {KEY_A, KEY_B}}}}});
}

/// @brief it should keep an empty key set apart from an absent one through protobuf.
TEST(Where, RoundTripsEmptyKeySetProto) {
    expect_request_round_trips<Where>({{"key", {{"in", x::json::json::array()}}}});
    expect_request_round_trips<Where>({{"key", x::json::json::object()}});
}

/// @brief it should round trip nested and empty clause lists through protobuf.
TEST(Where, RoundTripsClausesProto) {
    expect_request_round_trips<Where>({
        {"and", {{{"key", {{"eq", KEY_A}}}}, {{"not", {{"key", {{"in", {KEY_B}}}}}}}}},
        {"or", {{{"key", {{"eq", KEY_B}}}}}},
    });
    expect_request_round_trips<Where>(
        {{"and", x::json::json::array()}, {"or", x::json::json::array()}}
    );
}

/// @brief it should round trip a retrieve request through protobuf.
TEST(RetrieveRequest, RoundTripsProto) {
    expect_request_round_trips<RetrieveRequest>({
        {"where", {{"key", {{"in", {KEY_A}}}}}},
        {"search_term", "rig"},
        {"after", {{"key", KEY_B}}},
        {"limit", 10},
    });
    expect_request_round_trips<RetrieveRequest>({{"search_term", ""}});
}

/// @brief it should round trip a retrieve response through protobuf.
TEST(RetrieveResponse, RoundTripsProto) {
    expect_request_round_trips<RetrieveResponse>({{"icds", {FULL_ICD}}});
}

/// @brief it should reject a cursor key that is not a UUID.
TEST(Cursor, RejectsInvalidKeyFromProto) {
    ::api::icd::pb::Cursor pb;
    pb.set_key("rig");
    const auto [_, err] = Cursor::from_proto(pb);
    ASSERT_OCCURRED_AS(err, x::uuid::INVALID);
    ASSERT_EQ(err.data, "Invalid UUID string at position 0: hex digit expected");
}

/// @brief it should reject a message with no key.
TEST(Message, RejectsEmptyKeyFromProto) {
    const ::service::icd::pb::Message pb;
    const auto [_, err] = Message::from_proto(pb);
    ASSERT_OCCURRED_AS(err, x::uuid::INVALID);
    ASSERT_EQ(err.data, "empty string");
}
}
