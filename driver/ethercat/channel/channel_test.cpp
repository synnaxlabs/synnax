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

nlohmann::json read_entry(
    const uint16_t index,
    const uint8_t sub_index,
    const std::string &data_type = "uint8",
    const bool disabled = false
) {
    return {
        {"type", "entry"},
        {"device", "dev-001"},
        {"channel", 42},
        {"disabled", disabled},
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
    ASSERT_TRUE(ch->enabled);
    ASSERT_TRUE(ch->is_input);
    ASSERT_EQ(ch->device_key, "dev-001");
    ASSERT_EQ(ch->synnax_key, 42);
    ASSERT_EQ(ch->slave_position, 1);
    ASSERT_EQ(ch->index, 0x6000);
    ASSERT_EQ(ch->sub_index, 0x11);
    ASSERT_EQ(ch->bit_length, 16);
    ASSERT_EQ(ch->data_type, x::telem::UINT16_T);
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
        "data type float64 is 8 bytes but entry 0x6000:11 on slave EL3102 holds 2 "
        "bytes. Pick a 2 byte type."
    );
    ASSERT_EQ(ch->index, 0);
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
        "data type int8 is 1 byte but entry 0x6000:11 on slave EL3102 holds 2 bytes. "
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
        "data type timestamp cannot be exchanged with entry 0x6000:11 on slave "
        "EL3102. Pick a 2 byte number type."
    );
    ASSERT_EQ(ch->index, 0);
}

/// @brief it should bind no entry when the slave has no entry at the address.
TEST(Input, BindsNoEntryAtAnAddressTheSlaveDoesNotHave) {
    const auto slave = padded_slave();
    const auto j = read_entry(0x6010, 1);
    auto parser = x::json::Parser(j);
    const auto ch = input(slave, parser);
    ASSERT_FALSE(parser.ok());
    ASSERT_EQ(ch->index, 0);
    ASSERT_EQ(ch->bit_length, 0);
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
        ASSERT_EQ(ch->index, 0x6000 + 0x10 * i);
        ASSERT_EQ(ch->sub_index, 1);
        ASSERT_EQ(ch->bit_length, 1);
        addresses.emplace(ch->index, ch->sub_index);
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
    ASSERT_EQ(message, "slave EL3102 has no input entry at 0x6010:01");
}

/// @brief it should report the padding address as an address with no entry.
TEST(Input, ReportsAPaddingAddress) {
    const auto slave = padded_slave();
    const auto j = read_entry(0, 0);
    auto parser = x::json::Parser(j);
    input(slave, parser);
    const auto [path, message] = only_error(parser);
    ASSERT_EQ(path, "index");
    ASSERT_EQ(message, "slave EL3102 has no input entry at 0x0000:00");
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
    ASSERT_EQ(message, "slave EL1008 has no input entry at 0x7000:01");
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
        "'Input' matches more than one input entry on slave EL1008: 0x6000:01 "
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
        "'Analog Input Ch1' matches no input entry on slave EL3102. Pick an entry."
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
        "'Value' matches input entry 0x6000:11 on slave EL3102. Pick an entry."
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
        "'Sync error' matches input entry 0x1C32:20 (Module 1) on slave EJ7342. Pick "
        "an entry."
    );
}

/// @brief it should report no errors for a disabled channel, even when its entry cannot
/// be bound.
TEST(Input, DisabledChannelsCarryNoErrors) {
    const auto slave = padded_slave();
    auto unresolved = read_unresolved("Value");
    unresolved["disabled"] = true;
    auto unresolved_parser = x::json::Parser(unresolved);
    ASSERT_FALSE(input(slave, unresolved_parser)->enabled);
    ASSERT_TRUE(unresolved_parser.ok()) << unresolved_parser.error_json().dump();
    const auto missing = read_entry(0x6100, 1, "uint8", true);
    auto missing_parser = x::json::Parser(missing);
    ASSERT_FALSE(input(slave, missing_parser)->enabled);
    ASSERT_TRUE(missing_parser.ok()) << missing_parser.error_json().dump();
}

/// @brief it should keep the disabled flag of a channel.
TEST(Input, KeepsTheDisabledFlag) {
    const auto slave = padded_slave();
    const auto j = read_entry(0x6000, 1, "uint8", true);
    auto parser = x::json::Parser(j);
    const auto ch = input(slave, parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_FALSE(ch->enabled);
}

/// @brief it should enable a channel when its config has no disabled field.
TEST(Input, EnabledDefaultsToTrue) {
    const auto slave = padded_slave();
    auto j = read_entry(0x6000, 1);
    j.erase("disabled");
    auto parser = x::json::Parser(j);
    const auto ch = input(slave, parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_TRUE(ch->enabled);
}

/// @brief it should copy the key, name, and data type of the Synnax channel it binds.
TEST(Input, BindRemoteInfoCopiesChannelInformation) {
    const auto slave = padded_slave();
    const auto j = read_entry(0x6000, 0x11, "int16");
    auto parser = x::json::Parser(j);
    const auto ch = input(slave, parser);
    synnax::channel::Channel remote_ch;
    remote_ch.key = 42;
    remote_ch.name = "test_channel";
    remote_ch.data_type = x::telem::INT16_T;
    ch->bind_remote_info(remote_ch);
    ASSERT_EQ(ch->ch.key, 42);
    ASSERT_EQ(ch->ch.name, "test_channel");
    ASSERT_EQ(ch->ch.data_type, x::telem::INT16_T);
}

/// @brief it should bind an output channel to the entry at its address, with the bit
/// length from the slave.
TEST(Output, BindsAnEntryWithBitLengthFromTheSlave) {
    const auto slave = repeated_name_slave();
    const auto j = write_entry(0x7030, 1);
    auto parser = x::json::Parser(j);
    const auto ch = output(slave, parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_FALSE(ch->is_input);
    ASSERT_EQ(ch->command_key, 7);
    ASSERT_EQ(ch->state_key, 8);
    ASSERT_EQ(ch->slave_position, 3);
    ASSERT_EQ(ch->index, 0x7030);
    ASSERT_EQ(ch->sub_index, 1);
    ASSERT_EQ(ch->bit_length, 1);
    ASSERT_EQ(ch->data_type, x::telem::UINT8_T);
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
    ASSERT_EQ(message, "slave EL1008 has no output entry at 0x6000:01");
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
        "data type float32 is 4 bytes but entry 0x7000:01 on slave EL1008 holds 1 "
        "byte. Pick a 1 byte type."
    );
    ASSERT_EQ(ch->index, 0);
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
        "'Output' matches more than one output entry on slave EL1008: 0x7000:01 "
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
        "'Analog Output Ch2' matches no output entry on slave EL1008. Pick an entry."
    );
}

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
        claims.claim(*ch, slave, parser);
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
    claims.claim(*input(slave, first), slave, first);
    ASSERT_TRUE(first.ok());
    auto second = x::json::Parser(j);
    claims.claim(*input(slave, second), slave, second, "channels.1.");
    const auto [path, message] = only_error(second);
    ASSERT_EQ(path, "channels.1.index");
    ASSERT_EQ(message, "another channel already uses entry 0x6030:01 on slave EL1008");
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
    claims.claim(*input(first_slave, first), first_slave, first);
    auto second = x::json::Parser(j);
    claims.claim(*input(second_slave, second), second_slave, second);
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
    claims.claim(*input(slave, read_parser), slave, read_parser);
    const auto w = write_entry(0x6000, 1);
    auto write_parser = x::json::Parser(w);
    claims.claim(*output(slave, write_parser), slave, write_parser);
    ASSERT_TRUE(read_parser.ok()) << read_parser.error_json().dump();
    ASSERT_TRUE(write_parser.ok()) << write_parser.error_json().dump();
}

/// @brief it should not claim an entry for a channel that bound no entry.
TEST(Claims, IgnoresChannelsThatBoundNoEntry) {
    const auto slave = repeated_name_slave();
    Claims claims;
    const auto j = read_unresolved("Input");
    auto first = x::json::Parser(j);
    claims.claim(*input(slave, first), slave, first);
    auto second = x::json::Parser(j);
    claims.claim(*input(slave, second), slave, second);
    ASSERT_EQ(only_error(second).first, "pdo");
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
    ASSERT_EQ(channels[0]->slave_position, 1);
    ASSERT_EQ(channels[0]->index, 0x6010);
    ASSERT_EQ(channels[1]->slave_position, 1);
    ASSERT_EQ(channels[1]->index, 0x6020);
    ASSERT_EQ(channels[2]->slave_position, 2);
    ASSERT_EQ(channels[2]->index, 0x6000);
}
}
