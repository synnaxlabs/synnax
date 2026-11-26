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
#include <string>
#include <string_view>
#include <type_traits>
#include <utility>
#include <variant>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/bench/alloc.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/series.h"
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

/// @brief returns a message named reading with integer fields v and w at /v and /w,
/// each bound to an int64 channel.
JSONMessage two_integers() {
    const auto v = field<icd::IntegerJSONField>("v");
    const auto w = field<icd::IntegerJSONField>("w");
    return {
        .name = "reading",
        .payload = {.fields = {v, w}},
        .bindings = {
            {.field = v.key, .data_type = x::telem::INT64_T},
            {.field = w.key, .data_type = x::telem::INT64_T},
        },
    };
}

/// @brief returns two messages that first and second match: other, with one field, and
/// status, with a scaled float, an integer, and a string.
std::vector<JSONMessage>
routed(const icd::JSONMatch &first, const icd::JSONMatch &second) {
    const auto other = field<icd::FloatJSONField>("other");
    auto temp = field<icd::FloatJSONField>("temp");
    temp.scale = 0.5;
    temp.offset = -10;
    temp.min = 0;
    temp.max = 100;
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

/// @brief returns a message named command with fields, where binding i binds field i to
/// a channel of data_types[i].
JSONMessage command(
    const std::vector<icd::JSONField> &fields,
    const std::vector<x::telem::DataType> &data_types = {}
) {
    JSONMessage m{.name = "command", .payload = {.fields = fields}};
    for (std::size_t i = 0; i < data_types.size(); i++)
        m.bindings.push_back({
            .field = icd::base(fields[i]).key,
            .data_type = data_types[i],
        });
    return m;
}

/// @brief returns a vector that holds each series of s, in order.
template<typename... S>
std::vector<x::telem::Series> series_of(S &&...s) {
    std::vector<x::telem::Series> out;
    (out.push_back(std::forward<S>(s)), ...);
    return out;
}

/// @brief returns the document that enc encodes from samples, parsed.
nlohmann::json encoded(JSONEncoder &enc, const std::vector<x::telem::Series> &samples) {
    std::vector<std::uint8_t> payload;
    EXPECT_NIL(enc.encode(samples, payload));
    return nlohmann::json::parse(payload);
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

/// @brief it should reject a JSON field whose default repeats a member name.
TEST(JSON, RejectsJSONDefaultWithRepeatedName) {
    auto m = reading<icd::JSONJSONField>(x::telem::JSON_T);
    std::get<icd::JSONJSONField>(m.payload.fields[0]).default_ = R"({"a":1,"a":2})";
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

/// @brief it should write the time of each encode for a now default.
TEST(JSONEncoder, WritesNowDefault) {
    auto v = field<icd::TimestampJSONField>("v");
    v.time_format = "unix_ns";
    v.default_ = icd::NowTimestampDefault{};
    auto enc = ASSERT_NIL_P(JSONEncoder::create(command({v})));
    const auto before = x::telem::TimeStamp::now().nanoseconds();
    const auto first = encoded(enc, {})["v"].get<std::int64_t>();
    const auto second = encoded(enc, {})["v"].get<std::int64_t>();
    ASSERT_GE(first, before);
    ASSERT_GE(second, first);
    ASSERT_LE(second, x::telem::TimeStamp::now().nanoseconds());
}

/// @brief it should write a new UUID at each encode for a random default, bound or not.
TEST(JSONEncoder, WritesRandomDefault) {
    auto unbound = field<icd::UuidJSONField>("unbound");
    unbound.default_ = icd::RandomUUIDDefault{};
    auto bound = field<icd::UuidJSONField>("bound");
    bound.default_ = icd::RandomUUIDDefault{};
    auto enc = ASSERT_NIL_P(
        JSONEncoder::create(command({bound, unbound}, {x::telem::UUID_T}))
    );
    const auto samples = series_of(x::telem::Series(x::telem::UUID_T, 0));
    const auto first = encoded(enc, samples);
    const auto second = encoded(enc, samples);
    for (const auto *name: {"bound", "unbound"}) {
        const auto id = ASSERT_NIL_P(
            x::uuid::UUID::parse(first[name].get<std::string>())
        );
        ASSERT_NE(id, x::uuid::NIL);
        ASSERT_NE(second[name], first[name]);
    }
}

/// @brief it should be ready when each field has a sample or a default.
TEST(JSONEncoder, Ready) {
    auto mode = field<icd::StringJSONField>("mode");
    mode.default_ = "auto";
    const auto temp = field<icd::FloatJSONField>("temp");
    const auto enc = ASSERT_NIL_P(
        JSONEncoder::create(
            command({mode, temp}, {x::telem::STRING_T, x::telem::FLOAT64_T})
        )
    );
    EXPECT_FALSE(enc.ready(series_of(
        x::telem::Series(x::telem::STRING_T, 0),
        x::telem::Series(x::telem::FLOAT64_T, 0)
    )));
    EXPECT_TRUE(enc.ready(
        series_of(x::telem::Series(x::telem::STRING_T, 0), x::telem::Series(1.5))
    ));
}

/// @brief it should fail to encode an empty series whose field has no default.
TEST(JSONEncoder, RejectsMissingSampleWithNoDefault) {
    auto enc = ASSERT_NIL_P(
        JSONEncoder::create(
            command({field<icd::FloatJSONField>("temp")}, {x::telem::FLOAT64_T})
        )
    );
    std::vector<std::uint8_t> payload;
    const auto err = enc.encode(
        series_of(x::telem::Series(x::telem::FLOAT64_T, 0)),
        payload
    );
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "message command: field temp: no sample and no default");
}

/// @brief it should write a generated default beside the value of a match rule, or as
/// the value of a present rule.
TEST(JSONEncoder, WritesGeneratedDefaultBesideMatch) {
    auto id = field<icd::UuidJSONField>("id");
    id.default_ = icd::RandomUUIDDefault{};
    for (const auto &[rule, type]:
         std::vector<std::pair<icd::JSONMatch, nlohmann::json>>{
             {match<icd::StringJSONMatch>("/type", std::string("status")), "status"},
             {present("/id"), nullptr},
         }) {
        auto m = command({id});
        m.payload.match = rule;
        auto enc = ASSERT_NIL_P(JSONEncoder::create(m));
        const auto doc = encoded(enc, {});
        ASSERT_EQ(doc.value("type", nlohmann::json()), type);
        ASSERT_NIL_P(x::uuid::UUID::parse(doc["id"].get<std::string>()));
    }
}

/// @brief it should reject a float default that is not finite.
TEST(JSONEncoder, RejectsDefaultThatIsNotFinite) {
    auto v = field<icd::FloatJSONField>("v");
    v.default_ = std::numeric_limits<double>::infinity();
    const auto err = JSONEncoder::create(command({v})).second;
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    ASSERT_EQ(err.data, "message command: field v: default: must be finite");
}

/// @brief it should reject a NaN sample for a float field, in encode and in check.
TEST(JSONEncoder, RejectsNaNSample) {
    auto v = field<icd::FloatJSONField>("v");
    v.scale = 2;
    auto enc = ASSERT_NIL_P(JSONEncoder::create(command({v}, {x::telem::FLOAT64_T})));
    const auto samples = series_of(
        x::telem::Series(std::vector{std::numeric_limits<double>::quiet_NaN()})
    );
    std::vector<std::uint8_t> payload;
    const auto err = enc.encode(samples, payload);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "message command: field v: NaN is not a JSON number");
    const auto check_err = enc.check(0, samples[0], 0);
    ASSERT_OCCURRED_AS(check_err, x::errors::VALIDATION);
    ASSERT_EQ(check_err.data, err.data);
}

/// @brief it should reject an infinite sample before it scales or rounds it.
TEST(JSONEncoder, RejectsInfiniteSampleBeforeConversion) {
    auto v = field<icd::IntegerJSONField>("v");
    v.scale = -1;
    auto enc = ASSERT_NIL_P(JSONEncoder::create(command({v}, {x::telem::FLOAT64_T})));
    const auto samples = series_of(
        x::telem::Series(std::vector{std::numeric_limits<double>::infinity()})
    );
    std::vector<std::uint8_t> payload;
    const auto err = enc.encode(samples, payload);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "message command: field v: Infinity is not a JSON number");
}

/// @brief it should write a boolean sample to a number field as 0 or 1.
TEST(JSONEncoder, WritesBooleanSampleAsZeroOrOne) {
    auto enc = ASSERT_NIL_P(
        JSONEncoder::create(
            command({field<icd::IntegerJSONField>("v")}, {x::telem::BOOLEAN_T})
        )
    );
    const auto samples = series_of(
        x::telem::Series(std::uint8_t{2}, x::telem::BOOLEAN_T)
    );
    ASSERT_EQ(encoded(enc, samples), nlohmann::json({{"v", 1}}));
}

/// @brief it should check the sample at an index of a series.
TEST(JSONEncoder, ChecksSampleAtIndex) {
    auto enc = ASSERT_NIL_P(
        JSONEncoder::create(
            command({field<icd::BooleanJSONField>("on")}, {x::telem::FLOAT64_T})
        )
    );
    const x::telem::Series s(std::vector{1.0, 2.5});
    ASSERT_NIL(enc.check(0, s, 0));
    const auto err = enc.check(0, s, 1);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "message command: field on: 2.5 is not 0 or 1");
}

/// @brief it should write the sample of a JSON field as it is, and reject one that is
/// not JSON.
TEST(JSONEncoder, WritesJSONSample) {
    auto enc = ASSERT_NIL_P(
        JSONEncoder::create(
            command({field<icd::JSONJSONField>("config")}, {x::telem::JSON_T})
        )
    );
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(enc.encode(
        series_of(x::telem::Series(std::string(R"({"rate":10})"), x::telem::JSON_T)),
        payload
    ));
    ASSERT_EQ(std::string(payload.begin(), payload.end()), R"({"config":{"rate":10}})");
    const auto err = enc.encode(
        series_of(x::telem::Series(std::string(R"({"rate")"), x::telem::JSON_T)),
        payload
    );
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "message command: field config: invalid JSON");
}

/// @brief it should reject a JSON sample that repeats a member name, in encode and in
/// check.
TEST(JSONEncoder, RejectsJSONSampleWithRepeatedName) {
    auto enc = ASSERT_NIL_P(
        JSONEncoder::create(
            command({field<icd::JSONJSONField>("config")}, {x::telem::JSON_T})
        )
    );
    const auto samples = series_of(
        x::telem::Series(std::string(R"({"pid":{"p":1,"p":2}})"), x::telem::JSON_T)
    );
    std::vector<std::uint8_t> payload;
    const auto err = enc.encode(samples, payload);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "message command: field config: invalid JSON");
    const auto check_err = enc.check(0, samples[0], 0);
    ASSERT_OCCURRED_AS(check_err, x::errors::VALIDATION);
    ASSERT_EQ(check_err.data, err.data);
}

/// @brief it should encode a document without allocating, once a first encode is done.
TEST(JSONEncoder, EncodesWithoutAllocating) {
    auto temp = field<icd::FloatJSONField>("temp");
    temp.scale = 0.5;
    temp.offset = -10;
    temp.min = 0;
    temp.max = 100;
    auto m = command(
        {temp,
         field<icd::IntegerJSONField>("count"),
         field<icd::StringJSONField>("state")},
        {x::telem::FLOAT64_T, x::telem::FLOAT64_T, x::telem::STRING_T}
    );
    m.payload.match = match<icd::IntegerJSONMatch>("/id", 7);
    auto enc = ASSERT_NIL_P(JSONEncoder::create(m));
    const auto samples = series_of(
        x::telem::Series(std::vector{10.0}),
        x::telem::Series(std::vector{3.0}),
        x::telem::Series(std::vector<std::string>{"run"})
    );
    std::vector<std::uint8_t> payload;
    ASSERT_NIL(enc.encode(samples, payload));
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++)
        ASSERT_NIL(enc.encode(samples, payload));
    EXPECT_EQ(x::bench::alloc_count.load(std::memory_order_relaxed) - before, 0);
    ASSERT_EQ(
        std::string(payload.begin(), payload.end()),
        R"({"id":7,"temp":40,"count":3,"state":"run"})"
    );
}

/// @brief it should write a now default without allocating, once an encode has written
/// a longer document.
TEST(JSONEncoder, WritesNowDefaultWithoutAllocating) {
    auto pad = field<icd::StringJSONField>("pad");
    pad.default_ = "";
    auto v = field<icd::TimestampJSONField>("v");
    v.time_format = "iso8601";
    v.default_ = icd::NowTimestampDefault{};
    auto enc = ASSERT_NIL_P(
        JSONEncoder::create(command({pad, v}, {x::telem::STRING_T}))
    );
    std::vector<std::uint8_t> payload;
    // A now text drops the trailing zeros of its fraction, so a later one can be up to
    // 10 characters longer. The first pad is longer than that.
    ASSERT_NIL(enc.encode(
        series_of(x::telem::Series(std::vector<std::string>{std::string(16, 'p')})),
        payload
    ));
    const auto samples = series_of(x::telem::Series(x::telem::STRING_T, 0));
    const auto before = x::bench::alloc_count.load(std::memory_order_relaxed);
    for (int i = 0; i < 100; i++)
        ASSERT_NIL(enc.encode(samples, payload));
    EXPECT_EQ(x::bench::alloc_count.load(std::memory_order_relaxed) - before, 0);
}

/// @brief it should give a field whose value does not convert no sample, list it in the
/// failures of its message, and decode the other fields.
TEST(JSON, ListsFieldThatDoesNotConvert) {
    auto dec = ASSERT_NIL_P(JSON::create({two_integers()}));
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(R"({"v": 1.5, "w": 2})")), std::size_t{0});
    ASSERT_TRUE(dec.samples(0)[0].empty());
    ASSERT_EQ(dec.samples(0)[1].at<std::int64_t>(0), 2);
    ASSERT_EQ(dec.failures(0).size(), 1U);
    ASSERT_EQ(dec.failures(0)[0].binding, 0U);
    ASSERT_FALSE(dec.failures(0)[0].missing);
    ASSERT_OCCURRED_AS(dec.failures(0)[0].error, x::errors::VALIDATION);
    ASSERT_EQ(dec.failures(0)[0].error.data, "field v: 1.5 is not an integer");
}

/// @brief it should list a field with no value and no default as missing.
TEST(JSON, ListsMissingField) {
    auto dec = ASSERT_NIL_P(JSON::create({two_integers()}));
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(R"({"v": 1})")), std::size_t{0});
    ASSERT_TRUE(dec.samples(0)[1].empty());
    ASSERT_EQ(dec.failures(0).size(), 1U);
    ASSERT_EQ(dec.failures(0)[0].binding, 1U);
    ASSERT_TRUE(dec.failures(0)[0].missing);
    ASSERT_OCCURRED_AS(dec.failures(0)[0].error, x::errors::VALIDATION);
    ASSERT_EQ(dec.failures(0)[0].error.data, "field w: no value at /w");
}

/// @brief it should list no failures for a decode after one that had them.
TEST(JSON, ClearsFailuresOnEachDecode) {
    auto dec = ASSERT_NIL_P(JSON::create({two_integers()}));
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(R"({"v": 1.5, "w": 2})")), std::size_t{0});
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(R"({"v": 1, "w": 2})")), std::size_t{0});
    ASSERT_TRUE(dec.failures(0).empty());
    ASSERT_EQ(dec.samples(0)[0].at<std::int64_t>(0), 1);
}

/// @brief it should name the message of a failure when the decoder holds more than one.
TEST(JSON, NamesMessageOfFailure) {
    auto dec = ASSERT_NIL_P(JSON::create(routed(present("/other"), present("/temp"))));
    ASSERT_EQ(
        ASSERT_NIL_P(dec.decode(R"({"temp": 40, "count": 1.5, "state": "run"})")),
        std::size_t{1}
    );
    ASSERT_EQ(dec.failures(1).size(), 1U);
    ASSERT_EQ(
        dec.failures(1)[0].error.data,
        "message status: field count: 1.5 is not an integer"
    );
}
}
