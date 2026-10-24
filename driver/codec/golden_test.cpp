// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cmath>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <limits>
#include <map>
#include <optional>
#include <string>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/icd/json.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/test/test.h"

#include "driver/codec/errors.h"
#include "driver/codec/matcher.h"
#include "driver/codec/plan.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;
using json = nlohmann::json;

/// @brief Golden is one DBC file: each message as the Core's importer writes it to an
/// ICD, and the vectors cantools decodes and encodes for each message it can.
struct Golden {
    std::string file;
    std::vector<icd::Message> messages;
    /// @brief vectors holds the vectors of each message, or nullopt when cantools
    /// skipped it.
    std::vector<std::optional<json>> vectors;
    /// @brief problems lists where the importer's ICD and cantools disagree.
    std::vector<std::string> problems;
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

/// @returns a signal that cantools decodes and the message has no field for, or an
/// empty string.
std::string unknown_signal(const icd::Message &m, const json &vectors) {
    const auto &fs = std::get<icd::BinaryPayload>(m.payload).fields;
    for (const auto &v: vectors)
        for (const auto &[name, _]: v["values"].items())
            if (std::ranges::none_of(fs, [&name](const auto &f) {
                    return f.name == name;
                }))
                return name;
    return "";
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
        auto icd_path = stem;
        icd_path += ".icd.json";
        auto vectors_path = stem;
        vectors_path += ".vectors.json";
        Golden g{.file = path.filename().string()};
        if (!std::filesystem::exists(icd_path)) {
            g.problems.emplace_back("the importer wrote no ICD");
            out.push_back(std::move(g));
            continue;
        }
        x::json::Parser parser(read(icd_path));
        const auto ic = icd::ICD::parse(parser);
        if (!parser.ok()) g.problems.push_back(parser.error().message());
        const auto vectors = read(vectors_path);
        const auto &skipped = vectors["skipped"];
        std::map<std::string, json> by_name;
        for (const auto &m: vectors["messages"])
            by_name[m["name"].get<std::string>()] = m["vectors"];
        for (const auto &m: ic.messages) {
            g.messages.push_back(m);
            const auto it = by_name.find(m.name);
            if (it != by_name.end()) {
                const auto unknown = unknown_signal(m, it->second);
                if (unknown.empty())
                    g.vectors.emplace_back(it->second);
                else {
                    g.vectors.emplace_back(std::nullopt);
                    g.problems.push_back(m.name + " has no field " + unknown);
                }
                by_name.erase(it);
                continue;
            }
            g.vectors.emplace_back(std::nullopt);
            if (skipped.contains(m.name))
                std::cout << g.file << ": cantools skipped " << m.name << ": "
                          << skipped[m.name].get<std::string>() << "\n";
            else if (skipped.contains(g.file))
                std::cout << g.file << ": cantools skipped the file: "
                          << skipped[g.file].get<std::string>() << "\n";
            else
                g.problems.push_back("cantools has no message " + m.name);
        }
        for (const auto &[name, _]: by_name)
            g.problems.push_back("the ICD has no message " + name);
        out.push_back(std::move(g));
    }
    return out;
}

const std::vector<icd::BinaryField> &fields(const icd::Message &m) {
    return std::get<icd::BinaryPayload>(m.payload).fields;
}

/// @returns how far two computations of raw * scale + offset can differ when one
/// fuses the multiply and add and the other rounds the product first.
double rounding(const double value, const double offset) {
    return 4 * std::numeric_limits<double>::epsilon() *
           (std::abs(value) + 2 * std::abs(offset));
}

bool exact(const icd::BinaryField &f) {
    return !f.float_ && f.scale == 1 && f.offset == 0;
}

const std::vector<Golden> &goldens() {
    static const auto g = load();
    return g;
}
}

TEST(GoldenVectors, ImportEveryMessageAndSignalCantoolsLoads) {
    ASSERT_FALSE(goldens().empty());
    for (const auto &g: goldens())
        for (const auto &p: g.problems)
            ADD_FAILURE() << g.file << ": " << p;
}

/// @brief The Driver must compile every message the Core's importer accepts.
TEST(GoldenVectors, CompileEveryMessageTheImporterAccepts) {
    for (const auto &g: goldens()) {
        for (const auto &m: g.messages) {
            const auto [plan, err] = Plan::compile(m);
            EXPECT_FALSE(err) << g.file << " " << m.name << ": " << err;
        }
        const auto [matcher, err] = Matcher::compile(g.messages);
        EXPECT_FALSE(err) << g.file << ": " << err;
    }
}

TEST(GoldenVectors, DecodeToTheValuesCantoolsDecodes) {
    for (const auto &g: goldens())
        for (std::size_t m = 0; m < g.messages.size(); m++) {
            if (!g.vectors[m].has_value()) continue;
            const auto &message = g.messages[m];
            const auto [plan, err] = Plan::compile(message);
            // CompileEveryMessageTheImporterAccepts reports the error.
            if (err) continue;
            auto values = plan.values();
            const auto &fs = fields(message);
            for (const auto &v: *g.vectors[m]) {
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
                        const auto want = e.get<double>();
                        EXPECT_NEAR(values.get(i), want, rounding(want, f.offset))
                            << ctx;
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
TEST(GoldenVectors, EncodeToThePayloadCantoolsEncodes) {
    for (const auto &g: goldens())
        for (std::size_t m = 0; m < g.messages.size(); m++) {
            if (!g.vectors[m].has_value()) continue;
            const auto &message = g.messages[m];
            const auto [plan, err] = Plan::compile(message);
            // CompileEveryMessageTheImporterAccepts reports the error.
            if (err) continue;
            auto values = plan.values();
            const auto &fs = fields(message);
            std::vector<std::uint8_t> payload;
            for (const auto &v: *g.vectors[m]) {
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

TEST(GoldenVectors, MatchEachMessageByItsCANIdentifier) {
    for (const auto &g: goldens()) {
        const auto [matcher, err] = Matcher::compile(g.messages);
        // CompileEveryMessageTheImporterAccepts reports the error.
        if (err) continue;
        for (std::size_t i = 0; i < g.messages.size(); i++) {
            const auto &payload = std::get<icd::BinaryPayload>(g.messages[i].payload);
            const auto &id = std::get<icd::CanIdentifier>(*payload.identifier);
            const auto match = matcher.match(id.id, id.extended);
            if (!match.has_value()) {
                ADD_FAILURE() << g.file << " " << g.messages[i].name;
                continue;
            }
            EXPECT_EQ(
                std::get<icd::CanIdentifier>(
                    *std::get<icd::BinaryPayload>(g.messages[*match].payload).identifier
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
