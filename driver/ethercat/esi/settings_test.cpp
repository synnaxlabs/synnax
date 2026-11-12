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
using testutil::IOLITE_DIO;

/// @brief the product code of the IOLITEi 1xSTG, a card with one EtherCAT bus.
constexpr uint32_t IOLITEI_1XSTG = 0x9E;
/// @brief the revision of an IOLITE 6xSTG in its ESI.
constexpr uint32_t STG_REVISION = 0x00020001;

/// @brief returns a read of a standalone 6xSTG in which input 1 counts count ranges.
coe::Read counting_ranges(const uint32_t count) {
    auto objects = testutil::stg_objects(0);
    objects[{0x8050, 3}] = count;
    return testutil::read_of(std::move(objects));
}

/// @brief the sync modes that the SII of the 6xSTG lists.
const std::vector<dc::OpMode> STG_OP_MODES{
    {.name = "SM", .assign_activate = 0x0000},
    {.name = "DC", .assign_activate = 0x0300},
};

/// @brief returns a read of a 6xSTG that holds mode in ECAT0 App (0x8020:09) and
/// power_on in PowerONdefault (0x8020:02).
coe::Read stg(const uint32_t mode, const uint32_t power_on = 1) {
    return testutil::read_of(testutil::stg_objects(mode, power_on));
}

/// @brief returns a read of a standalone 6xSTG that aborts with abort_code at
/// index:sub_index.
coe::Read
aborting_at(const uint16_t index, const uint8_t sub_index, const uint32_t abort_code) {
    return [read = stg(0),
            index,
            sub_index,
            abort_code](const uint16_t i, const uint8_t s) {
        if (i == index && s == sub_index)
            return std::pair{uint32_t{0}, coe::read_error(i, s, abort_code)};
        return read(i, s);
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
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, STG_OP_MODES, stg(0))
    );
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

/// @brief it should give each setting of a 6xSTG input the value that the card holds.
TEST(ReadSettings, Gives6xSTGInputSettingsTheValuesOfTheCard) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, STG_OP_MODES, stg(0))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &input_2 = settings.value()["channels"][1];
    EXPECT_EQ(setting(input_2, 1)["value"], 6);
    EXPECT_EQ(setting(input_2, 3)["value"], 2);
    EXPECT_EQ(setting(input_2, 28)["value"], 5.0);
    EXPECT_EQ(setting(input_2, 34)["value"], 0.0);
}

/// @brief it should give each setting of the whole 6xSTG the value that the card holds.
TEST(ReadSettings, Gives6xSTGDeviceSettingsTheValuesOfTheCard) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, STG_OP_MODES, stg(0))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &sample_rate = settings.value()["device"][0];
    EXPECT_EQ(sample_rate["sub_index"], 1);
    EXPECT_EQ(sample_rate["value"], 12);
}

/// @brief it should leave out a setting at a sub-index that the card does not have,
/// with the rule that names it, and keep the other settings and the mode.
TEST(ReadSettings, LeavesOutASettingAtASubIndexTheCardDoesNotHave) {
    const auto [settings, err] = read_settings(
        DEWESOFT,
        IOLITE_6XSTG,
        STG_REVISION,
        STG_OP_MODES,
        aborting_at(0x8041, 28, 0x06090011)
    );
    ASSERT_MATCHES(err, coe::SUBINDEX_MISSING);
    EXPECT_EQ(err.data, "failed to read 0x8041:1C: Subindex does not exist");
    ASSERT_TRUE(settings.has_value());
    const auto &channels = settings.value()["channels"];
    ASSERT_EQ(channels.size(), 6);
    EXPECT_THROW(setting(channels[1], 28), std::runtime_error);
    EXPECT_EQ(setting(channels[1], 3)["value"], 2);
    EXPECT_EQ(setting(channels[0], 28)["value"], 5.0);
    ASSERT_EQ(channels[1]["requires"].size(), 1);
    EXPECT_EQ(channels[1]["requires"][0]["then"]["sub_index"], 34);
    EXPECT_EQ(channels[0]["requires"].size(), 2);
    EXPECT_TRUE(settings.value().contains("mode"));
}

/// @brief it should leave out each setting and rule that depends on a setting that the
/// card cannot read.
TEST(ReadSettings, LeavesOutTheSettingsThatDependOnAnUnreadSetting) {
    auto objects = testutil::stg_objects(0);
    objects.erase({0x8040, 1});
    const auto [settings, err] = read_settings(
        DEWESOFT,
        IOLITE_6XSTG,
        STG_REVISION,
        STG_OP_MODES,
        testutil::read_of(std::move(objects))
    );
    ASSERT_MATCHES(err, coe::OBJECT_MISSING);
    ASSERT_TRUE(settings.has_value());
    const auto &channels = settings.value()["channels"];
    EXPECT_EQ(
        names(channels[0]["settings"]),
        (std::vector<std::string>{
            "Low-pass filter",
            "Coupling",
            "Excitation",
            "Excitation polarity",
            "Excitation voltage",
            "Excitation voltage",
            "Excitation current",
        })
    );
    EXPECT_TRUE(channels[0]["requires"].empty());
    EXPECT_EQ(channels[1]["settings"].size(), 13);
    EXPECT_EQ(channels[1]["requires"].size(), 2);
}

/// @brief it should leave out a setting whose options the card cannot give, and keep
/// the other settings at its address.
TEST(ReadSettings, LeavesOutASettingWhoseOptionsTheCardCannotGive) {
    const auto [settings, err] = read_settings(
        DEWESOFT,
        IOLITE_6XSTG,
        STG_REVISION,
        STG_OP_MODES,
        counting_ranges(0)
    );
    ASSERT_MATCHES(err, errors::SDO_READ_ERROR);
    ASSERT_TRUE(settings.has_value());
    const auto &channels = settings.value()["channels"];
    EXPECT_EQ(setting(channels[0], 3)["visible_when"][0]["values"][0], 1);
    EXPECT_THROW(setting(channels[0], 3, 1), std::runtime_error);
    EXPECT_NO_THROW(setting(channels[1], 3, 1));
}

/// @brief it should give each 6xSTG input the entries that read it in the unbuffered
/// and buffered PDO layouts.
TEST(ReadSettings, Lists6xSTGInputEntriesInEveryLayout) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, STG_OP_MODES, stg(0))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &entries = settings.value()["channels"][1]["entries"];
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

/// @brief it should give each SIRIUS input the calibration objects of its channel.
TEST(ReadSettings, GivesEachSIRIUSInputTheCalibrationOfItsChannel) {
    constexpr uint32_t SIRIUS = 0x66;
    const auto settings = ASSERT_NIL_P(read_settings(
        DEWESOFT,
        SIRIUS,
        0x02000001,
        {},
        testutil::read_of({{{0x8020, 7}, 2}})
    ));
    ASSERT_TRUE(settings.has_value());
    const auto &channels = settings.value()["channels"];
    ASSERT_EQ(channels.size(), 16);
    EXPECT_EQ(
        channels[3]["entries"],
        x::json::json::parse(R"([{"index": 24624, "sub_index": 4, "calibration": {
            "data_type": "int32", "index": 32835, "gain_sub_index": 21,
            "offset_sub_index": 20, "divisor": 1}}])")
    );
}

/// @brief it should give a standalone 6xSTG the DC sync mode that its mode needs and
/// nothing to switch.
TEST(ReadSettings, Gives6xSTGInStandaloneModeItsSyncMode) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, STG_OP_MODES, stg(0))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &mode = settings.value()["mode"];
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

/// @brief it should name the object that holds the mode of a 6xSTG.
TEST(ReadSettings, NamesTheModeObjectOf6xSTG) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, STG_OP_MODES, stg(0))
    );
    ASSERT_TRUE(settings.has_value());
    EXPECT_EQ(settings.value()["mode"]["index"], 0x8020);
    EXPECT_EQ(settings.value()["mode"]["sub_index"], 9);
}

/// @brief it should mark a 6xSTG that DewesoftX also sets up as shared, and switch it
/// to standalone with a write that saves the mode.
TEST(ReadSettings, Switches6xSTGInDualModeToStandaloneAndSaves) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, STG_OP_MODES, stg(1, 1))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &mode = settings.value()["mode"];
    EXPECT_EQ(mode["value"], 1);
    EXPECT_TRUE(mode["shared"].get<bool>());
    EXPECT_EQ(mode["shared_with"], "DewesoftX");
    EXPECT_FALSE(mode.contains("only_read_by"));
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

/// @brief it should put a 6xSTG on bus 1 or bus 2 by the bus bit of its revision.
TEST(ReadSettings, Puts6xSTGOnTheBusOfItsRevision) {
    for (const auto &[revision, bus]:
         std::vector<std::pair<uint32_t, int>>{{0x00011001, 2}, {0x00010001, 1}}) {
        const auto settings = ASSERT_NIL_P(
            read_settings(DEWESOFT, IOLITE_6XSTG, revision, STG_OP_MODES, stg(1))
        );
        ASSERT_TRUE(settings.has_value());
        EXPECT_EQ(settings.value()["mode"]["bus"], bus);
    }
}

/// @brief it should name DewesoftX as the only reader of a card with one bus in the DC
/// buffered mode of DewesoftX.
TEST(ReadSettings, NamesDewesoftXAsTheOnlyReaderOfASingleBusCard) {
    const auto buffered = testutil::read_of({{{0x8020, 7}, 1}, {{0x8020, 2}, 0}});
    const auto [settings, err] = read_settings(
        DEWESOFT,
        IOLITEI_1XSTG,
        0x00100002,
        {},
        buffered
    );
    ASSERT_MATCHES(err, coe::OBJECT_MISSING);
    ASSERT_TRUE(settings.has_value());
    const auto &mode = settings.value()["mode"];
    EXPECT_TRUE(mode["shared"].get<bool>());
    EXPECT_EQ(mode["only_read_by"], "DewesoftX");
    EXPECT_FALSE(mode.contains("shared_with"));
    EXPECT_FALSE(mode.contains("bus"));
}

/// @brief it should not add the save write for a 6xSTG that saves each change.
TEST(ReadSettings, Switches6xSTGThatSavesEachChangeWithoutTheSaveWrite) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, STG_OP_MODES, stg(1, 0))
    );
    ASSERT_TRUE(settings.has_value());
    const auto &change = settings.value()["mode"]["switch"];
    ASSERT_EQ(change.size(), 1);
    EXPECT_EQ(change[0]["sub_index"], 9);
}

/// @brief it should give a 6xSTG in SM mode no sync mode when the SII lists no mode
/// with the catalog's assign activate value.
TEST(ReadSettings, OmitsTheSyncModeThatTheSIIDoesNotList) {
    const std::vector<dc::OpMode> sm_only{{.name = "SM", .assign_activate = 0}};
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, sm_only, stg(0))
    );
    ASSERT_TRUE(settings.has_value());
    EXPECT_FALSE(settings.value()["mode"].contains("op_mode"));
}

/// @brief it should return no settings for a product that the catalog does not
/// describe.
TEST(ReadSettings, ReturnsNoneForAnUndescribedProduct) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, 0xFFFF, 0, STG_OP_MODES, stg(0))
    );
    EXPECT_FALSE(settings.has_value());
}

/// @brief it should select the description of a product that matches the revision of
/// the slave.
TEST(ReadSettings, SelectsTheDescriptionOfTheRevision) {
    const auto read = testutil::read_of({{{0x8020, 1}, 0}, {{0x8020, 9}, 0}});
    const auto outputs = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_DIO, 0x00051000, {}, read)
    );
    ASSERT_TRUE(outputs.has_value());
    ASSERT_EQ(outputs.value()["device"].size(), 1);
    EXPECT_EQ(outputs.value()["device"][0]["name"], "Outputs on connection loss");
    const auto inputs = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_DIO, 0x00040000, {}, read)
    );
    ASSERT_TRUE(inputs.has_value());
    EXPECT_TRUE(inputs.value()["device"].empty());
    EXPECT_EQ(inputs.value()["mode"]["index"], 0x8020);
}

/// @brief it should return no settings for a revision that no description of the
/// product lists.
TEST(ReadSettings, ReturnsNoneForAnUndescribedRevision) {
    const auto settings = ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_DIO, 0x00010000, {}, testutil::read_of({}))
    );
    EXPECT_FALSE(settings.has_value());
}

/// @brief it should keep the other settings of a card that does not have the mode
/// object.
TEST(ReadSettings, KeepsTheOtherSettingsWhenTheCardHasNoModeObject) {
    auto objects = testutil::stg_objects(0);
    objects.erase({0x8020, 9});
    const auto [settings, err] = read_settings(
        DEWESOFT,
        IOLITE_6XSTG,
        STG_REVISION,
        STG_OP_MODES,
        testutil::read_of(std::move(objects))
    );
    ASSERT_MATCHES(err, coe::OBJECT_MISSING);
    EXPECT_EQ(
        err.data,
        "failed to read 0x8020:09: Object does not exist in the object dictionary"
    );
    ASSERT_TRUE(settings.has_value());
    EXPECT_FALSE(settings.value().contains("mode"));
    EXPECT_EQ(settings.value()["channels"].size(), 6);
}

/// @brief it should return no settings, and read nothing more, after a read that gets
/// no answer.
TEST(ReadSettings, StopsAtAReadThatGetsNoAnswer) {
    const auto read = aborting_at(0x8040, 1, 0);
    bool failed = false;
    int reads_after = 0;
    const coe::Read counted = [&](const uint16_t index, const uint8_t sub_index) {
        if (failed) reads_after++;
        auto result = read(index, sub_index);
        failed |= static_cast<bool>(result.second);
        return result;
    };
    const auto [settings, err] = read_settings(
        DEWESOFT,
        IOLITE_6XSTG,
        STG_REVISION,
        STG_OP_MODES,
        counted
    );
    ASSERT_MATCHES(err, errors::SDO_READ_ERROR);
    EXPECT_EQ(err.data, "failed to read 0x8040:01: no response");
    EXPECT_FALSE(settings.has_value());
    EXPECT_EQ(reads_after, 0);
}

/// @brief it should return no settings when the read of the mode gets no answer.
TEST(ReadSettings, ReturnsNoSettingsWhenTheModeReadGetsNoAnswer) {
    const auto [settings, err] = read_settings(
        DEWESOFT,
        IOLITE_6XSTG,
        STG_REVISION,
        STG_OP_MODES,
        aborting_at(0x8020, 9, 0)
    );
    ASSERT_MATCHES(err, errors::SDO_READ_ERROR);
    EXPECT_FALSE(err.matches(coe::OBJECT_MISSING));
    EXPECT_EQ(err.data, "failed to read 0x8020:09: no response");
    EXPECT_FALSE(settings.has_value());
}

/// @brief it should return the error of a failed read of a card table.
TEST(ReadSettings, ReturnsTheErrorOfAFailedRead) {
    const coe::Read missing = [](const uint16_t index, const uint8_t sub_index) {
        return std::pair{uint32_t{0}, coe::read_error(index, sub_index, 0x06020000)};
    };
    ASSERT_OCCURRED_AS_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, STG_OP_MODES, missing),
        coe::OBJECT_MISSING
    );
}

/// @brief it should reject a card table that counts more options than its object holds.
TEST(ReadSettings, RejectsAnOptionCountPastTheObject) {
    const auto err = read_settings(
                         DEWESOFT,
                         IOLITE_6XSTG,
                         STG_REVISION,
                         STG_OP_MODES,
                         counting_ranges(0xFFFFFFFF)
    )
                         .second;
    ASSERT_MATCHES(err, errors::SDO_READ_ERROR);
    EXPECT_EQ(
        err.data,
        "0x8050:03 counts 4294967295 options. The count must be from 1 to 252"
    );
}

/// @brief it should read each object of the card once, even when two settings share it.
TEST(ReadSettings, ReadsEachObjectOnce) {
    std::map<std::pair<uint16_t, uint8_t>, int> reads;
    const auto read = stg(0);
    const coe::Read counted = [&](const uint16_t index, const uint8_t sub_index) {
        reads[{index, sub_index}]++;
        return read(index, sub_index);
    };
    ASSERT_NIL_P(
        read_settings(DEWESOFT, IOLITE_6XSTG, STG_REVISION, STG_OP_MODES, counted)
    );
    for (const auto &[address, count]: reads)
        EXPECT_EQ(count, 1) << std::hex << address.first << ":" << +address.second;
}

/// @brief it should reject a card table that counts no options.
TEST(ReadSettings, RejectsAnOptionCountOfZero) {
    const auto err = read_settings(
                         DEWESOFT,
                         IOLITE_6XSTG,
                         STG_REVISION,
                         STG_OP_MODES,
                         counting_ranges(0)
    )
                         .second;
    ASSERT_MATCHES(err, errors::SDO_READ_ERROR);
    EXPECT_EQ(err.data, "0x8050:03 counts 0 options. The count must be from 1 to 252");
}
}
