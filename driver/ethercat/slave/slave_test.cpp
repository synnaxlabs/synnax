// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/slave/slave.h"

namespace driver::ethercat::slave {

/// @brief it should name each slave state as the EtherCAT standard does.
TEST(StateToString, NamesEachState) {
    ASSERT_EQ(to_string(State::INIT), "INIT");
    ASSERT_EQ(to_string(State::PRE_OP), "PRE-OP");
    ASSERT_EQ(to_string(State::SAFE_OP), "SAFE-OP");
    ASSERT_EQ(to_string(State::OP), "OP");
    ASSERT_EQ(to_string(State::BOOT), "BOOT");
    ASSERT_EQ(to_string(State::UNKNOWN), "UNKNOWN");
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
    ASSERT_TRUE(props.config.enabled);
}

/// @brief it should read back the settings that a scan wrote for a slave.
TEST(PropertiesParse, ReadsTheSettingsOfTheCatalog) {
    const x::json::json settings{
        {"channels", x::json::json::array({{{"entries", x::json::json::array()}}})},
    };
    auto parser = x::json::Parser(
        Properties{.name = "6xSTG", .settings = settings}.to_json()
    );
    EXPECT_EQ(Properties::parse(parser).settings, settings);
}

/// @brief it should leave the settings empty for a slave that the catalog does not
/// describe.
TEST(PropertiesParse, LeavesMissingSettingsEmpty) {
    auto parser = x::json::Parser(Properties{.name = "EL3102"}.to_json());
    EXPECT_FALSE(Properties::parse(parser).settings.has_value());
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

/// @brief it should parse a device stored before slaves listed their OpModes as a slave
/// with no OpModes.
TEST(PropertiesParse, DefaultsToNoOpModes) {
    auto j = Properties{.position = 1, .name = "EL2008"}.to_json();
    j.erase("op_modes");
    auto parser = x::json::Parser(j);
    const auto props = Properties::parse(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_TRUE(props.op_modes.empty());
}

/// @brief it should give back the OpModes of a slave in SII order after a JSON round
/// trip.
TEST(PropertiesParse, RoundTripsOpModes) {
    const Properties props{
        .position = 1,
        .name = "6xSTG",
        .op_modes = {
            {.name = "SM-Synchron", .sync0_factor = 1},
            {.name = "DC", .assign_activate = 0x0700, .sync0_cycle = 1000000000},
            {.name = "DC-bus1",
             .assign_activate = 0x0700,
             .sync0_cycle = 1000,
             .sync1_factor = -1000}
        },
    };
    auto parser = x::json::Parser(props.to_json());
    const auto parsed = Properties::parse(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(parsed.op_modes, props.op_modes);
}

/// @brief it should give back the chosen mode of a slave after a JSON round trip.
TEST(PropertiesParse, RoundTripsTheChosenMode) {
    const Properties props{
        .position = 1,
        .name = "6xSTG",
        .config = {
            .op_mode = dc::Choice{
                .index = 1,
                .assign_activate = 0x0300,
                .cycles = {.sync0 = x::telem::MICROSECOND, .sync1 = x::telem::SECOND},
            },
        },
    };
    auto parser = x::json::Parser(props.to_json());
    const auto parsed = Properties::parse(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(parsed.config.op_mode, props.config.op_mode);
}

/// @brief it should leave out the chosen mode of a slave with no choice, as the Console
/// stores no choice as a missing key.
TEST(PropertiesToJson, LeavesOutAMissingChoice) {
    EXPECT_FALSE(Properties{.name = "6xSTG"}.to_json().contains("op_mode"));
}

/// @brief it should write the settings that the catalog describes for a slave.
TEST(PropertiesToJson, WritesTheSettingsOfTheCatalog) {
    const x::json::json settings{
        {"device", x::json::json::array()},
        {"channels", x::json::json::array({{{"entries", x::json::json::array()}}})},
    };
    const Properties props{.name = "6xSTG", .settings = settings};
    EXPECT_EQ(props.to_json().at("settings"), settings);
}

/// @brief it should replace a startup parameter at the same address in place, and add a
/// parameter at a new address last.
TEST(ConfigAddStartupParameters, ReplacesAParameterAtTheSameAddressInPlace) {
    const auto param = [](const uint8_t sub_index, const double value) {
        return StartupParameter{
            .index = 0x8040,
            .sub_index = sub_index,
            .data_type = x::telem::UINT16_T,
            .value = value,
        };
    };
    Config config{.startup_parameters = {param(1, 1), param(2, 2)}};
    config.add_startup_parameters({param(1, 5), param(3, 7)});
    EXPECT_EQ(
        config.startup_parameters,
        (std::vector<StartupParameter>{param(1, 5), param(2, 2), param(3, 7)})
    );
}

/// @brief it should leave out the settings of a slave that the catalog does not
/// describe.
TEST(PropertiesToJson, LeavesOutMissingSettings) {
    EXPECT_FALSE(Properties{.name = "EL3102"}.to_json().contains("settings"));
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
        .input_pdos =
            {{.pdo_index = 0x1A00,
              .index = 0x6000,
              .sub_index = 1,
              .bit_length = 16,
              .is_input = true,
              .name = "Value",
              .data_type = x::telem::INT16_T}},
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 1,
             .bit_length = 8,
             .is_input = false,
             .name = "Control",
             .data_type = x::telem::UINT8_T}
        },
    };

    auto j = props.to_json();
    ASSERT_EQ(j["vendor_id"], 0x00000002);
    ASSERT_EQ(j["product_code"], 0x12345678);
    ASSERT_EQ(j["revision"], 1);
    ASSERT_EQ(j["serial"], 12345);
    ASSERT_EQ(j["name"], "EL3001");
    ASSERT_EQ(j["network"], "eth0");
    ASSERT_EQ(j["position"], 3);
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
            {"position", 1},
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
    ASSERT_EQ(props.config.startup_parameters.size(), 1);
    EXPECT_EQ(props.config.startup_parameters[0].index, 0x8000);
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
            {"position", 1},
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

/// @brief it should reject a position that no slave on a bus can have.
TEST(Properties, RejectsAPositionOutsideTheBus) {
    for (const auto &position:
         {x::json::json(0),
          x::json::json(-1),
          x::json::json(65536),
          x::json::json(1.5)}) {
        x::json::Parser parser(
            x::json::json{
                {"serial", 1},
                {"vendor_id", 2},
                {"product_code", 3},
                {"revision", 4},
                {"name", "EL3102"},
                {"position", position},
                {"enabled", true},
                {"pdos",
                 {{"inputs", x::json::json::array()},
                  {"outputs", x::json::json::array()}}},
            }
        );
        Properties::parse(parser);
        const auto errors = parser.error_json()["errors"];
        ASSERT_EQ(errors.size(), 1) << position;
        EXPECT_EQ(errors[0]["path"], "position");
        EXPECT_EQ(errors[0]["message"], "must be a whole number from 1 to 65535");
    }
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
            {"position", 1},
            {"enabled", true},
            {"pdos",
             {{"inputs", x::json::json::array()}, {"outputs", x::json::json::array()}}},
        }
    );
    const auto props = Properties::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_TRUE(props.config.startup_parameters.empty());
}

/// @brief it should return the sync mode that the catalog gives for the card's mode.
TEST(PropertiesCatalogOpMode, ReturnsTheModeThatTheCatalogGives) {
    Properties props;
    props.settings = x::json::json{
        {"mode",
         {{"value", 0},
          {"op_mode",
           {{"index", 1},
            {"assign_activate", 0x0300},
            {"sync0_cycle", 1000},
            {"sync1_cycle", 1000000000}}}}},
    };
    EXPECT_EQ(
        props.catalog_op_mode(),
        (dc::Choice{
            .index = 1,
            .assign_activate = 0x0300,
            .cycles = {
                .sync0 = x::telem::MICROSECOND,
                .sync1 = x::telem::SECOND,
            },
        })
    );
}

/// @brief it should return no sync mode when the catalog gives none for the card's
/// mode, or describes no mode for the card.
TEST(PropertiesCatalogOpMode, ReturnsNoneWhenTheCatalogGivesNone) {
    Properties props;
    EXPECT_EQ(props.catalog_op_mode(), std::nullopt);
    props.settings = x::json::json{{"device", x::json::json::array()}};
    EXPECT_EQ(props.catalog_op_mode(), std::nullopt);
    props.settings = x::json::json{{"mode", {{"value", 1}, {"shared", true}}}};
    EXPECT_EQ(props.catalog_op_mode(), std::nullopt);
}

namespace {
/// @brief returns a slave at position on the bus with a product and a serial number.
DiscoveryResult on_bus(
    const uint16_t position,
    const uint32_t product_code,
    const uint32_t serial = 0,
    const uint32_t vendor_id = 2
) {
    DiscoveryResult result;
    result.properties.position = position;
    result.properties.vendor_id = vendor_id;
    result.properties.product_code = product_code;
    result.properties.serial = serial;
    return result;
}

/// @brief returns a slave saved at position with a product and a serial number.
Identity saved(
    const uint16_t position,
    const uint32_t product_code,
    const uint32_t serial = 0,
    const uint32_t vendor_id = 2
) {
    return {
        .position = position,
        .vendor_id = vendor_id,
        .product_code = product_code,
        .serial = serial,
        .name = "Slave " + std::to_string(position),
    };
}

using Positions = std::map<uint16_t, uint16_t>;
}

/// @brief it should find each slave at its saved position, and ignore other slaves.
TEST(Bind, FindsTheSlavesAtTheirSavedPositions) {
    EXPECT_EQ(
        ASSERT_NIL_P(bind(
            {saved(1, 0x1), saved(2, 0x2, 777)},
            {on_bus(1, 0x1), on_bus(2, 0x2, 777), on_bus(3, 0x3)}
        )),
        (Positions{{1, 1}, {2, 2}})
    );
}

/// @brief it should reject a saved position with no slave.
TEST(Bind, RejectsAMissingSlave) {
    const auto err = bind({saved(2, 0x2)}, {on_bus(1, 0x1)}).second;
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(err.data, "Slave 2 (Slave 2) is not on the bus.");
}

/// @brief it should reject a slave of another vendor at a saved position.
TEST(Bind, RejectsAnotherVendor) {
    const auto err = bind({saved(1, 0x1, 0, 0x99)}, {on_bus(1, 0x1)}).second;
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(
        err.data,
        "Slave 1 (Slave 1): expected vendor_id 0x00000099, found 0x00000002."
    );
}

/// @brief it should reject another product at a saved position.
TEST(Bind, RejectsAnotherProduct) {
    const auto err = bind({saved(1, 0x87654321)}, {on_bus(1, 0x12345678)}).second;
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(
        err.data,
        "Slave 1 (Slave 1): expected product_code 0x87654321, found 0x12345678."
    );
}

/// @brief it should report the mismatch at the lowest saved position.
TEST(Bind, ReportsTheLowestMismatchedPosition) {
    const auto err = bind(
                         {saved(4, 0x9), saved(3, 0x9), saved(1, 0x1)},
                         {on_bus(1, 0x1), on_bus(3, 0x3), on_bus(4, 0x4)}
    )
                         .second;
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(
        err.data,
        "Slave 3 (Slave 3): expected product_code 0x00000009, found 0x00000003."
    );
}

/// @brief it should find a slave with a serial number after it moves.
TEST(Bind, FindsASlaveBySerialNumberAfterItMoves) {
    EXPECT_EQ(
        ASSERT_NIL_P(
            bind({saved(2, 0x1, 777)}, {on_bus(1, 0x1, 888), on_bus(4, 0x1, 777)})
        ),
        (Positions{{2, 4}})
    );
}

/// @brief it should reject a slave whose serial number is not on the bus, even when a
/// slave of its model is at its position.
TEST(Bind, RejectsASlaveWhoseSerialNumberIsNotOnTheBus) {
    const auto err = bind({saved(1, 0x1, 777)}, {on_bus(1, 0x1, 888)}).second;
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(err.data, "Slave 1 (serial 777) is not on the bus.");
}

/// @brief it should find each of two slaves with one serial number at its saved
/// position.
TEST(Bind, FindsSlavesWithOneSerialNumberAtTheirPositions) {
    EXPECT_EQ(
        ASSERT_NIL_P(bind(
            {saved(2, 0x1, 777), saved(1, 0x1, 777)},
            {on_bus(1, 0x1, 777), on_bus(2, 0x1, 777)}
        )),
        (Positions{{1, 1}, {2, 2}})
    );
}

/// @brief it should keep a slave at its saved position before it gives a moved slave
/// with one serial number a position.
TEST(Bind, KeepsASlaveAtItsPositionBeforeItMovesAnother) {
    EXPECT_EQ(
        ASSERT_NIL_P(bind(
            {saved(1, 0x1, 777), saved(3, 0x1, 777)},
            {on_bus(3, 0x1, 777), on_bus(5, 0x1, 777)}
        )),
        (Positions{{1, 5}, {3, 3}})
    );
}

/// @brief it should reject a slave whose unit is only at a position that a slave with
/// serial number 0 holds.
TEST(Bind, RejectsASlaveWhoseUnitIsAtAHeldPosition) {
    const auto err = bind({saved(1, 0x1), saved(3, 0x1, 777)}, {on_bus(1, 0x1, 777)})
                         .second;
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(err.data, "Slave 3 (serial 777) is not on the bus.");
}

/// @brief it should reject two slaves with one saved position.
TEST(Bind, RejectsTwoSlavesWithOneSavedPosition) {
    auto other = saved(1, 0x1, 888);
    other.name = "Other";
    const auto err = bind(
                         {saved(1, 0x1, 777), other},
                         {on_bus(1, 0x1, 888), on_bus(2, 0x1, 777)}
    )
                         .second;
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(err.data, "Other and Slave 1 are both saved as slave 1.");
}

/// @brief it should leave out each saved slave that it does not find on the bus.
TEST(Locate, LeavesOutTheSlavesThatItDoesNotFind) {
    EXPECT_EQ(
        locate(
            {saved(1, 0x1), saved(2, 0x2), saved(3, 0x3, 777), saved(4, 0x4, 888)},
            {on_bus(1, 0x1), on_bus(2, 0x9), on_bus(5, 0x3, 777)}
        ),
        (Positions{{1, 1}, {3, 5}})
    );
}
}
