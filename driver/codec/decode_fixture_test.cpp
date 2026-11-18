// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <optional>
#include <sstream>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/icd/json.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/base64/base64.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/extract.h"
#include "x/cpp/json/json.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/json/text.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/codec/json.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;

/// @brief the Core's decode fixtures. Each holds the JSON messages of one source and
/// either the error that creating a decoder for them returns, or the documents they
/// decode.
const std::filesystem::path FIXTURES = "core/pkg/service/icd/testdata/decode";

/// @brief the suffix of a Go validation error, which a C++ error keeps in its type.
constexpr std::string_view VALIDATION_SUFFIX = ": validation error";

/// @brief the prefix of an error from Go's JSON syntax parser, whose text the Driver
/// does not copy.
constexpr std::string_view SYNTAX_PREFIX = "jsontext: ";

/// @brief returns the text of the value at pointer in doc. The fixtures hold numbers
/// such as 1e999 that nlohmann rejects, so it reads them as text.
std::optional<std::string> raw_at(const std::string &doc, const std::string &pointer) {
    x::json::Extractor e({ASSERT_NIL_P(x::json::Pointer::parse(pointer))});
    std::vector<std::optional<std::string_view>> values;
    EXPECT_NIL(e.extract(doc, values));
    const auto &value = values[0];
    if (!value.has_value()) return std::nullopt;
    return std::string(*value);
}

/// @brief returns the text of the value at pointer in doc, failing when there is none.
std::string required_at(const std::string &doc, const std::string &pointer) {
    auto raw = raw_at(doc, pointer);
    if (!raw.has_value()) ADD_FAILURE() << "no value at " << pointer;
    return std::move(raw).value_or("");
}

/// @brief returns the string at pointer in doc, or an empty string when there is none.
std::string string_at(const std::string &doc, const std::string &pointer) {
    const auto raw = raw_at(doc, pointer);
    if (!raw.has_value()) return "";
    std::string out;
    EXPECT_NIL(x::json::unquote(*raw, out));
    return out;
}

/// @brief returns the error text of a Go validation error without its suffix.
std::string error_data(const std::string &go) {
    EXPECT_TRUE(go.ends_with(VALIDATION_SUFFIX)) << go;
    return go.substr(0, go.size() - VALIDATION_SUFFIX.size());
}

std::vector<JSONMessage> parse_messages(const std::string &doc) {
    std::vector<JSONMessage> out;
    for (const auto &m: nlohmann::json::parse(required_at(doc, "/messages"))) {
        x::json::Parser parser(m["payload"]);
        JSONMessage msg{
            .name = m["name"].get<std::string>(),
            .payload = icd::JSONPayload::parse(parser),
        };
        EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
        for (const auto &b: m["bindings"])
            msg.bindings.push_back({
                .field = ASSERT_NIL_P(
                    x::uuid::UUID::parse(b["field"].get<std::string>())
                ),
                .data_type = x::telem::DataType(b["data_type"].get<std::string>()),
            });
        out.push_back(std::move(msg));
    }
    return out;
}

/// @brief returns the JSON form of the one sample in s, as the fixtures write it.
nlohmann::json sample_json(const x::telem::Series &s) {
    const auto dt = s.data_type();
    if (dt == x::telem::FLOAT64_T) return s.at<double>(0);
    if (dt == x::telem::FLOAT32_T) return static_cast<double>(s.at<float>(0));
    if (dt == x::telem::INT64_T) return s.at<std::int64_t>(0);
    if (dt == x::telem::INT32_T) return s.at<std::int32_t>(0);
    if (dt == x::telem::INT16_T) return s.at<std::int16_t>(0);
    if (dt == x::telem::INT8_T) return s.at<std::int8_t>(0);
    if (dt == x::telem::UINT64_T) return s.at<std::uint64_t>(0);
    if (dt == x::telem::TIMESTAMP_T)
        return static_cast<std::uint64_t>(s.at<x::telem::TimeStamp>(0).nanoseconds());
    if (dt == x::telem::UINT32_T) return s.at<std::uint32_t>(0);
    if (dt == x::telem::UINT16_T) return s.at<std::uint16_t>(0);
    if (dt == x::telem::UINT8_T) return s.at<std::uint8_t>(0);
    if (dt == x::telem::BOOLEAN_T) return s.at<std::uint8_t>(0) == 1;
    if (dt == x::telem::STRING_T) return s.at<std::string>(0);
    if (dt == x::telem::JSON_T) return nlohmann::json::parse(s.at<std::string>(0));
    if (dt == x::telem::UUID_T) return s.uuids()[0].to_string();
    if (dt == x::telem::BYTES_T) return x::base64::encode(s.at<std::string>(0));
    ADD_FAILURE() << "unknown data type " << dt.name();
    return nullptr;
}

/// @brief checks the decode of one case of a fixture.
void check_case(
    JSON &dec,
    const std::vector<JSONMessage> &messages,
    const std::string &doc,
    const std::string &c
) {
    const auto document = raw_at(doc, c + "/text").has_value()
                            ? string_at(doc, c + "/text")
                            : required_at(doc, c + "/document");
    const auto [message, err] = dec.decode(document);
    if (const auto expected = string_at(doc, c + "/error"); !expected.empty()) {
        ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
        if (!expected.starts_with(SYNTAX_PREFIX)) {
            ASSERT_EQ(err.data, error_data(expected));
        }
        return;
    }
    ASSERT_NIL(err);
    if (raw_at(doc, c + "/unmatched") == "true") {
        ASSERT_FALSE(message.has_value());
        return;
    }
    const auto name = string_at(doc, c + "/message");
    const auto want = static_cast<std::size_t>(
        std::ranges::find(messages, name, &JSONMessage::name) - messages.begin()
    );
    ASSERT_LT(want, messages.size()) << "no message " << name;
    ASSERT_EQ(message, want);
    const auto expected = nlohmann::json::parse(required_at(doc, c + "/samples"));
    const auto &samples = dec.samples(want);
    ASSERT_EQ(samples.size(), expected.size());
    for (std::size_t k = 0; k < samples.size(); k++) {
        ASSERT_EQ(samples[k].size(), 1) << "binding " << k;
        ASSERT_EQ(sample_json(samples[k]), expected[k]) << "binding " << k;
    }
}

/// @brief checks that a decoder for the messages of the fixture at path decodes its
/// documents, or fails to create with its error.
void check_fixture(const std::filesystem::path &path) {
    std::ifstream file(path);
    std::stringstream buf;
    buf << file.rdbuf();
    const auto doc = buf.str();
    ASSERT_FALSE(string_at(doc, "/description").empty());
    const auto messages = parse_messages(doc);
    auto [dec, err] = JSON::create(messages);
    if (const auto expected = string_at(doc, "/error"); !expected.empty()) {
        ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
        ASSERT_EQ(err.data, error_data(expected));
        return;
    }
    ASSERT_NIL(err);
    ASSERT_TRUE(raw_at(doc, "/cases/0").has_value());
    for (std::size_t i = 0; raw_at(doc, "/cases/" + std::to_string(i)); i++) {
        SCOPED_TRACE("case " + std::to_string(i));
        check_case(dec, messages, doc, "/cases/" + std::to_string(i));
    }
}
}

/// @brief it should decode the documents of every decode fixture of the Core, or reject
/// its messages with the same error.
TEST(DecodeFixtures, MatchTheCore) {
    std::vector<std::filesystem::path> paths;
    for (const auto &e: std::filesystem::directory_iterator(FIXTURES))
        if (e.path().extension() == ".json") paths.push_back(e.path());
    std::ranges::sort(paths);
    ASSERT_FALSE(paths.empty());
    for (const auto &path: paths) {
        SCOPED_TRACE(path.filename().string());
        check_fixture(path);
    }
}
}
