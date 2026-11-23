// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <string>
#include <string_view>
#include <type_traits>
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

#include "driver/codec/errors.h"
#include "driver/codec/json.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;

/// @brief returns a field of type F named name at /name.
template<typename F>
F field(const std::string &name) {
    F f;
    f.key = x::uuid::create();
    f.name = name;
    f.pointer = "/" + name;
    return f;
}

/// @brief returns a message named reading with one field v of type F at /v, bound to a
/// channel of dt.
template<typename F>
JSONMessage reading(const x::telem::DataType &dt) {
    auto f = field<F>("v");
    if constexpr (std::is_same_v<F, icd::TimestampJSONField>) f.time_format = "iso8601";
    return {
        .name = "reading",
        .payload = {.fields = {f}},
        .bindings = {{.field = f.key, .data_type = dt}},
    };
}

/// @brief returns the error that creating a decoder for one field of type F bound to a
/// channel of dt gives.
template<typename F>
x::errors::Error create_error(const x::telem::DataType &dt) {
    return JSON::create({reading<F>(dt)}).second;
}

/// @brief returns two messages that first and second match: other, with one field, and
/// status, with a scaled float, an integer, and a string.
std::vector<JSONMessage>
routed(const icd::JSONMatch &first, const icd::JSONMatch &second) {
    const auto other = field<icd::FloatJSONField>("other");
    auto temp = field<icd::FloatJSONField>("temp");
    temp.scale = 0.5;
    temp.offset = -10;
    const auto count = field<icd::IntegerJSONField>("count");
    const auto state = field<icd::StringJSONField>("state");
    return {
        {
            .name = "other",
            .payload = {.match = first, .fields = {other}},
            .bindings = {{.field = other.key, .data_type = x::telem::FLOAT64_T}},
        },
        {
            .name = "status",
            .payload = {.match = second, .fields = {temp, count, state}},
            .bindings = {
                {.field = temp.key, .data_type = x::telem::FLOAT32_T},
                {.field = count.key, .data_type = x::telem::FLOAT64_T},
                {.field = state.key, .data_type = x::telem::STRING_T},
            },
        },
    };
}

/// @brief returns a match rule of type M at pointer with value.
template<typename M, typename V>
icd::JSONMatch match(const std::string &pointer, const V &value) {
    M m;
    m.pointer = pointer;
    m.value = value;
    return m;
}

/// @brief returns a present rule at pointer.
icd::JSONMatch present(const std::string &pointer) {
    icd::PresentJSONMatch m;
    m.pointer = pointer;
    return m;
}

/// @brief asserts that a decoder for routed(first, second) decodes doc as status
/// without allocating, once a first decode is done.
void assert_decodes_without_allocating(
    const icd::JSONMatch &first,
    const icd::JSONMatch &second,
    const std::string_view doc
) {
    auto dec = ASSERT_NIL_P(JSON::create(routed(first, second)));
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(doc)), std::size_t{1});
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++)
        ASSERT_EQ(ASSERT_NIL_P(dec.decode(doc)), std::size_t{1});
    EXPECT_EQ(x::bench::alloc_count.load(std::memory_order_relaxed) - before, 0);
}
}

/// @brief it should write each field kind to the data types it converts to.
TEST(JSON, ConvertsKindsToDataTypes) {
    ASSERT_NIL(create_error<icd::IntegerJSONField>(x::telem::FLOAT32_T));
    ASSERT_NIL(create_error<icd::BooleanJSONField>(x::telem::FLOAT64_T));
}

/// @brief it should reject a field kind that does not convert to its data type.
TEST(JSON, RejectsKindsThatDoNotConvert) {
    const auto check = [](const x::errors::Error &err, const std::string &data) {
        ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
        ASSERT_EQ(err.data, "message reading: field v: " + data);
    };
    check(
        create_error<icd::IntegerJSONField>(x::telem::UINT32_T),
        "cannot write integer values to uint32 channels"
    );
    check(
        create_error<icd::TimestampJSONField>(x::telem::INT64_T),
        "cannot write timestamp values to int64 channels"
    );
    check(
        create_error<icd::FloatJSONField>(x::telem::DataType("complex128")),
        "cannot write float values to complex128 channels"
    );
}

/// @brief it should replace the samples of a message on each decode.
TEST(JSON, ReplacesSamplesOnEachDecode) {
    auto dec = ASSERT_NIL_P(
        JSON::create({reading<icd::FloatJSONField>(x::telem::FLOAT64_T)})
    );
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(R"({"v": 1})")), std::size_t{0});
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(R"({"v": 2})")), std::size_t{0});
    ASSERT_EQ(dec.samples(0)[0].values<double>(), std::vector<double>{2});
}

/// @brief it should reject a document that is not JSON.
TEST(JSON, RejectsDocumentThatIsNotJSON) {
    auto dec = ASSERT_NIL_P(
        JSON::create({reading<icd::FloatJSONField>(x::telem::FLOAT64_T)})
    );
    const auto [message, err] = dec.decode("not json");
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid character 'o' after offset 1");
}

/// @brief it should give a timestamp field with a now default the time of each decode.
TEST(JSON, DecodesNowDefault) {
    auto m = reading<icd::TimestampJSONField>(x::telem::TIMESTAMP_T);
    std::get<icd::TimestampJSONField>(m.payload.fields[0])
        .default_ = icd::NowTimestampDefault{};
    auto dec = ASSERT_NIL_P(JSON::create({m}));
    const auto before = x::telem::TimeStamp::now();
    ASSERT_EQ(ASSERT_NIL_P(dec.decode("{}")), std::size_t{0});
    const auto stamp = dec.samples(0)[0].at<x::telem::TimeStamp>(0);
    ASSERT_GE(stamp, before);
    ASSERT_LE(stamp, x::telem::TimeStamp::now());
}

/// @brief it should give a UUID field with a random default a new UUID on each decode.
TEST(JSON, DecodesRandomDefault) {
    auto m = reading<icd::UuidJSONField>(x::telem::UUID_T);
    std::get<icd::UuidJSONField>(m.payload.fields[0])
        .default_ = icd::RandomUUIDDefault{};
    auto dec = ASSERT_NIL_P(JSON::create({m}));
    ASSERT_EQ(ASSERT_NIL_P(dec.decode("{}")), std::size_t{0});
    const auto first = dec.samples(0)[0].uuids()[0];
    ASSERT_EQ(ASSERT_NIL_P(dec.decode("{}")), std::size_t{0});
    ASSERT_NE(first, x::uuid::NIL);
    ASSERT_NE(dec.samples(0)[0].uuids()[0], first);
}

/// @brief it should reject a JSON field whose default is not JSON text.
TEST(JSON, RejectsJSONDefaultThatIsNotJSON) {
    auto m = reading<icd::JSONJSONField>(x::telem::JSON_T);
    std::get<icd::JSONJSONField>(m.payload.fields[0]).default_ = "{";
    const auto err = JSON::create({m}).second;
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    ASSERT_EQ(err.data, "message reading: field v: default: not valid JSON");
}

/// @brief it should decode a document without allocating when a string match rule
/// finds its message.
TEST(JSON, DecodesWithoutAllocatingByString) {
    assert_decodes_without_allocating(
        match<icd::StringJSONMatch>("/type", std::string("event")),
        match<icd::StringJSONMatch>("/type", std::string("status")),
        R"({"type":"status","temp":40,"count":3,"state":"run"})"
    );
}

/// @brief it should decode a document without allocating when a number match rule
/// finds its message.
TEST(JSON, DecodesWithoutAllocatingByNumber) {
    assert_decodes_without_allocating(
        match<icd::FloatJSONMatch>("/id", 2.5),
        match<icd::IntegerJSONMatch>("/id", 7),
        R"({"id":7,"temp":40,"count":3,"state":"run"})"
    );
}

/// @brief it should decode a document without allocating when a present rule finds its
/// message.
TEST(JSON, DecodesWithoutAllocatingByPresence) {
    assert_decodes_without_allocating(
        present("/other"),
        present("/temp"),
        R"({"other":null,"temp":40,"count":3,"state":"run"})"
    );
}
}
