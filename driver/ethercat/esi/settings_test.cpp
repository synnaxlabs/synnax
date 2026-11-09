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

#include "driver/ethercat/esi/settings.h"

namespace driver::ethercat::esi {
namespace {
constexpr uint32_t DEWESOFT = 0xDEBE50F7;
constexpr uint32_t IOLITE_6XSTG = 252;

/// @brief the range, filter, and excitation tables of input 1 of the 6xSTG on the
/// station. Each other input holds the same tables at its own indices.
const std::map<std::pair<uint16_t, uint8_t>, uint32_t> STG_TABLES{
    {{0x8050, 3}, 5},
    {{0x8050, 4}, 500},
    {{0x8050, 5}, 100},
    {{0x8050, 6}, 30},
    {{0x8050, 7}, 10},
    {{0x8050, 8}, 1},
    {{0x8060, 1}, 2},
    {{0x8060, 2}, 10000},
    {{0x8060, 3}, 500},
    {{0x8070, 2}, 240},
    {{0x8070, 3}, 42},
    {{0x8070, 4}, 120},
};

std::pair<uint32_t, x::errors::Error>
read_stg(const uint16_t index, const uint8_t sub_index) {
    const auto it = STG_TABLES.find({static_cast<uint16_t>(index & 0xFFF8), sub_index});
    if (it == STG_TABLES.end() || (index & 0x7) > 5)
        return {0, coe::read_error(index, sub_index, 0x06020000)};
    return {it->second, x::errors::NIL};
}

/// @brief the sync modes that the SII of the 6xSTG lists.
const std::vector<dc::OpMode> STG_OP_MODES{
    {.name = "SM", .assign_activate = 0x0000},
    {.name = "DC", .assign_activate = 0x0300},
};

/// @brief returns a read of a 6xSTG that holds mode in ECAT0 App (0x8020:09) and
/// power_on in PowerONdefault (0x8020:02).
coe::Read stg(const uint32_t mode, const uint32_t power_on = 1) {
    return [mode, power_on](const uint16_t index, const uint8_t sub_index) {
        if (index == 0x8020 && sub_index == 9) return std::pair{mode, x::errors::NIL};
        if (index == 0x8020 && sub_index == 2)
            return std::pair{power_on, x::errors::NIL};
        return read_stg(index, sub_index);
    };
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
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_OP_MODES, stg(0))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &channels = (*settings)["channels"];
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
    EXPECT_EQ((*settings)["device"][0]["index"], 0x8020);
}

/// @brief it should give each 6xSTG input the entries that read it in the unbuffered
/// and buffered PDO layouts.
TEST(ReadSettings, Lists6xSTGInputEntriesInEveryLayout) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_OP_MODES, stg(0))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &entries = (*settings)["channels"][1]["entries"];
    std::vector<std::pair<int, int>> addresses;
    for (const auto &e: entries)
        addresses.emplace_back(e["index"].get<int>(), e["sub_index"].get<int>());
    std::vector<std::pair<int, int>> want{{0x6030, 2}};
    for (int k = 1; k <= 8; k++)
        want.emplace_back(0x6041, k);
    EXPECT_EQ(addresses, want);
    EXPECT_FALSE(entries[0].contains("only_read_by"));
    EXPECT_EQ(entries[1]["only_read_by"], "DewesoftX");
}

/// @brief it should give a standalone 6xSTG the DC sync mode that its mode needs and
/// nothing to switch.
TEST(ReadSettings, Gives6xSTGInStandaloneModeItsSyncMode) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_OP_MODES, stg(0))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &mode = (*settings)["mode"];
    EXPECT_EQ(mode["value"], 0);
    EXPECT_FALSE(mode["shared"].get<bool>());
    EXPECT_EQ(
        mode["op_mode"],
        (x::json::json{
            {"index", 1},
            {"assign_activate", 0x0300},
            {"sync0_cycle", 1000},
            {"sync1_cycle", 1000000000},
        })
    );
    EXPECT_TRUE(mode["switch"].empty());
}

/// @brief it should mark a 6xSTG that DewesoftX also sets up as shared, and switch it
/// to standalone with a write that saves the mode.
TEST(ReadSettings, Switches6xSTGInDualModeToStandaloneAndSaves) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_OP_MODES, stg(1, 1))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &mode = (*settings)["mode"];
    EXPECT_EQ(mode["value"], 1);
    EXPECT_TRUE(mode["shared"].get<bool>());
    EXPECT_FALSE(mode.contains("op_mode"));
    EXPECT_EQ(
        mode["switch"],
        (x::json::json{
            {{"index", 0x8020},
             {"sub_index", 9},
             {"data_type", "uint16"},
             {"value", 0}},
            {{"index", 0x8020},
             {"sub_index", 2},
             {"data_type", "uint16"},
             {"value", 2}},
        })
    );
}

/// @brief it should not add the save write for a 6xSTG that saves each change.
TEST(ReadSettings, Switches6xSTGThatSavesEachChangeWithoutTheSaveWrite) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_OP_MODES, stg(1, 0))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &change = (*settings)["mode"]["switch"];
    ASSERT_EQ(change.size(), 1);
    EXPECT_EQ(change[0]["sub_index"], 9);
}

/// @brief it should give a 6xSTG in SM mode no sync mode when the SII lists no mode
/// with the catalog's assign activate value.
TEST(ReadSettings, OmitsTheSyncModeThatTheSIIDoesNotList) {
    const std::vector<dc::OpMode> sm_only{{.name = "SM", .assign_activate = 0}};
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, sm_only, stg(0))
    );
    ASSERT_TRUE(settings.has_value());
    EXPECT_FALSE((*settings)["mode"].contains("op_mode"));
}

/// @brief it should return no settings for a product that the catalog does not
/// describe.
TEST(ReadSettings, ReturnsNoneForAnUndescribedProduct) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, 0xFFFF, STG_OP_MODES, stg(0))
    );
    EXPECT_FALSE(settings.has_value());
}

/// @brief it should return the error of a failed read of a card table.
TEST(ReadSettings, ReturnsTheErrorOfAFailedRead) {
    const coe::Read missing = [](const uint16_t index, const uint8_t sub_index) {
        return std::pair{uint32_t{0}, coe::read_error(index, sub_index, 0x06020000)};
    };
    ASSERT_OCCURRED_AS_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_OP_MODES, missing),
        coe::OBJECT_MISSING
    );
}
}
