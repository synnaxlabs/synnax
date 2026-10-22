// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/library/json.gen.h"
#include "client/cpp/library/types.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/test/test.h"

#include "driver/codec/errors.h"
#include "driver/codec/matcher.h"
#include "driver/codec/plan.h"

namespace driver::codec {
namespace {
namespace library = synnax::library;
using json = nlohmann::json;

/// @brief Golden is one DBC file: each message as the Core's importer writes it to a
/// library, and the vectors cantools decodes and encodes for that message.
struct Golden {
    std::string file;
    std::vector<library::MessageEntry> messages;
    std::vector<json> vectors;
    json skipped;
};

/// @returns the directory of DBC files with their golden JSON. SYNNAX_GOLDEN_DIR
/// replaces it, as for a downloaded corpus.
std::filesystem::path golden_dir() {
    if (const char *dir = std::getenv("SYNNAX_GOLDEN_DIR")) return dir;
    return "driver/codec/testdata/dbc";
}

json read(const std::filesystem::path &path) {
    std::ifstream file(path);
    return json::parse(file);
}

std::vector<std::uint8_t> from_hex(const std::string &hex) {
    std::vector<std::uint8_t> out;
    for (std::size_t i = 0; i < hex.size(); i += 2)
        out.push_back(
            static_cast<std::uint8_t>(std::stoul(hex.substr(i, 2), nullptr, 16))
        );
    return out;
}

std::vector<Golden> load() {
    std::vector<std::filesystem::path> paths;
    for (const auto &e: std::filesystem::directory_iterator(golden_dir()))
        if (e.path().extension() == ".dbc") paths.push_back(e.path());
    std::ranges::sort(paths);
    std::vector<Golden> out;
    for (const auto &path: paths) {
        auto stem = path;
        stem.replace_extension();
        auto lib_path = stem;
        lib_path += ".library.json";
        auto vectors_path = stem;
        vectors_path += ".vectors.json";
        x::json::Parser parser(read(lib_path));
        const auto lib = library::Library::parse(parser);
        EXPECT_TRUE(parser.ok()) << lib_path << ": " << parser.error();
        const auto vectors = read(vectors_path);
        Golden g{.file = path.filename().string(), .skipped = vectors["skipped"]};
        for (const auto &m: vectors["messages"]) {
            const auto name = m["name"].get<std::string>();
            const auto it = std::ranges::find_if(lib.entries, [&name](const auto &e) {
                const auto *msg = std::get_if<library::MessageEntry>(&e);
                return msg != nullptr && msg->name == name;
            });
            if (it == lib.entries.end()) {
                ADD_FAILURE() << g.file << ": the library has no message " << name;
                continue;
            }
            g.messages.push_back(std::get<library::MessageEntry>(*it));
            g.vectors.push_back(m["vectors"]);
        }
        out.push_back(std::move(g));
    }
    return out;
}

const std::vector<library::BinaryField> &fields(const library::MessageEntry &m) {
    return std::get<library::BinaryPayload>(m.payload).fields;
}

bool exact(const library::BinaryField &f) {
    return !f.float_ && f.scale == 1 && f.offset == 0;
}

class GoldenVectors : public ::testing::Test {
protected:
    std::vector<Golden> goldens = load();
};
}

TEST_F(GoldenVectors, CoverEveryMessageOfEveryDBC) {
    ASSERT_FALSE(this->goldens.empty());
    for (const auto &g: this->goldens) {
        EXPECT_TRUE(g.skipped.empty()) << g.file << ": " << g.skipped.dump();
        EXPECT_FALSE(g.messages.empty()) << g.file;
        for (std::size_t i = 0; i < g.messages.size(); i++)
            EXPECT_GE(g.vectors[i].size(), 4) << g.file << " " << g.messages[i].name;
    }
}

TEST_F(GoldenVectors, DecodeToTheValuesCantoolsDecodes) {
    for (const auto &g: this->goldens)
        for (std::size_t m = 0; m < g.messages.size(); m++) {
            const auto &message = g.messages[m];
            const auto plan = ASSERT_NIL_P(Plan::compile(message));
            auto values = plan.values();
            const auto &fs = fields(message);
            for (const auto &v: g.vectors[m]) {
                const auto payload = from_hex(v["payload"]);
                ASSERT_NIL(plan.decode(payload, values));
                const auto &expected = v["values"];
                for (std::size_t i = 0; i < fs.size(); i++) {
                    const auto &f = fs[i];
                    const auto ctx = g.file + " " + message.name + "." + f.name + " " +
                                     v["payload"].get<std::string>();
                    if (!expected.contains(f.name)) {
                        EXPECT_FALSE(values.present(i)) << ctx;
                        continue;
                    }
                    ASSERT_TRUE(values.present(i)) << ctx;
                    const auto &e = expected[f.name];
                    if (!exact(f)) {
                        EXPECT_DOUBLE_EQ(values.get(i), e.get<double>()) << ctx;
                        continue;
                    }
                    if (f.signed_) {
                        x::telem::Series s(x::telem::INT64_T, 1);
                        ASSERT_EQ(values.write(i, s), 1);
                        EXPECT_EQ(s.at<std::int64_t>(0), e.get<std::int64_t>()) << ctx;
                    } else {
                        x::telem::Series s(x::telem::UINT64_T, 1);
                        ASSERT_EQ(values.write(i, s), 1);
                        EXPECT_EQ(s.at<std::uint64_t>(0), e.get<std::uint64_t>())
                            << ctx;
                    }
                }
            }
        }
}

/// @brief cantools encodes any value, so a value outside a field's min and max must be
/// refused, and the payload is compared only when every value is accepted.
TEST_F(GoldenVectors, EncodeToThePayloadCantoolsEncodes) {
    for (const auto &g: this->goldens)
        for (std::size_t m = 0; m < g.messages.size(); m++) {
            const auto &message = g.messages[m];
            const auto plan = ASSERT_NIL_P(Plan::compile(message));
            auto values = plan.values();
            const auto &fs = fields(message);
            std::vector<std::uint8_t> payload;
            for (const auto &v: g.vectors[m]) {
                const auto ctx = g.file + " " + message.name + " " +
                                 v["payload"].get<std::string>();
                values.clear();
                bool accepted = true;
                for (std::size_t i = 0; i < fs.size(); i++) {
                    const auto &f = fs[i];
                    const auto &expected = v["values"];
                    if (!expected.contains(f.name)) continue;
                    const auto &e = expected[f.name];
                    x::telem::SampleValue sample;
                    if (!exact(f))
                        sample = e.get<double>();
                    else if (f.signed_)
                        sample = e.get<std::int64_t>();
                    else
                        sample = e.get<std::uint64_t>();
                    const auto physical = e.get<double>();
                    if ((f.min && physical < *f.min) || (f.max && physical > *f.max)) {
                        ASSERT_OCCURRED_AS(plan.set(values, i, sample), RANGE_ERROR);
                        accepted = false;
                        continue;
                    }
                    ASSERT_NIL(plan.set(values, i, sample));
                }
                if (!accepted) continue;
                payload.assign(plan.length(), 0);
                plan.encode(values, payload);
                EXPECT_EQ(payload, from_hex(v["encoded"])) << ctx;
            }
        }
}

TEST_F(GoldenVectors, MatchEachMessageByItsCANIdentifier) {
    for (const auto &g: this->goldens) {
        const auto matcher = ASSERT_NIL_P(Matcher::compile(g.messages));
        for (std::size_t i = 0; i < g.messages.size(); i++) {
            const auto &payload = std::get<library::BinaryPayload>(
                g.messages[i].payload
            );
            const auto &id = std::get<library::CanIdentifier>(*payload.identifier);
            const auto match = matcher.match(id.id, id.extended);
            ASSERT_TRUE(match.has_value()) << g.file << " " << g.messages[i].name;
            EXPECT_EQ(
                std::get<library::CanIdentifier>(
                    *std::get<library::BinaryPayload>(g.messages[*match].payload)
                         .identifier
                )
                    .id,
                id.id
            ) << g.file
              << " " << g.messages[i].name;
            EXPECT_EQ(matcher.match(id.id, !id.extended), std::nullopt)
                << g.file << " " << g.messages[i].name;
        }
    }
}
}
