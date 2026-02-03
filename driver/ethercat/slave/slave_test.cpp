// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <gtest/gtest.h>

#include "driver/ethercat/slave/slave.h"

namespace ethercat::slave {

TEST(SlaveStateToString, ConvertsAllStatesToExpectedStrings) {
    ASSERT_EQ(slave_state_to_string(State::INIT), "INIT");
    ASSERT_EQ(slave_state_to_string(State::PRE_OP), "PRE-OP");
    ASSERT_EQ(slave_state_to_string(State::SAFE_OP), "SAFE-OP");
    ASSERT_EQ(slave_state_to_string(State::OP), "OP");
    ASSERT_EQ(slave_state_to_string(State::BOOT), "BOOT");
    ASSERT_EQ(slave_state_to_string(State::UNKNOWN), "UNKNOWN");
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

TEST(PropertiesFindInputPdo, ReturnsNulloptWhenEmpty) {
    Properties props;
    ASSERT_FALSE(props.find_input_pdo("nonexistent").has_value());
}

TEST(PropertiesFindInputPdo, ReturnsNulloptWhenNameNotFound) {
    Properties props{
        .input_pdos = {
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 1,
             .bit_length = 16,
             .name = "Position"},
        },
    };
    ASSERT_FALSE(props.find_input_pdo("Velocity").has_value());
}

TEST(PropertiesFindInputPdo, ReturnsMatchingPdo) {
    Properties props{
        .input_pdos = {
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 1,
             .bit_length = 16,
             .name = "Position"},
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 2,
             .bit_length = 32,
             .name = "Velocity"},
        },
    };
    auto result = props.find_input_pdo("Velocity");
    ASSERT_TRUE(result.has_value());
    ASSERT_EQ(result->name, "Velocity");
    ASSERT_EQ(result->bit_length, 32);
}

TEST(PropertiesFindOutputPdo, ReturnsNulloptWhenEmpty) {
    Properties props;
    ASSERT_FALSE(props.find_output_pdo("nonexistent").has_value());
}

TEST(PropertiesFindOutputPdo, ReturnsNulloptWhenNameNotFound) {
    Properties props{
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 1,
             .bit_length = 16,
             .name = "TargetPosition"},
        },
    };
    ASSERT_FALSE(props.find_output_pdo("TargetVelocity").has_value());
}

TEST(PropertiesFindOutputPdo, ReturnsMatchingPdo) {
    Properties props{
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 1,
             .bit_length = 16,
             .name = "TargetPosition"},
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 2,
             .bit_length = 8,
             .name = "ControlWord"},
        },
    };
    auto result = props.find_output_pdo("ControlWord");
    ASSERT_TRUE(result.has_value());
    ASSERT_EQ(result->name, "ControlWord");
    ASSERT_EQ(result->bit_length, 8);
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
    auto parser = xjson::Parser(j);
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
    auto parser = xjson::Parser(j);
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
    auto parser = xjson::Parser(j);
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
              .data_type = telem::INT16_T}},
        .output_pdos =
            {{.pdo_index = 0x1600,
              .index = 0x7000,
              .sub_index = 1,
              .bit_length = 8,
              .is_input = false,
              .name = "Control",
              .data_type = telem::UINT8_T}},
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

}
