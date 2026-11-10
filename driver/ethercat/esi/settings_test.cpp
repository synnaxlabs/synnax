// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/settings.h"
#include "driver/ethercat/testutil/stg.h"

namespace driver::ethercat::esi {
namespace {
using testutil::DEWESOFT;
using testutil::IOLITE_6XSTG;

const coe::Read read_stg = testutil::read_of(testutil::stg_objects());

/// @brief returns a read of the 6xSTG tables in which input 1 counts count ranges.
coe::Read counting_ranges(const uint32_t count) {
    auto objects = testutil::stg_objects();
    objects[{0x8050, 3}] = count;
    return testutil::read_of(std::move(objects));
}

std::vector<std::string> names(const x::json::json &options) {
    std::vector<std::string> result;
    for (const auto &option: options)
        result.push_back(option["name"].get<std::string>());
    return result;
}

/// @brief returns the nth setting of channel at sub_index.
const x::json::json &
setting(const x::json::json &channel, const uint8_t sub_index, const size_t nth = 0) {
    size_t found = 0;
    for (const auto &s: channel["settings"])
        if (s["sub_index"] == sub_index && found++ == nth) return s;
    throw std::runtime_error("no setting at sub-index " + std::to_string(sub_index));
}
}

/// @brief it should describe the six inputs of a 6xSTG with the ranges, filters, and
/// excitation limits that the card holds.
TEST(ReadSettings, Describes6xSTGInputsFromTheCardTables) {
    const auto settings = ASSERT_NIL_P(read_settings(DEWESOFT, IOLITE_6XSTG, read_stg));
    ASSERT_TRUE(settings.has_value());
    const auto &channels = settings.value()["channels"];
    ASSERT_EQ(channels.size(), 6);
    const auto &input_2 = channels[1];
    const auto &range = setting(input_2, 3);
    EXPECT_EQ(range["index"], 0x8041);
    EXPECT_EQ(
        names(range["options"]),
        (std::vector<std::string>{"50 V", "10 V", "3 V", "1 V", "0.1 V"})
    );
    EXPECT_EQ(range["options"][4]["value"], 4);
    EXPECT_FALSE(range.contains("options_from"));
    EXPECT_EQ(range["visible_when"][0]["index"], 0x8041);
    const auto &filter = setting(input_2, 4);
    EXPECT_EQ(names(filter["options"]), (std::vector<std::string>{"20 kHz", "1 kHz"}));
    EXPECT_EQ(filter["options"][0]["value"], 1);
    EXPECT_EQ(setting(input_2, 28, 0)["max"], 24);
    EXPECT_EQ(setting(input_2, 28, 1)["max"], 12);
    EXPECT_EQ(setting(input_2, 34)["max"], 42);
    EXPECT_EQ(input_2["requires"][0]["then"]["index"], 0x8041);
    EXPECT_EQ(input_2["requires"][0]["when"][0]["index"], 0x8041);
    EXPECT_EQ(settings.value()["device"][0]["index"], 0x8020);
}

/// @brief it should give each 6xSTG input the entries that read it in the unbuffered
/// and buffered PDO layouts.
TEST(ReadSettings, Lists6xSTGInputEntriesInEveryLayout) {
    const auto settings = ASSERT_NIL_P(read_settings(DEWESOFT, IOLITE_6XSTG, read_stg));
    ASSERT_TRUE(settings.has_value());
    const auto &entries = settings.value()["channels"][1]["entries"];
    std::vector<std::pair<int, int>> addresses;
    for (const auto &e: entries)
        addresses.emplace_back(e["index"].get<int>(), e["sub_index"].get<int>());
    std::vector<std::pair<int, int>> want{{0x6030, 2}};
    for (int k = 1; k <= 8; k++)
        want.emplace_back(0x6041, k);
    EXPECT_EQ(addresses, want);
}

/// @brief it should return no settings for a product that the catalog does not
/// describe.
TEST(ReadSettings, ReturnsNoneForAnUndescribedProduct) {
    const auto settings = ASSERT_NIL_P(read_settings(DEWESOFT, 0xFFFF, read_stg));
    EXPECT_FALSE(settings.has_value());
}

/// @brief it should return the error of a failed read of a card table.
TEST(ReadSettings, ReturnsTheErrorOfAFailedRead) {
    const coe::Read missing = [](const uint16_t index, const uint8_t sub_index) {
        return std::pair{uint32_t{0}, coe::read_error(index, sub_index, 0x06020000)};
    };
    ASSERT_OCCURRED_AS_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, missing),
        coe::OBJECT_MISSING
    );
}

/// @brief it should reject a card table that counts more options than its object
/// holds.
TEST(ReadSettings, RejectsAnOptionCountPastTheObject) {
    const auto err = read_settings(DEWESOFT, IOLITE_6XSTG, counting_ranges(0xFFFFFFFF))
                         .second;
    ASSERT_MATCHES(err, errors::SDO_READ_ERROR);
    EXPECT_EQ(
        err.data,
        "0x8050:03 counts 4294967295 options. The count must be from 1 to 252"
    );
}

/// @brief it should reject a card table that counts no options.
TEST(ReadSettings, RejectsAnOptionCountOfZero) {
    const auto err = read_settings(DEWESOFT, IOLITE_6XSTG, counting_ranges(0)).second;
    ASSERT_MATCHES(err, errors::SDO_READ_ERROR);
    EXPECT_EQ(err.data, "0x8050:03 counts 0 options. The count must be from 1 to 252");
}
}
