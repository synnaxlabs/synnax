// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <limits>
#include <string>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "x/cpp/base64/base64.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/number.h"
#include "x/cpp/json/text.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/binary.h"
#include "driver/codec/errors.h"
#include "driver/codec/json.h"
#include "driver/codec/testutil/fixture.h"
#include "driver/codec/text.h"

namespace driver::codec {
namespace {
using testutil::error_data;
using testutil::integer_of;
using testutil::raw_at;
using testutil::required_at;
using testutil::string_at;

/// @brief the Core's encode fixtures. Each holds one JSON, text, or binary message and
/// either the error that creating an encoder for it returns, or the samples it encodes.
const std::filesystem::path FIXTURES = "core/pkg/service/icd/testdata/encode";

/// @brief returns the float that raw holds: a JSON number, or a string for NaN,
/// Infinity, and -Infinity.
double float_of(const std::string &raw) {
    if (raw == "\"NaN\"") return std::numeric_limits<double>::quiet_NaN();
    if (raw == "\"Infinity\"") return std::numeric_limits<double>::infinity();
    if (raw == "\"-Infinity\"") return -std::numeric_limits<double>::infinity();
    return ASSERT_NIL_P(x::json::parse_float64(raw));
}

/// @brief returns the series of dt that holds the sample raw, or an empty series for
/// null. An integer reads without a double, which holds only 53 bits.
x::telem::Series sample_of(const x::telem::DataType &dt, const std::string &raw) {
    if (raw == "null") return {dt, std::size_t{0}};
    if (dt == x::telem::FLOAT64_T || dt == x::telem::FLOAT32_T)
        return x::telem::Series(dt.cast(float_of(raw)), dt);
    if (dt == x::telem::INT64_T || dt == x::telem::INT32_T || dt == x::telem::INT16_T ||
        dt == x::telem::INT8_T || dt == x::telem::TIMESTAMP_T)
        return x::telem::Series(dt.cast(integer_of<std::int64_t>(raw)), dt);
    if (dt == x::telem::UINT64_T || dt == x::telem::UINT32_T ||
        dt == x::telem::UINT16_T || dt == x::telem::UINT8_T)
        return x::telem::Series(dt.cast(integer_of<std::uint64_t>(raw)), dt);
    if (dt == x::telem::BOOLEAN_T)
        return x::telem::Series(static_cast<std::uint8_t>(raw == "true"), dt);
    if (dt == x::telem::JSON_T) return x::telem::Series(raw, dt);
    std::string text;
    EXPECT_NIL(x::json::unquote(raw, text));
    if (dt == x::telem::STRING_T) return x::telem::Series(text, dt);
    if (dt == x::telem::BYTES_T)
        return x::telem::Series(ASSERT_NIL_P(x::base64::decode_strict(text)), dt);
    if (dt == x::telem::UUID_T) {
        const auto id = ASSERT_NIL_P(x::uuid::UUID::parse(text));
        x::telem::Series s(dt, 1);
        s.write(id.data(), 1);
        return s;
    }
    ADD_FAILURE() << "unknown data type " << dt.name();
    return {dt, std::size_t{0}};
}

/// @brief checks the encode of one case of a fixture into payload, which holds the
/// payload of the case before it.
template<typename Encoder, typename Message>
void check_case(
    Encoder &enc,
    const Message &message,
    const std::string &doc,
    const std::string &c,
    std::vector<std::uint8_t> &payload
) {
    std::vector<x::telem::Series> samples;
    std::vector<x::errors::Error> rejected;
    for (std::size_t k = 0; raw_at(doc, c + "/samples/" + std::to_string(k)); k++) {
        ASSERT_LT(k, message.bindings.size());
        samples.push_back(sample_of(
            message.bindings[k].data_type,
            required_at(doc, c + "/samples/" + std::to_string(k))
        ));
        if (!samples.back().empty())
            if (auto err = enc.check(k, samples.back(), 0)) rejected.push_back(err);
    }
    ASSERT_EQ(samples.size(), message.bindings.size());
    const auto err = enc.encode(samples, payload);
    if (const auto expected = string_at(doc, c + "/error"); !expected.empty()) {
        ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
        ASSERT_EQ(err.data, error_data(expected));
        ASSERT_EQ(rejected.size(), 1);
        ASSERT_OCCURRED_AS(rejected[0], x::errors::VALIDATION);
        ASSERT_EQ(rejected[0].data, err.data);
        return;
    }
    ASSERT_NIL(err);
    ASSERT_TRUE(rejected.empty()) << rejected[0];
    if (raw_at(doc, c + "/hex").has_value()) {
        ASSERT_EQ(payload, testutil::bytes_at(doc, c + "/hex"));
        return;
    }
    const std::string text(payload.begin(), payload.end());
    if (raw_at(doc, c + "/text").has_value()) {
        ASSERT_EQ(text, string_at(doc, c + "/text"));
        return;
    }
    ASSERT_EQ(
        nlohmann::json::parse(text),
        nlohmann::json::parse(required_at(doc, c + "/document"))
    );
}

/// @brief checks that an Encoder for the message of doc encodes the samples of its
/// cases, or fails to create with its error.
template<typename Encoder, typename Message>
void check_codec(const std::string &doc) {
    const auto message = testutil::parse_message<Message>(
        nlohmann::json::parse(required_at(doc, "/message"))
    );
    auto [enc, err] = Encoder::create(message);
    if (const auto expected = string_at(doc, "/error"); !expected.empty()) {
        ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
        ASSERT_EQ(err.data, error_data(expected));
        return;
    }
    ASSERT_NIL(err);
    ASSERT_TRUE(raw_at(doc, "/cases/0").has_value());
    // The writer encodes each command into one buffer, so the cases share one too.
    std::vector<std::uint8_t> payload;
    for (std::size_t i = 0; raw_at(doc, "/cases/" + std::to_string(i)); i++) {
        SCOPED_TRACE("case " + std::to_string(i));
        check_case(enc, message, doc, "/cases/" + std::to_string(i), payload);
    }
}

/// @brief checks the fixture at path with the encoder of the payload format of its
/// message.
void check_fixture(const std::filesystem::path &path) {
    const auto doc = testutil::read_fixture(path);
    ASSERT_FALSE(string_at(doc, "/description").empty());
    const auto format = string_at(doc, "/message/payload/format");
    if (format == "json")
        check_codec<JSONEncoder, JSONMessage>(doc);
    else if (format == "text")
        check_codec<TextEncoder, TextMessage>(doc);
    else if (format == "binary")
        check_codec<BinaryEncoder, BinaryMessage>(doc);
    else
        FAIL() << "unknown payload format " << format;
}
}

/// @brief it should encode the samples of every encode fixture of the Core, or reject
/// its message or samples with the same error.
TEST(EncodeFixtures, MatchTheCore) {
    const auto paths = testutil::fixture_paths(FIXTURES);
    ASSERT_FALSE(paths.empty());
    for (const auto &path: paths) {
        SCOPED_TRACE(path.filename().string());
        check_fixture(path);
    }
}
}
