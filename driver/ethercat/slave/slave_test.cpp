// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

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

/// @brief parses a startup parameter and returns it with the first field error, or an
/// empty path and message when parsing succeeds.
std::tuple<StartupParameter, std::string, std::string>
parse_parameter(const x::json::json &json) {
    x::json::Parser parser(json);
    const auto param = StartupParameter::parse(parser);
    if (parser.ok()) return {param, "", ""};
    const auto err = parser.error_json()["errors"][0];
    return {param, err["path"], err["message"]};
}

/// @brief it should parse each field of a startup parameter.
TEST(StartupParameter, ParsesEachField) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x8000},
         {"sub_index", 0x15},
         {"data_type", "uint16"},
         {"value", 3},
         {"name", "Filter"}}
    );
    EXPECT_EQ(path, "");
    EXPECT_EQ(
        param,
        (StartupParameter{
            .index = 0x8000,
            .sub_index = 0x15,
            .data_type = x::telem::UINT16_T,
            .value = 3,
            .name = "Filter",
        })
    );
}

/// @brief it should reject an index that does not fit in 16 bits.
TEST(StartupParameter, RejectsAnIndexAboveSixteenBits) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x10000}, {"sub_index", 1}, {"data_type", "uint8"}, {"value", 1}}
    );
    EXPECT_EQ(path, "index");
    EXPECT_EQ(message, "must be a whole number from 0x0000 to 0xFFFF");
}

/// @brief it should reject a sub-index that does not fit in 8 bits.
TEST(StartupParameter, RejectsASubIndexAboveEightBits) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x8000}, {"sub_index", 256}, {"data_type", "uint8"}, {"value", 1}}
    );
    EXPECT_EQ(path, "sub_index");
    EXPECT_EQ(message, "must be a whole number from 0 to 255");
}

/// @brief it should reject a data type that is not numeric.
TEST(StartupParameter, RejectsANonNumericDataType) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x8000}, {"sub_index", 1}, {"data_type", "string"}, {"value", 1}}
    );
    EXPECT_EQ(path, "data_type");
    EXPECT_EQ(message, "unsupported data type string");
}

/// @brief it should reject a value outside the range of its data type.
TEST(StartupParameter, RejectsAValueOutsideTheDataType) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x8000}, {"sub_index", 1}, {"data_type", "uint8"}, {"value", 300}}
    );
    EXPECT_EQ(path, "value");
    EXPECT_EQ(message, "must be a whole number from 0 to 255");
}

/// @brief it should reject a fractional value for an integer data type.
TEST(StartupParameter, RejectsAFractionForAnIntegerType) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x8000}, {"sub_index", 1}, {"data_type", "int16"}, {"value", 1.5}}
    );
    EXPECT_EQ(path, "value");
    EXPECT_EQ(message, "must be a whole number from -32768 to 32767");
}

/// @brief it should reject a sub-index with a fraction instead of truncating it.
TEST(StartupParameter, RejectsAFractionalSubIndex) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x8000}, {"sub_index", 21.9}, {"data_type", "uint8"}, {"value", 1}}
    );
    EXPECT_EQ(path, "sub_index");
    EXPECT_EQ(message, "must be a whole number from 0 to 255");
}

/// @brief it should reject an int64 value of 2^53 or more, which a double may have
/// rounded.
TEST(StartupParameter, RejectsAnInt64ThatADoubleRounds) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x8000},
         {"sub_index", 1},
         {"data_type", "int64"},
         {"value", 9007199254740993}}
    );
    EXPECT_EQ(path, "value");
    EXPECT_EQ(
        message,
        "must be a whole number from -9007199254740991 to 9007199254740991"
    );
}

/// @brief it should accept the largest int64 magnitude that a double holds exactly.
TEST(StartupParameter, AcceptsTheLargestExactInt64) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x8000},
         {"sub_index", 1},
         {"data_type", "int64"},
         {"value", -9007199254740991}}
    );
    EXPECT_EQ(path, "");
    EXPECT_EQ(param.value, -9007199254740991.0);
}

/// @brief it should reject a uint64 value of 2^53 or more, which a double may have
/// rounded.
TEST(StartupParameter, RejectsAUint64ThatADoubleRounds) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x8000},
         {"sub_index", 1},
         {"data_type", "uint64"},
         {"value", 18446744073709551615ULL}}
    );
    EXPECT_EQ(path, "value");
    EXPECT_EQ(message, "must be a whole number from 0 to 9007199254740991");
}

/// @brief it should reject a value above the float32 maximum.
TEST(StartupParameter, RejectsAValueAboveTheFloat32Maximum) {
    const auto [param, path, message] = parse_parameter(
        {{"index", 0x8000}, {"sub_index", 1}, {"data_type", "float32"}, {"value", 1e39}}
    );
    EXPECT_EQ(path, "value");
    EXPECT_EQ(message, "1e+39 does not fit in float32");
}

/// @brief it should encode the value as the little-endian bytes of its data type.
TEST(StartupParameter, EncodesLittleEndianBytesOfItsType) {
    EXPECT_EQ(
        (StartupParameter{.data_type = x::telem::UINT16_T, .value = 0x1234}).bytes(),
        (std::vector<uint8_t>{0x34, 0x12})
    );
    EXPECT_EQ(
        (StartupParameter{.data_type = x::telem::INT32_T, .value = -2}).bytes(),
        (std::vector<uint8_t>{0xFE, 0xFF, 0xFF, 0xFF})
    );
    EXPECT_EQ(
        (StartupParameter{.data_type = x::telem::FLOAT32_T, .value = 1}).bytes(),
        (std::vector<uint8_t>{0x00, 0x00, 0x80, 0x3F})
    );
}

/// @brief it should parse the startup parameters of a slave.
TEST(Properties, ParsesStartupParameters) {
    x::json::Parser parser(
        x::json::json{
            {"serial", 1},
            {"vendor_id", 2},
            {"product_code", 3},
            {"revision", 4},
            {"name", "EL3102"},
            {"position", 0},
            {"enabled", true},
            {"pdos",
             {{"inputs", x::json::json::array()}, {"outputs", x::json::json::array()}}},
            {"startup_parameters",
             {{{"index", 0x8000},
               {"sub_index", 0x15},
               {"data_type", "uint16"},
               {"value", 3}}}},
        }
    );
    const auto props = Properties::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(props.startup_parameters.size(), 1);
    EXPECT_EQ(props.startup_parameters[0].index, 0x8000);
    EXPECT_FALSE(props.to_json().contains("startup_parameters"));
}

/// @brief it should report a field error for each invalid startup parameter.
TEST(Properties, ReportsEveryInvalidStartupParameter) {
    x::json::Parser parser(
        x::json::json{
            {"serial", 1},
            {"vendor_id", 2},
            {"product_code", 3},
            {"revision", 4},
            {"name", "EL3102"},
            {"position", 0},
            {"enabled", true},
            {"pdos",
             {{"inputs", x::json::json::array()}, {"outputs", x::json::json::array()}}},
            {"startup_parameters",
             {{{"index", 0x8000},
               {"sub_index", 0x15},
               {"data_type", "uint8"},
               {"value", 300}},
              {{"index", 0x8000},
               {"sub_index", 0x19},
               {"data_type", "uint16"},
               {"value", 70000}}}},
        }
    );
    Properties::parse(parser);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 2);
    EXPECT_EQ(errors[0]["path"], "startup_parameters.0.value");
    EXPECT_EQ(errors[1]["path"], "startup_parameters.1.value");
    EXPECT_EQ(errors[1]["message"], "must be a whole number from 0 to 65535");
}

/// @brief it should parse a slave that has no startup parameters field.
TEST(Properties, ParsesWithoutStartupParameters) {
    x::json::Parser parser(
        x::json::json{
            {"serial", 1},
            {"vendor_id", 2},
            {"product_code", 3},
            {"revision", 4},
            {"name", "EL3102"},
            {"position", 0},
            {"enabled", true},
            {"pdos",
             {{"inputs", x::json::json::array()}, {"outputs", x::json::json::array()}}},
        }
    );
    const auto props = Properties::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_TRUE(props.startup_parameters.empty());
}
}
