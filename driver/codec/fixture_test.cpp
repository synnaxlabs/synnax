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
#include <string>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/icd/json.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/json/json.h"

#include "driver/codec/matcher.h"
#include "driver/codec/plan.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;

/// @brief the Core's check fixtures. Each holds an ICD, and an invalid one says whether
/// the codec also rejects it.
const std::filesystem::path FIXTURES = "core/pkg/service/icd/testdata/fixtures";

struct Fixture {
    std::string file;
    bool codec_rejects = false;
    icd::ICD ic;
};

std::vector<Fixture> load(const std::string &kind) {
    std::vector<std::filesystem::path> paths;
    for (const auto &e: std::filesystem::directory_iterator(FIXTURES / kind))
        if (e.path().extension() == ".json") paths.push_back(e.path());
    std::ranges::sort(paths);
    std::vector<Fixture> out;
    for (const auto &path: paths) {
        std::ifstream file(path);
        const auto data = nlohmann::json::parse(file);
        x::json::Parser parser(data["icd"]);
        Fixture f{
            .file = path.filename().string(),
            .codec_rejects = data.value("codec_rejects", false),
            .ic = icd::ICD::parse(parser),
        };
        EXPECT_TRUE(parser.ok()) << f.file << ": " << parser.error_json().dump();
        out.push_back(std::move(f));
    }
    return out;
}

/// @returns the first error of compiling each message of ic and a matcher over them.
x::errors::Error compile(const icd::ICD &ic) {
    for (const auto &m: ic.messages)
        if (auto [_, err] = Plan::compile(m); err) return err;
    return Matcher::compile(ic.messages).second;
}
}

TEST(CheckFixtures, CompileEveryValidICD) {
    const auto fixtures = load("valid");
    ASSERT_FALSE(fixtures.empty());
    for (const auto &f: fixtures) {
        const auto err = compile(f.ic);
        EXPECT_FALSE(err) << f.file << ": " << err;
    }
}

TEST(CheckFixtures, RejectExactlyTheInvalidICDsTheCoreSaysTheCodecRejects) {
    const auto fixtures = load("invalid");
    ASSERT_FALSE(fixtures.empty());
    for (const auto &f: fixtures) {
        const auto err = compile(f.ic);
        EXPECT_EQ(static_cast<bool>(err), f.codec_rejects) << f.file << ": " << err;
    }
}
}
