// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <map>
#include <string>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/channel/json.gen.h"
#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/json.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/codec/testutil/fixture.h"
#include "driver/errors/errors.h"
#include "driver/icd/receiver.h"

namespace driver::icd {
namespace {
/// @brief the Core's task fixtures. Each holds the entries of a bus read task, with
/// their ICDs and channels, and an invalid one holds the error that refuses it.
const std::filesystem::path FIXTURES = "core/pkg/service/icd/testdata/task";

/// @brief returns each value of values, parsed as a T.
template<typename T>
std::vector<T> parse_all(const nlohmann::json &values) {
    std::vector<T> out;
    for (const auto &v: values) {
        x::json::Parser parser(v);
        out.push_back(T::parse(parser));
        EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
    }
    return out;
}

/// @brief returns the error of a stream Receiver for the entries of the fixture doc.
x::errors::Error create(const nlohmann::json &doc) {
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;
    for (auto &ch: parse_all<synnax::channel::Channel>(doc.at("channels")))
        channels.emplace(ch.key, std::move(ch));
    return Receiver::create(
               parse_all<synnax::icd::ReadEntry>(doc.at("entries")),
               parse_all<synnax::icd::ICD>(doc.at("icds")),
               channels,
               false
    )
        .second;
}

/// @brief returns the fixture at path.
nlohmann::json load(const std::filesystem::path &path) {
    auto doc = nlohmann::json::parse(codec::testutil::read_fixture(path));
    EXPECT_FALSE(doc.at("description").get<std::string>().empty());
    return doc;
}
}

/// @brief it should accept every valid task fixture of the Core.
TEST(ReceiverFixtures, AcceptValidTasks) {
    const auto paths = codec::testutil::fixture_paths(FIXTURES / "valid");
    ASSERT_FALSE(paths.empty());
    for (const auto &path: paths) {
        SCOPED_TRACE(path.filename().string());
        EXPECT_NIL(create(load(path)));
    }
}

/// @brief it should refuse every invalid task fixture of the Core with its error.
TEST(ReceiverFixtures, RefuseInvalidTasksWithTheirErrors) {
    const auto paths = codec::testutil::fixture_paths(FIXTURES / "invalid");
    ASSERT_FALSE(paths.empty());
    for (const auto &path: paths) {
        SCOPED_TRACE(path.filename().string());
        const auto doc = load(path);
        const auto err = create(doc);
        EXPECT_TRUE(err.matches(driver::errors::CONFIGURATION_ERROR)) << err;
        EXPECT_EQ(err.data(), doc.at("error").get<std::string>());
    }
}
}
