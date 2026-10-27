// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ethercat/slave/slave.h"

namespace driver::ethercat::slave {

TEST(SlaveStateToString, ConvertsAllStatesToExpectedStrings) {
    ASSERT_EQ(slave_state_to_string(State::INIT), "INIT");
    ASSERT_EQ(slave_state_to_string(State::PRE_OP), "PRE-OP");
    ASSERT_EQ(slave_state_to_string(State::SAFE_OP), "SAFE-OP");
    ASSERT_EQ(slave_state_to_string(State::OP), "OP");
    ASSERT_EQ(slave_state_to_string(State::BOOT), "BOOT");
    ASSERT_EQ(slave_state_to_string(State::UNKNOWN), "UNKNOWN");
}

TEST(FromAlState, ConvertsStandardStates) {
    ASSERT_EQ(from_al_state(0x01), State::INIT);
    ASSERT_EQ(from_al_state(0x02), State::PRE_OP);
    ASSERT_EQ(from_al_state(0x03), State::BOOT);
    ASSERT_EQ(from_al_state(0x04), State::SAFE_OP);
    ASSERT_EQ(from_al_state(0x08), State::OP);
}

TEST(FromAlState, ReturnsUnknownForInvalidValues) {
    ASSERT_EQ(from_al_state(0x00), State::UNKNOWN);
    ASSERT_EQ(from_al_state(0x05), State::UNKNOWN);
    ASSERT_EQ(from_al_state(0x0F), State::UNKNOWN);
}

TEST(FromAlState, MasksErrorFlagBits) {
    ASSERT_EQ(from_al_state(0x11), State::INIT);
    ASSERT_EQ(from_al_state(0x12), State::PRE_OP);
    ASSERT_EQ(from_al_state(0x14), State::SAFE_OP);
    ASSERT_EQ(from_al_state(0x18), State::OP);
}

TEST(FromAlState, AcceptsUint8Values) {
    const uint8_t igh_state = 0x08;
    ASSERT_EQ(from_al_state(igh_state), State::OP);
}

TEST(PropertiesPdoCount, ReturnsZeroForEmptyProperties) {
    Properties props;
    ASSERT_EQ(props.pdo_count(), 0);
}

TEST(PropertiesPdoCount, ReturnsSumOfInputAndOutputPdos) {
    Properties props{
        .input_pdos =
            {
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 1,
                 .bit_length = 16},
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 2,
                 .bit_length = 16},
            },
        .output_pdos = {
            {.pdo_index = 0x1600, .index = 0x7000, .sub_index = 1, .bit_length = 8},
        },
    };
    ASSERT_EQ(props.pdo_count(), 3);
}

Properties find_pdo_slave() {
    return Properties{
        .input_pdos =
            {
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 1,
                 .bit_length = 1,
                 .is_input = true,
                 .name = "Status"},
                {.pdo_index = 0x1A00,
                 .index = 0,
                 .sub_index = 0,
                 .bit_length = 7,
                 .is_input = true,
                 .name = ""},
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 0x11,
                 .bit_length = 16,
                 .is_input = true,
                 .name = "Value"},
            },
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 1,
             .bit_length = 8,
             .is_input = false,
             .name = "ControlWord"},
        },
    };
}

/// @brief it should find no entry on a slave that has no entries.
TEST(PropertiesFindPdo, ReturnsNulloptWhenEmpty) {
    Properties props;
    ASSERT_FALSE(props.find_pdo(0x6000, 1, true).has_value());
}

/// @brief it should find the input entry at an address.
TEST(PropertiesFindPdo, ReturnsTheInputEntryAtAnAddress) {
    const auto result = find_pdo_slave().find_pdo(0x6000, 0x11, true);
    ASSERT_TRUE(result.has_value());
    ASSERT_EQ(result.value().name, "Value");
    ASSERT_EQ(result.value().bit_length, 16);
}

/// @brief it should find the output entry at an address.
TEST(PropertiesFindPdo, ReturnsTheOutputEntryAtAnAddress) {
    const auto result = find_pdo_slave().find_pdo(0x7000, 1, false);
    ASSERT_TRUE(result.has_value());
    ASSERT_EQ(result.value().name, "ControlWord");
    ASSERT_EQ(result.value().bit_length, 8);
}

/// @brief it should find no entry at an address the slave does not have.
TEST(PropertiesFindPdo, ReturnsNulloptForAnAddressTheSlaveDoesNotHave) {
    ASSERT_FALSE(find_pdo_slave().find_pdo(0x6010, 1, true).has_value());
    ASSERT_FALSE(find_pdo_slave().find_pdo(0x6000, 2, true).has_value());
}

/// @brief it should search only the entries of the given direction.
TEST(PropertiesFindPdo, SearchesOnlyTheGivenDirection) {
    ASSERT_FALSE(find_pdo_slave().find_pdo(0x7000, 1, true).has_value());
    ASSERT_FALSE(find_pdo_slave().find_pdo(0x6000, 1, false).has_value());
}

/// @brief it should never find a padding entry.
TEST(PropertiesFindPdo, NeverMatchesPadding) {
    ASSERT_FALSE(find_pdo_slave().find_pdo(0, 0, true).has_value());
}

TEST(PropertiesParse, ParsesBasicProperties) {
    nlohmann::json j = {
        {"serial", 12345},
        {"vendor_id", 0x00000002},
        {"product_code", 0x12345678},
        {"revision", 1},
        {"name", "EL2008"},
        {"network", "eth0"},
        {"position", 1},
        {"enabled", true}
    };
    auto parser = x::json::Parser(j);
    auto props = Properties::parse(parser);
    ASSERT_EQ(props.serial, 12345);
    ASSERT_EQ(props.vendor_id, 0x00000002);
    ASSERT_EQ(props.product_code, 0x12345678);
    ASSERT_EQ(props.revision, 1);
    ASSERT_EQ(props.name, "EL2008");
    ASSERT_EQ(props.network, "eth0");
    ASSERT_EQ(props.position, 1);
    ASSERT_TRUE(props.enabled);
}

TEST(PropertiesParse, ParsesWithPdos) {
    nlohmann::json j = {
        {"serial", 0},
        {"vendor_id", 0x00000002},
        {"product_code", 0x12345678},
        {"revision", 1},
        {"name", "EL3001"},
        {"network", "eth0"},
        {"position", 2},
        {"enabled", true},
        {"pdos",
         {{"inputs",
           {{{"pdo_index", 0x1A00},
             {"index", 0x6000},
             {"sub_index", 1},
             {"bit_length", 16},
             {"name", "Value"},
             {"data_type", "int16"}}}},
          {"outputs",
           {{{"pdo_index", 0x1600},
             {"index", 0x7000},
             {"sub_index", 1},
             {"bit_length", 8},
             {"name", "Control"},
             {"data_type", "uint8"}}}}}}
    };
    auto parser = x::json::Parser(j);
    auto props = Properties::parse(parser);
    ASSERT_EQ(props.input_pdos.size(), 1);
    ASSERT_EQ(props.input_pdos[0].name, "Value");
    ASSERT_EQ(props.input_pdos[0].bit_length, 16);
    ASSERT_TRUE(props.input_pdos[0].is_input);
    ASSERT_EQ(props.output_pdos.size(), 1);
    ASSERT_EQ(props.output_pdos[0].name, "Control");
    ASSERT_EQ(props.output_pdos[0].bit_length, 8);
    ASSERT_FALSE(props.output_pdos[0].is_input);
}

TEST(PropertiesParse, HandlesMissingPdosField) {
    nlohmann::json j = {
        {"serial", 0},
        {"vendor_id", 0x00000002},
        {"product_code", 0x12345678},
        {"revision", 1},
        {"name", "EL2008"},
        {"network", "eth0"},
        {"position", 1},
        {"enabled", false}
    };
    auto parser = x::json::Parser(j);
    auto props = Properties::parse(parser);
    ASSERT_EQ(props.input_pdos.size(), 0);
    ASSERT_EQ(props.output_pdos.size(), 0);
}

TEST(PropertiesToJson, SerializesAllFields) {
    Properties props{
        .network = "eth0",
        .position = 3,
        .vendor_id = 0x00000002,
        .product_code = 0x12345678,
        .revision = 1,
        .serial = 12345,
        .name = "EL3001",
        .input_bits = 16,
        .output_bits = 8,
        .input_pdos =
            {{.pdo_index = 0x1A00,
              .index = 0x6000,
              .sub_index = 1,
              .bit_length = 16,
              .is_input = true,
              .name = "Value",
              .data_type = x::telem::INT16_T}},
        .output_pdos =
            {{.pdo_index = 0x1600,
              .index = 0x7000,
              .sub_index = 1,
              .bit_length = 8,
              .is_input = false,
              .name = "Control",
              .data_type = x::telem::UINT8_T}},
        .coe_pdo_order_reliable = true,
        .enabled = true,
    };

    auto j = props.to_json();
    ASSERT_EQ(j["vendor_id"], 0x00000002);
    ASSERT_EQ(j["product_code"], 0x12345678);
    ASSERT_EQ(j["revision"], 1);
    ASSERT_EQ(j["serial"], 12345);
    ASSERT_EQ(j["name"], "EL3001");
    ASSERT_EQ(j["network"], "eth0");
    ASSERT_EQ(j["position"], 3);
    ASSERT_EQ(j["input_bits"], 16);
    ASSERT_EQ(j["output_bits"], 8);
    ASSERT_EQ(j["pdo_order_reliable"], true);
    ASSERT_EQ(j["enabled"], true);
    ASSERT_EQ(j["pdos"]["inputs"].size(), 1);
    ASSERT_EQ(j["pdos"]["inputs"][0]["name"], "Value");
    ASSERT_EQ(j["pdos"]["outputs"].size(), 1);
    ASSERT_EQ(j["pdos"]["outputs"][0]["name"], "Control");
}

/// @brief it should leave padding entries out of the JSON of a slave.
TEST(PropertiesToJson, OmitsPaddingEntries) {
    Properties props{
        .input_pdos = {
            {.pdo_index = 0x1A00, .index = 0x6000, .sub_index = 1, .bit_length = 1},
            {.pdo_index = 0x1A00, .index = 0, .bit_length = 7},
            {.pdo_index = 0x1A00, .index = 0x6000, .sub_index = 17, .bit_length = 16},
        },
    };
    const auto j = props.to_json();
    ASSERT_EQ(j["pdos"]["inputs"].size(), 2);
    EXPECT_EQ(j["pdos"]["inputs"][1]["sub_index"], 17);
}
}
