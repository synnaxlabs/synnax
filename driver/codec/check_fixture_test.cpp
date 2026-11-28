// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <string>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/icd/json.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/binary.h"
#include "driver/codec/framing/framing.h"
#include "driver/codec/json.h"
#include "driver/codec/testutil/fixture.h"
#include "driver/codec/text.h"

namespace driver::codec {
namespace {
/// @brief the Core's check fixtures. Each holds one ICD, and an invalid one says
/// whether the framer or a codec rejects it too.
const std::filesystem::path FIXTURES = "core/pkg/service/icd/testdata/fixtures";

/// @brief returns the data type of a channel that takes a field of kind unscaled.
x::telem::DataType data_type_of(const std::string &kind) {
    if (kind == "float") return x::telem::FLOAT64_T;
    if (kind == "integer") return x::telem::INT64_T;
    if (kind == "unsigned") return x::telem::UINT64_T;
    if (kind == "boolean") return x::telem::BOOLEAN_T;
    if (kind == "string") return x::telem::STRING_T;
    if (kind == "timestamp") return x::telem::TIMESTAMP_T;
    if (kind == "json") return x::telem::JSON_T;
    if (kind == "uuid") return x::telem::UUID_T;
    if (kind == "bytes") return x::telem::BYTES_T;
    ADD_FAILURE() << "unknown field kind " << kind;
    return x::telem::UNKNOWN_T;
}

/// @brief returns m with each field bound as the Go check test binds it: to the data
/// type of its kind, or to float64 when it scales or is text.
template<typename Message>
Message bind(nlohmann::json m) {
    const auto &payload = m["payload"];
    auto bindings = nlohmann::json::array();
    for (const auto &f: payload["fields"]) {
        const bool scaled = f.value("scale", 1.0) != 1 || f.value("offset", 0.0) != 0;
        auto dt = x::telem::FLOAT64_T;
        if (!scaled && payload["format"] != "text")
            dt = data_type_of(f.value("kind", f.value("type", "")));
        bindings.push_back({{"field", f["key"]}, {"data_type", dt.name()}});
    }
    m["bindings"] = bindings;
    return testutil::parse_message<Message>(m);
}

/// @brief reports whether the Decoder of the messages of format rejects them.
template<typename Decoder, typename Message>
bool rejects(const nlohmann::json &messages, const std::string &format) {
    std::vector<Message> bound;
    for (const auto &m: messages)
        if (m["payload"]["format"] == format) bound.push_back(bind<Message>(m));
    if (bound.empty()) return false;
    return static_cast<bool>(Decoder::create(bound).second);
}

/// @brief reports whether the framer of icd rejects its framing.
bool framing_rejects(const nlohmann::json &icd) {
    const auto f = icd.value("framing", nlohmann::json());
    if (f.is_null()) return false;
    x::json::Parser parser(f);
    const auto framing = synnax::icd::parse_framing(parser);
    EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
    return static_cast<bool>(framing::create(framing).second);
}

/// @brief reports whether the framer or a codec of the Driver rejects icd.
bool codec_rejects(const nlohmann::json &icd) {
    const auto &messages = icd["messages"];
    return framing_rejects(icd) || rejects<Binary, BinaryMessage>(messages, "binary") ||
           rejects<JSON, JSONMessage>(messages, "json") ||
           rejects<Text, TextMessage>(messages, "text");
}
}

/// @brief it should build the framer and codecs of every valid check fixture of the
/// Core.
TEST(CheckFixtures, ValidBuildTheCodecs) {
    const auto paths = testutil::fixture_paths(FIXTURES / "valid");
    ASSERT_FALSE(paths.empty());
    for (const auto &path: paths) {
        SCOPED_TRACE(path.filename().string());
        const auto doc = nlohmann::json::parse(testutil::read_fixture(path));
        EXPECT_FALSE(codec_rejects(doc["icd"]));
    }
}

/// @brief it should reject an invalid check fixture of the Core in the framer or a
/// codec exactly when the fixture says so.
TEST(CheckFixtures, InvalidRejectAsTheySay) {
    const auto paths = testutil::fixture_paths(FIXTURES / "invalid");
    ASSERT_FALSE(paths.empty());
    for (const auto &path: paths) {
        SCOPED_TRACE(path.filename().string());
        const auto doc = nlohmann::json::parse(testutil::read_fixture(path));
        EXPECT_EQ(codec_rejects(doc["icd"]), doc.at("codec_rejects").get<bool>());
    }
}
}
