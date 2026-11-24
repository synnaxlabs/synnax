// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
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
#include "x/cpp/json/extract.h"
#include "x/cpp/json/json.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/json/text.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/json.h"
#include "driver/codec/testutil/fixture.h"

namespace driver::codec::testutil {
namespace {
/// @brief the suffix of a Go validation error, which a C++ error keeps in its type.
constexpr std::string_view VALIDATION_SUFFIX = ": validation error";
}

std::optional<std::string> raw_at(const std::string &doc, const std::string &pointer) {
    x::json::Extractor e({ASSERT_NIL_P(x::json::Pointer::parse(pointer))});
    std::vector<std::optional<std::string_view>> values;
    EXPECT_NIL(e.extract(doc, values));
    const auto &value = values[0];
    if (!value.has_value()) return std::nullopt;
    return std::string(*value);
}

std::string required_at(const std::string &doc, const std::string &pointer) {
    auto raw = raw_at(doc, pointer);
    if (!raw.has_value()) ADD_FAILURE() << "no value at " << pointer;
    return std::move(raw).value_or("");
}

std::string string_at(const std::string &doc, const std::string &pointer) {
    const auto raw = raw_at(doc, pointer);
    if (!raw.has_value()) return "";
    std::string out;
    EXPECT_NIL(x::json::unquote(*raw, out));
    return out;
}

std::string error_data(const std::string &go) {
    EXPECT_TRUE(go.ends_with(VALIDATION_SUFFIX)) << go;
    return go.substr(0, go.size() - VALIDATION_SUFFIX.size());
}

JSONMessage parse_message(const nlohmann::json &m) {
    x::json::Parser parser(m["payload"]);
    JSONMessage msg{
        .name = m["name"].get<std::string>(),
        .payload = synnax::icd::JSONPayload::parse(parser),
    };
    EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
    for (const auto &b: m["bindings"])
        msg.bindings.push_back({
            .field = ASSERT_NIL_P(x::uuid::UUID::parse(b["field"].get<std::string>())),
            .data_type = x::telem::DataType(b["data_type"].get<std::string>()),
        });
    return msg;
}

std::string read_fixture(const std::filesystem::path &path) {
    std::ifstream file(path);
    std::stringstream buf;
    buf << file.rdbuf();
    return buf.str();
}

std::vector<std::filesystem::path> fixture_paths(const std::filesystem::path &dir) {
    std::vector<std::filesystem::path> paths;
    for (const auto &e: std::filesystem::directory_iterator(dir))
        if (e.path().extension() == ".json") paths.push_back(e.path());
    std::ranges::sort(paths);
    return paths;
}
}
