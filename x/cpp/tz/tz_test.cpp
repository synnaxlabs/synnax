// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstdint>
#include <fstream>
#include <functional>
#include <set>
#include <sstream>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"
#include "x/cpp/tz/tz.h"

namespace x::tz {
namespace {
/// @brief a Saturday in March before daylight time starts, and a day in July.
constexpr int64_t WINTER = 1'772'895'845;
constexpr int64_t SUMMER = 1'784'116'800;

/// @brief returns the TZif data gen.go in x/go/tz builds for rules.tsv.
std::string rule_zone(const std::string &rule, const uint32_t types = 1) {
    std::string b;
    const auto header = [&b](const uint32_t types, const uint32_t chars) {
        b += "TZif2";
        b.append(15, '\0');
        for (const uint32_t n: {0u, 0u, 0u, 0u, types, chars})
            for (int shift = 24; shift >= 0; shift -= 8)
                b += static_cast<char>(n >> shift & 0xFF);
    };
    header(0, 0);
    header(types, 4 * types);
    for (uint32_t i = 0; i < types; i++) {
        b += std::string("\x00\x00\x0E\x10", 4);
        b += std::string("\x00\x00", 2);
    }
    for (uint32_t i = 0; i < types; i++)
        b += std::string("FAL\0", 4);
    b += "\n" + rule + "\n";
    return b;
}

/// @brief calls check with each line's fields and returns the line count.
size_t each_vector(
    const std::string &path,
    const std::function<void(const std::vector<std::string> &)> &check
) {
    std::ifstream f(path);
    EXPECT_TRUE(f.is_open()) << path;
    size_t count = 0;
    std::string line;
    while (std::getline(f, line)) {
        if (line.empty() || line[0] == '#') continue;
        std::vector<std::string> fields;
        std::stringstream ss(line);
        std::string field;
        while (std::getline(ss, field, '\t'))
            fields.push_back(field);
        // A rule vector for the empty rule starts with a tab.
        if (line.back() == '\t') fields.emplace_back();
        EXPECT_EQ(fields.size(), 4u) << line;
        if (fields.size() != 4) continue;
        check(fields);
        count++;
    }
    return count;
}
}

/// @brief a zone loads by its exact name, in standard and daylight time.
TEST(DatabaseTest, LoadsAZoneByItsExactName) {
    Database db;
    const auto zone = ASSERT_NIL_P(db.load("America/Denver"));
    EXPECT_EQ(zone->name(), "America/Denver");
    const auto winter = zone->at(WINTER);
    EXPECT_EQ(winter.abbreviation, "MST");
    EXPECT_EQ(winter.offset, -7 * 3600);
    const auto summer = zone->at(SUMMER);
    EXPECT_EQ(summer.abbreviation, "MDT");
    EXPECT_EQ(summer.offset, -6 * 3600);
}

/// @brief a name matches without regard to case.
TEST(DatabaseTest, MatchesANameWithoutRegardToCase) {
    Database db;
    for (const auto *name: {"america/denver", "AMERICA/DENVER", "aMeRiCa/DeNvEr"})
        EXPECT_EQ(ASSERT_NIL_P(db.load(name))->name(), "America/Denver") << name;
}

/// @brief a short alias has the same rules as its zone.
TEST(DatabaseTest, LoadsAShortAliasWithTheSameRules) {
    Database db;
    const auto alias = ASSERT_NIL_P(db.load("us/mountain"));
    EXPECT_EQ(alias->name(), "US/Mountain");
    const auto denver = ASSERT_NIL_P(db.load("America/Denver"));
    for (const auto sec: {WINTER, SUMMER}) {
        EXPECT_EQ(alias->at(sec).abbreviation, denver->at(sec).abbreviation);
        EXPECT_EQ(alias->at(sec).offset, denver->at(sec).offset);
    }
}

/// @brief a fixed zone has no daylight time.
TEST(DatabaseTest, LoadsAFixedZone) {
    Database db;
    const auto info = ASSERT_NIL_P(db.load("EST"))->at(SUMMER);
    EXPECT_EQ(info.abbreviation, "EST");
    EXPECT_EQ(info.offset, -5 * 3600);
}

/// @brief a name loaded twice gives the same zone.
TEST(DatabaseTest, ReturnsTheSameZoneForANameLoadedTwice) {
    Database db;
    const auto first = ASSERT_NIL_P(db.load("Europe/Berlin"));
    EXPECT_EQ(ASSERT_NIL_P(db.load("europe/berlin")), first);
}

/// @brief a name that is not a zone gives a NOT_FOUND error that quotes it.
TEST(DatabaseTest, RejectsANameThatIsNotAZone) {
    Database db;
    for (const auto *name: {"America/Denvr", "", "PST"}) {
        const auto [zone, err] = db.load(name);
        EXPECT_EQ(zone, nullptr) << name;
        EXPECT_TRUE(err.matches(x::errors::NOT_FOUND)) << name;
        EXPECT_NE(err.data.find("\"" + std::string(name) + "\""), std::string::npos)
            << err.data;
    }
}

/// @brief the database lists every zone in sorted order, and each one loads.
TEST(DatabaseTest, ListsEveryZoneInSortedOrder) {
    Database db;
    const auto names = db.names();
    std::set<std::string> vector_zones;
    each_vector(
        "x/go/tz/testdata/vectors.tsv",
        [&vector_zones](const std::vector<std::string> &f) {
            vector_zones.insert(f[0]);
        }
    );
    EXPECT_EQ(names.size(), vector_zones.size());
    EXPECT_TRUE(std::ranges::is_sorted(names));
    for (const auto *name:
         {"America/Denver",
          "America/Edmonton",
          "US/Mountain",
          "Canada/Mountain",
          "EST",
          "UTC"})
        EXPECT_NE(std::ranges::find(names, name), names.end()) << name;
    for (const auto &name: names)
        EXPECT_FALSE(db.load(name).second) << name;
}

/// @brief every zone matches Go at every vector.
TEST(DatabaseTest, MatchesEveryZoneVector) {
    Database db;
    const auto count = each_vector(
        "x/go/tz/testdata/vectors.tsv",
        [&db](const std::vector<std::string> &f) {
            const auto zone = ASSERT_NIL_P(db.load(f[0]));
            const auto info = zone->at(std::stoll(f[1]));
            EXPECT_EQ(std::to_string(info.offset), f[2]) << f[0] << " " << f[1];
            EXPECT_EQ(info.abbreviation, f[3]) << f[0] << " " << f[1];
        }
    );
    EXPECT_GT(count, 15'000u);
}

/// @brief every rule string reads as Go reads it.
TEST(ZoneTest, MatchesEveryRuleVector) {
    const auto count = each_vector(
        "x/go/tz/testdata/rules.tsv",
        [](const std::vector<std::string> &f) {
            const auto zone = ASSERT_NIL_P(Zone::parse("rule", rule_zone(f[0])));
            const auto info = zone->at(std::stoll(f[1]));
            EXPECT_EQ(std::to_string(info.offset), f[2]) << f[0] << " " << f[1];
            EXPECT_EQ(info.abbreviation, f[3]) << f[0] << " " << f[1];
        }
    );
    EXPECT_GT(count, 1'500u);
}

/// @brief data that is not valid TZif gives a VALIDATION error.
TEST(ZoneTest, RejectsDataThatIsNotTZif) {
    const auto valid = rule_zone("EST5");
    auto bad_version = valid;
    bad_version[4] = '9';
    for (const auto &data:
         {std::string("TZjf"),
          valid.substr(0, 30),
          bad_version,
          rule_zone("EST5", 0),
          std::string("")})
        ASSERT_OCCURRED_AS_P(Zone::parse("bad", data), x::errors::VALIDATION);
}

}
