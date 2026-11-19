// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>
#include <string>
#include <utility>

#include "gtest/gtest.h"

#include "client/cpp/ethercat/json.gen.h"

#include "driver/ethercat/channel/channel.h"

namespace driver::ethercat::channel {
namespace {
/// @brief returns a slave with eight 1-bit "Input" entries at 0x6000:01 through
/// 0x6070:01, one per PDO, and eight "Output" entries at 0x7000:01 through 0x7070:01.
slave::Properties repeated_name_slave() {
    slave::Properties slave{.position = 3, .name = "EL1008"};
    for (uint16_t i = 0; i < 8; i++) {
        const auto pdo_name = "Channel " + std::to_string(i + 1);
        slave.input_pdos.push_back(
            {.pdo_index = static_cast<uint16_t>(0x1A00 + i),
             .index = static_cast<uint16_t>(0x6000 + 0x10 * i),
             .sub_index = 1,
             .bit_length = 1,
             .is_input = true,
             .name = "Input",
             .data_type = x::telem::UINT8_T,
             .pdo_name = pdo_name}
        );
        slave.output_pdos.push_back(
            {.pdo_index = static_cast<uint16_t>(0x1600 + i),
             .index = static_cast<uint16_t>(0x7000 + 0x10 * i),
             .sub_index = 1,
             .bit_length = 1,
             .is_input = false,
             .name = "Output",
             .data_type = x::telem::UINT8_T,
             .pdo_name = pdo_name}
        );
    }
    return slave;
}

/// @brief returns a slave with a 1-bit status entry, a 7-bit padding entry, and a
/// 16-bit value entry, mapped as a Beckhoff analog input maps them.
slave::Properties padded_slave() {
    return slave::Properties{
        .position = 1,
        .name = "EL3102",
        .input_pdos = {
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 1,
             .bit_length = 1,
             .is_input = true,
             .name = "Underrange",
             .data_type = x::telem::UINT8_T},
            {.pdo_index = 0x1A00,
             .index = 0,
             .sub_index = 0,
             .bit_length = 7,
             .is_input = true,
             .name = "",
             .data_type = x::telem::UNKNOWN_T},
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 0x11,
             .bit_length = 16,
             .is_input = true,
             .name = "Value",
             .data_type = x::telem::INT16_T},
        },
    };
}

slave::Properties wide_output_slave() {
    return slave::Properties{
        .position = 2,
        .name = "EL2252",
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 1,
             .bit_length = 64,
             .is_input = false,
             .name = "Count",
             .data_type = x::telem::INT64_T},
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 2,
             .bit_length = 32,
             .is_input = false,
             .name = "Setpoint",
             .data_type = x::telem::FLOAT32_T},
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 3,
             .bit_length = 24,
             .is_input = false,
             .name = "Position",
             .data_type = x::telem::UINT32_T},
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 4,
             .bit_length = 20,
             .is_input = false,
             .name = "Torque",
             .data_type = x::telem::UINT32_T},
        },
    };
}

nlohmann::json read_entry(
    const uint16_t index,
    const uint8_t sub_index,
    const std::string &data_type = "uint8"
) {
    return {
        {"type", "entry"},
        {"device", "dev-001"},
        {"channel", 42},
        {"disabled", false},
        {"index", index},
        {"sub_index", sub_index},
        {"data_type", data_type},
    };
}

nlohmann::json write_entry(
    const uint16_t index,
    const uint8_t sub_index,
    const std::string &data_type = "uint8"
) {
    return {
        {"type", "entry"},
        {"device", "dev-001"},
        {"cmd_channel", 7},
        {"state_channel", 8},
        {"disabled", false},
        {"index", index},
        {"sub_index", sub_index},
        {"data_type", data_type},
    };
}

nlohmann::json read_unresolved(const std::string &pdo) {
    return {
        {"type", "unresolved"},
        {"device", "dev-001"},
        {"channel", 42},
        {"disabled", false},
        {"pdo", pdo},
    };
}

nlohmann::json write_unresolved(const std::string &pdo) {
    return {
        {"type", "unresolved"},
        {"device", "dev-001"},
        {"cmd_channel", 7},
        {"state_channel", 8},
        {"disabled", false},
        {"pdo", pdo},
    };
}

/// @brief returns the config of a calibrated read channel at the entry.
nlohmann::json calibrated_entry(
    const uint16_t index,
    const uint8_t sub_index,
    const std::string &type
) {
    auto j = read_entry(index, sub_index, type);
    j["calibrated"] = true;
    return j;
}

/// @brief returns a slave whose catalog calibrates the input 0x6030:01 with the gain at
/// 0x8040:15 and the offset at 0x8040:14. The slave sends samples of data_type there.
slave::Properties calibrated_slave(
    const x::telem::DataType &data_type,
    const std::string &calibration_type = "int32"
) {
    slave::Properties slave{.position = 2, .name = "SIRIUS"};
    slave.input_pdos.push_back(
        {.pdo_index = 0x1A30,
         .index = 0x6030,
         .sub_index = 1,
         .bit_length = 32,
         .is_input = true,
         .name = "CH1",
         .data_type = data_type}
    );
    slave.settings = x::json::json{
        {"channels",
         {{{"entries",
            {{{"index", 0x6030},
              {"sub_index", 1},
              {"calibration",
               {{"data_type", calibration_type},
                {"index", 0x8040},
                {"gain_sub_index", 0x15},
                {"offset_sub_index", 0x14},
                {"divisor", 256},
                {"raw_offset", true}}}}}}}}},
    };
    return slave;
}

std::unique_ptr<Input> input(const slave::Properties &slave, x::json::Parser &p) {
    return make_input(::synnax::ethercat::parse_read_channel(p), slave, p);
}

std::unique_ptr<Output> output(const slave::Properties &slave, x::json::Parser &p) {
    return make_output(::synnax::ethercat::parse_write_channel(p), slave, p);
}

/// @brief returns the path and message of the parser's only field error.
std::pair<std::string, std::string> only_error(const x::json::Parser &parser) {
    const auto errors = parser.error_json()["errors"];
    EXPECT_EQ(errors.size(), 1) << errors.dump();
    if (errors.empty()) return {};
    return {
        errors[0]["path"].get<std::string>(),
        errors[0]["message"].get<std::string>()
    };
}
}

/// @brief it should bind an input channel to the entry at its address, with the bit
/// length from the slave.
TEST(Input, BindsAUniqueEntryWithBitLengthFromTheSlave) {
    const auto slave = padded_slave();
    const auto j = read_entry(0x6000, 0x11, "uint16");
    auto parser = x::json::Parser(j);
    const auto ch = input(slave, parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_TRUE(ch->entry.is_input);
    ASSERT_EQ(ch->synnax_key, 42);
    ASSERT_EQ(ch->entry.slave_position, 1);
    ASSERT_EQ(ch->entry.index, 0x6000);
    ASSERT_EQ(ch->entry.sub_index, 0x11);
    ASSERT_EQ(ch->entry.bit_length, 16);
    ASSERT_EQ(ch->entry.data_type, x::telem::UINT16_T);
}

/// @brief it should give an integer input the calibration of its catalog, and decode
/// its samples in the data type of the calibration.
TEST(Input, TakesTheCalibrationOfTheCatalogForIntegerSamples) {
    const auto slave = calibrated_slave(x::telem::UINT32_T);
    auto parser = x::json::Parser(calibrated_entry(0x6030, 1, "uint32"));
    const auto ch = input(slave, parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_TRUE(ch->calibration.has_value());
    const auto &calibration = ch->calibration.value();
    EXPECT_EQ(calibration.index, 0x8040);
    EXPECT_EQ(calibration.gain_sub_index, 0x15);
    EXPECT_EQ(calibration.offset_sub_index, 0x14);
    EXPECT_EQ(ch->entry.data_type, x::telem::INT32_T);
}

/// @brief it should not calibrate an input that sends float samples.
TEST(Input, LeavesFloatSamplesUncalibrated) {
    const auto slave = calibrated_slave(x::telem::FLOAT32_T);
    auto parser = x::json::Parser(calibrated_entry(0x6030, 1, "float32"));
    const auto ch = input(slave, parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    EXPECT_FALSE(ch->calibration.has_value());
    EXPECT_EQ(ch->entry.data_type, x::telem::FLOAT32_T);
}

/// @brief it should leave the samples of a channel raw when the channel is not
/// calibrated, as a task from before calibrations is.
TEST(Input, LeavesAnUncalibratedChannelRaw) {
    const auto slave = calibrated_slave(x::telem::UINT32_T);
    auto parser = x::json::Parser(read_entry(0x6030, 1, "uint32"));
    const auto ch = input(slave, parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    EXPECT_FALSE(ch->calibration.has_value());
    EXPECT_EQ(ch->entry.data_type, x::telem::UINT32_T);
}

/// @brief it should report a calibrated channel whose entry the catalog does not
/// calibrate.
TEST(Input, ReportsACalibratedChannelThatTheCatalogDoesNotCalibrate) {
    auto slave = calibrated_slave(x::telem::INT32_T);
    slave.settings.reset();
    auto parser = x::json::Parser(calibrated_entry(0x6030, 1, "int32"));
    const auto ch = input(slave, parser);
    const auto [path, message] = only_error(parser);
    EXPECT_EQ(path, "calibrated");
    EXPECT_EQ(
        message,
        "the catalog gives no calibration for entry 0x6030:01 on module SIRIUS"
    );
    EXPECT_EQ(ch, nullptr);
}

/// @brief it should report a catalog calibration whose data type does not fit the
/// entry.
TEST(Input, ReportsACalibrationThatDoesNotFitTheEntry) {
    const auto slave = calibrated_slave(x::telem::INT32_T, "int16");
    auto parser = x::json::Parser(calibrated_entry(0x6030, 1, "int32"));
    const auto ch = input(slave, parser);
    const auto [path, message] = only_error(parser);
    EXPECT_EQ(path, "index");
    EXPECT_EQ(
        message,
        "the catalog gives int16 samples for entry 0x6030:01 on module SIRIUS, but the "
        "entry holds 32 bits"
    );
    EXPECT_EQ(ch, nullptr);
}

/// @brief it should turn a gain and a raw offset into a line: (raw / 256 + offset) *
/// gain.
TEST(Calibration, ScalesARawOffsetByTheGain) {
    const Calibration c{.divisor = 256, .raw_offset = true};
    const auto [slope, offset] = c.line(2, 3);
    EXPECT_DOUBLE_EQ(slope, 2.0 / 256);
    EXPECT_DOUBLE_EQ(offset, 6);
}

/// @brief it should turn a gain and a physical offset into a line: raw * gain + offset.
TEST(Calibration, AddsAPhysicalOffsetAfterTheGain) {
    const Calibration c{};
    const auto [slope, offset] = c.line(0.5F, -1.25F);
    EXPECT_DOUBLE_EQ(slope, 0.5);
    EXPECT_DOUBLE_EQ(offset, -1.25);
}

/// @brief it should report a data type that is wider than the entry.
TEST(Input, ReportsADataTypeWiderThanTheEntry) {
    const auto slave = padded_slave();
    const auto j = read_entry(0x6000, 0x11, "float64");
    auto parser = x::json::Parser(j);
    const auto ch = input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "data_type");
    ASSERT_EQ(
        message,
        "data type float64 is 8 bytes but entry 0x6000:11 on module EL3102 holds 2 "
        "bytes. Pick a 2 byte type."
    );
    ASSERT_EQ(ch, nullptr);
}

/// @brief it should report a data type that is narrower than the entry.
TEST(Input, ReportsADataTypeNarrowerThanTheEntry) {
    const auto slave = padded_slave();
    const auto j = read_entry(0x6000, 0x11, "int8");
    auto parser = x::json::Parser(j);
    input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "data_type");
    ASSERT_EQ(
        message,
        "data type int8 is 1 byte but entry 0x6000:11 on module EL3102 holds 2 bytes. "
        "Pick a 2 byte type."
    );
}

/// @brief it should report a data type that is not a number type.
TEST(Input, ReportsADataTypeTheCyclicExchangeCannotCarry) {
    const auto slave = padded_slave();
    const auto j = read_entry(0x6000, 0x11, "timestamp");
    auto parser = x::json::Parser(j);
    const auto ch = input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "data_type");
    ASSERT_EQ(
        message,
        "data type timestamp cannot be exchanged with entry 0x6000:11 on module "
        "EL3102. Pick a 2 byte number type."
    );
    ASSERT_EQ(ch, nullptr);
}

/// @brief it should give no channel when the slave has no entry at the address.
TEST(Input, BindsNoEntryAtAnAddressTheSlaveDoesNotHave) {
    const auto slave = padded_slave();
    const auto j = read_entry(0x6010, 1);
    auto parser = x::json::Parser(j);
    const auto ch = input(slave, parser);
    ASSERT_FALSE(parser.ok());
    ASSERT_EQ(ch, nullptr);
}

/// @brief it should bind each entry of a slave whose entries all have the same name.
TEST(Input, BindsEveryEntryOfASlaveThatRepeatsOneName) {
    const auto slave = repeated_name_slave();
    std::set<std::pair<uint16_t, uint8_t>> addresses;
    for (uint16_t i = 0; i < 8; i++) {
        const auto j = read_entry(0x6000 + 0x10 * i, 1);
        auto parser = x::json::Parser(j);
        const auto ch = input(slave, parser);
        ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
        ASSERT_EQ(ch->entry.index, 0x6000 + 0x10 * i);
        ASSERT_EQ(ch->entry.sub_index, 1);
        ASSERT_EQ(ch->entry.bit_length, 1);
        addresses.emplace(ch->entry.index, ch->entry.sub_index);
    }
    ASSERT_EQ(addresses.size(), 8);
}

/// @brief it should report an address at which the slave has no input entry.
TEST(Input, ReportsAnAddressTheSlaveDoesNotHave) {
    const auto slave = padded_slave();
    const auto j = read_entry(0x6010, 1);
    auto parser = x::json::Parser(j);
    input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "index");
    ASSERT_EQ(message, "module EL3102 has no input entry at 0x6010:01");
}

/// @brief it should report the padding address as an address with no entry.
TEST(Input, ReportsAPaddingAddress) {
    const auto slave = padded_slave();
    const auto j = read_entry(0, 0);
    auto parser = x::json::Parser(j);
    input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "index");
    ASSERT_EQ(message, "module EL3102 has no input entry at 0x0000:00");
}

/// @brief it should report the address of an output entry as missing for an input
/// channel.
TEST(Input, ReportsAnOutputAddressAsMissing) {
    const auto slave = repeated_name_slave();
    const auto j = read_entry(0x7000, 1);
    auto parser = x::json::Parser(j);
    input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "index");
    ASSERT_EQ(message, "module EL1008 has no input entry at 0x7000:01");
}

/// @brief it should list each matching address when an unresolved name matches more
/// than one entry.
TEST(Input, ListsEveryAddressOfARepeatedNameWhenUnresolved) {
    const auto slave = repeated_name_slave();
    const auto j = read_unresolved("Input");
    auto parser = x::json::Parser(j);
    input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "pdo");
    ASSERT_EQ(
        message,
        "'Input' matches more than one input entry on module EL1008: 0x6000:01 "
        "(Channel 1), 0x6010:01 (Channel 2), 0x6020:01 (Channel 3), 0x6030:01 "
        "(Channel 4), 0x6040:01 (Channel 5), 0x6050:01 (Channel 6), 0x6060:01 "
        "(Channel 7), 0x6070:01 (Channel 8). Pick an entry."
    );
}

/// @brief it should report an unresolved name that matches no entry on the slave.
TEST(Input, ReportsAnUnresolvedNameTheSlaveDoesNotHave) {
    const auto slave = padded_slave();
    const auto j = read_unresolved("Analog Input Ch1");
    auto parser = x::json::Parser(j);
    input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "pdo");
    ASSERT_EQ(
        message,
        "'Analog Input Ch1' matches no input entry on module EL3102. Pick an entry."
    );
}

/// @brief it should name the address of the one entry that an unresolved name matches.
TEST(Input, NamesTheOneAddressOfAnUnresolvedName) {
    const auto slave = padded_slave();
    const auto j = read_unresolved("Value");
    auto parser = x::json::Parser(j);
    input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "pdo");
    ASSERT_EQ(
        message,
        "'Value' matches input entry 0x6000:11 on module EL3102. Pick an entry."
    );
}

/// @brief it should name an address only once when several PDOs map the same entry.
TEST(Input, NamesAnAddressThatSeveralPDOsRepeatOnce) {
    slave::Properties slave{.position = 2, .name = "EJ7342"};
    for (uint16_t i = 0; i < 4; i++)
        slave.input_pdos.push_back(
            {.pdo_index = static_cast<uint16_t>(0x1A00 + i),
             .index = 0x1C32,
             .sub_index = 0x20,
             .bit_length = 1,
             .is_input = true,
             .name = "Sync error",
             .data_type = x::telem::UINT8_T,
             .pdo_name = "Module " + std::to_string(i + 1)}
        );
    const auto j = read_unresolved("Sync error");
    auto parser = x::json::Parser(j);
    input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "pdo");
    ASSERT_EQ(
        message,
        "'Sync error' matches input entry 0x1C32:20 (Module 1) on module EJ7342. Pick "
        "an entry."
    );
}

/// @brief it should bind an output channel to the entry at its address, with the bit
/// length from the slave.
TEST(Output, BindsAnEntryWithBitLengthFromTheSlave) {
    const auto slave = repeated_name_slave();
    const auto j = write_entry(0x7030, 1);
    auto parser = x::json::Parser(j);
    const auto ch = output(slave, parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_FALSE(ch->entry.is_input);
    ASSERT_EQ(ch->command_key, 7);
    ASSERT_EQ(ch->state_key, 8);
    ASSERT_EQ(ch->entry.slave_position, 3);
    ASSERT_EQ(ch->entry.index, 0x7030);
    ASSERT_EQ(ch->entry.sub_index, 1);
    ASSERT_EQ(ch->entry.bit_length, 1);
    ASSERT_EQ(ch->entry.data_type, x::telem::UINT8_T);
}

/// @brief it should report the address of an input entry as missing for an output
/// channel.
TEST(Output, ReportsAnInputAddressAsMissing) {
    const auto slave = repeated_name_slave();
    const auto j = write_entry(0x6000, 1);
    auto parser = x::json::Parser(j);
    output(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "index");
    ASSERT_EQ(message, "module EL1008 has no output entry at 0x6000:01");
}

/// @brief it should report a data type that is wider than the entry.
TEST(Output, ReportsADataTypeWiderThanTheEntry) {
    const auto slave = repeated_name_slave();
    const auto j = write_entry(0x7000, 1, "float32");
    auto parser = x::json::Parser(j);
    const auto ch = output(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "data_type");
    ASSERT_EQ(
        message,
        "data type float32 is 4 bytes but entry 0x7000:01 on module EL1008 holds 1 "
        "byte. Pick a 1 byte type."
    );
    ASSERT_EQ(ch, nullptr);
}

/// @brief it should reject an int64 stop value of 2^53 or more, which a double may have
/// rounded.
TEST(Output, ReportsAnInt64StopValueThatADoubleRounds) {
    auto j = write_entry(0x7000, 1, "int64");
    j["stop_value"] = 9007199254740993;
    auto parser = x::json::Parser(j);
    output(wide_output_slave(), parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "stop_value");
    ASSERT_EQ(
        message,
        "stop value 9007199254740992 does not fit entry 0x7000:01 on module EL2252. "
        "Pick a whole number from -9007199254740991 to 9007199254740991."
    );
}

/// @brief it should accept the largest int64 stop value that a double holds exactly.
TEST(Output, AcceptsTheLargestExactInt64StopValue) {
    auto j = write_entry(0x7000, 1, "int64");
    j["stop_value"] = -9007199254740991;
    auto parser = x::json::Parser(j);
    const auto ch = output(wide_output_slave(), parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_EQ(ch->stop_value, -9007199254740991.0);
}

/// @brief it should accept a high stop value on a 1-bit int8 entry, which reads back
/// unsigned.
TEST(Output, AcceptsAHighStopValueOnASubByteSignedEntry) {
    const auto slave = repeated_name_slave();
    auto j = write_entry(0x7000, 1, "int8");
    j["stop_value"] = 1;
    auto parser = x::json::Parser(j);
    const auto ch = output(slave, parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_EQ(ch->stop_value, 1.0);
}

/// @brief it should reject a negative stop value on a 1-bit int8 entry, which reads
/// back unsigned.
TEST(Output, RejectsANegativeStopValueOnASubByteSignedEntry) {
    const auto slave = repeated_name_slave();
    auto j = write_entry(0x7000, 1, "int8");
    j["stop_value"] = -1;
    auto parser = x::json::Parser(j);
    output(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "stop_value");
    ASSERT_EQ(
        message,
        "stop value -1 does not fit entry 0x7000:01 on module EL1008. Pick a whole "
        "number from 0 to 1."
    );
}

/// @brief it should reject a fractional stop value on an integer entry.
TEST(Output, RejectsAFractionalStopValueOnAnIntegerEntry) {
    auto j = write_entry(0x7000, 3, "uint32");
    j["stop_value"] = 1.5;
    auto parser = x::json::Parser(j);
    output(wide_output_slave(), parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "stop_value");
    ASSERT_EQ(
        message,
        "stop value 1.5 does not fit entry 0x7000:03 on module EL2252. Pick a whole "
        "number from 0 to 16777215."
    );
}

/// @brief it should reject a stop value outside the finite range of a float32 entry.
TEST(Output, ReportsAFloat32StopValueOutsideItsRange) {
    auto j = write_entry(0x7000, 2, "float32");
    j["stop_value"] = 1e100;
    auto parser = x::json::Parser(j);
    output(wide_output_slave(), parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "stop_value");
    ASSERT_EQ(
        message,
        "stop value 1e+100 does not fit entry 0x7000:02 on module EL2252. Pick a value "
        "from -3.4028235e+38 to 3.4028235e+38."
    );
}

/// @brief it should accept the largest stop value that a 24-bit entry holds for a
/// float32 output channel.
TEST(Output, AcceptsAFloatTypeOnANarrowerEntry) {
    auto j = write_entry(0x7000, 3, "float32");
    j["stop_value"] = 16777215;
    auto parser = x::json::Parser(j);
    const auto ch = output(wide_output_slave(), parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_EQ(ch->entry.index, 0x7000);
    ASSERT_EQ(ch->stop_value, 16777215.0);
}

/// @brief it should reject a float32 output channel on a narrower entry that is not 24
/// bits.
TEST(Output, ReportsAFloatTypeOnANarrowerEntryThatIsNot24Bits) {
    const auto j = write_entry(0x7000, 4, "float32");
    auto parser = x::json::Parser(j);
    const auto ch = output(wide_output_slave(), parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "data_type");
    ASSERT_EQ(
        message,
        "data type float32 needs a 32 bit entry, but entry 0x7000:04 on module EL2252 "
        "holds 20 bits. Pick an integer type."
    );
    ASSERT_EQ(ch, nullptr);
}

/// @brief it should reject a stop value that a 24-bit entry cannot hold for a float32
/// output channel.
TEST(Output, ReportsAStopValueThatANarrowerEntryCannotHoldForAFloatType) {
    auto j = write_entry(0x7000, 3, "float32");
    j["stop_value"] = 16777216;
    auto parser = x::json::Parser(j);
    const auto ch = output(wide_output_slave(), parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "stop_value");
    ASSERT_EQ(
        message,
        "stop value 16777216 does not fit entry 0x7000:03 on module EL2252. Pick a "
        "whole number from 0 to 16777215."
    );
}

/// @brief it should list each matching address when an unresolved name matches more
/// than one entry.
TEST(Output, ListsEveryAddressOfARepeatedNameWhenUnresolved) {
    auto slave = repeated_name_slave();
    slave.output_pdos.resize(2);
    const auto j = write_unresolved("Output");
    auto parser = x::json::Parser(j);
    output(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "pdo");
    ASSERT_EQ(
        message,
        "'Output' matches more than one output entry on module EL1008: 0x7000:01 "
        "(Channel 1), 0x7010:01 (Channel 2). Pick an entry."
    );
}

/// @brief it should report an unresolved name that matches no entry on the slave.
TEST(Output, ReportsAnUnresolvedNameTheSlaveDoesNotHave) {
    const auto slave = repeated_name_slave();
    const auto j = write_unresolved("Analog Output Ch2");
    auto parser = x::json::Parser(j);
    output(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "pdo");
    ASSERT_EQ(
        message,
        "'Analog Output Ch2' matches no output entry on module EL1008. Pick an entry."
    );
}

/// @brief it should give an output no state channel when the config leaves it out.
TEST(Output, StateKeyDefaultsToZero) {
    const auto slave = repeated_name_slave();
    auto j = write_entry(0x7000, 1);
    j.erase("state_channel");
    auto parser = x::json::Parser(j);
    const auto ch = output(slave, parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_EQ(ch->state_key, 0);
}

/// @brief it should accept channels that use different entries.
TEST(Claims, AcceptsChannelsOnDistinctEntries) {
    const auto slave = repeated_name_slave();
    Claims claims;
    for (uint16_t i = 0; i < 8; i++) {
        const auto j = read_entry(0x6000 + 0x10 * i, 1);
        auto parser = x::json::Parser(j);
        const auto ch = input(slave, parser);
        claims.claim(ch->entry, slave, parser);
        ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    }
}

/// @brief it should report a second channel that uses an entry another channel already
/// uses.
TEST(Claims, ReportsASecondChannelOnOneEntry) {
    const auto slave = repeated_name_slave();
    Claims claims;
    const auto j = read_entry(0x6030, 1);
    auto first = x::json::Parser(j);
    claims.claim(input(slave, first)->entry, slave, first);
    ASSERT_TRUE(first.ok());
    auto second = x::json::Parser(j);
    claims.claim(input(slave, second)->entry, slave, second, "channels.1.");
    const auto [path, message] = only_error(second);
    ASSERT_EQ(path, "channels.1.index");
    ASSERT_EQ(message, "another channel already uses entry 0x6030:01 on module EL1008");
}

/// @brief it should accept channels at the same address on two different slaves.
TEST(Claims, AcceptsOneAddressOnTwoSlaves) {
    auto first_slave = repeated_name_slave();
    first_slave.position = 1;
    auto second_slave = repeated_name_slave();
    second_slave.position = 2;
    Claims claims;
    const auto j = read_entry(0x6000, 1);
    auto first = x::json::Parser(j);
    claims.claim(input(first_slave, first)->entry, first_slave, first);
    auto second = x::json::Parser(j);
    claims.claim(input(second_slave, second)->entry, second_slave, second);
    ASSERT_TRUE(first.ok()) << first.error_json().dump();
    ASSERT_TRUE(second.ok()) << second.error_json().dump();
}

/// @brief it should accept an input channel and an output channel at the same address.
TEST(Claims, SeparatesInputAndOutputEntriesAtOneAddress) {
    slave::Properties slave{
        .position = 1,
        .name = "Both",
        .input_pdos =
            {
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 1,
                 .bit_length = 8,
                 .is_input = true,
                 .name = "Value",
                 .data_type = x::telem::UINT8_T},
            },
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x6000,
             .sub_index = 1,
             .bit_length = 8,
             .is_input = false,
             .name = "Value",
             .data_type = x::telem::UINT8_T},
        },
    };
    Claims claims;
    const auto r = read_entry(0x6000, 1);
    auto read_parser = x::json::Parser(r);
    claims.claim(input(slave, read_parser)->entry, slave, read_parser);
    const auto w = write_entry(0x6000, 1);
    auto write_parser = x::json::Parser(w);
    claims.claim(output(slave, write_parser)->entry, slave, write_parser);
    ASSERT_TRUE(read_parser.ok()) << read_parser.error_json().dump();
    ASSERT_TRUE(write_parser.ok()) << write_parser.error_json().dump();
}

/// @brief it should sort channels by slave position, then by entry index.
TEST(SortByPosition, SortsBySlavePositionThenByIndex) {
    slave::Properties slave1 = repeated_name_slave();
    slave1.position = 1;
    slave::Properties slave2 = repeated_name_slave();
    slave2.position = 2;
    const auto j1 = read_entry(0x6020, 1);
    const auto j2 = read_entry(0x6000, 1);
    const auto j3 = read_entry(0x6010, 1);
    auto p1 = x::json::Parser(j1);
    auto p2 = x::json::Parser(j2);
    auto p3 = x::json::Parser(j3);
    std::vector<std::unique_ptr<Input>> channels;
    channels.push_back(input(slave1, p1));
    channels.push_back(input(slave2, p2));
    channels.push_back(input(slave1, p3));
    sort_by_position(channels);
    ASSERT_EQ(channels[0]->entry.slave_position, 1);
    ASSERT_EQ(channels[0]->entry.index, 0x6010);
    ASSERT_EQ(channels[1]->entry.slave_position, 1);
    ASSERT_EQ(channels[1]->entry.index, 0x6020);
    ASSERT_EQ(channels[2]->entry.slave_position, 2);
    ASSERT_EQ(channels[2]->entry.index, 0x6000);
}
}
