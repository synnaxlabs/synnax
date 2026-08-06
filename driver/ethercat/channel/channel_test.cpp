// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ethercat/channel/channel.h"

namespace driver::ethercat::channel {

TEST(Input, ResolvesAutomaticPdoFromSlaveProperties) {
    slave::Properties slave{
        .position = 3,
        .input_pdos = {
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 1,
             .bit_length = 16,
             .is_input = true,
             .name = "Position",
             .data_type = x::telem::INT16_T},
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 2,
             .bit_length = 32,
             .is_input = true,
             .name = "Velocity",
             .data_type = x::telem::INT32_T},
        },
    };

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-001"},
        {"channel", 42},
        {"type", "automatic"},
        {"pdo", "Velocity"}
    };

    auto parser = x::json::Parser(j);
    const auto parsed = ::synnax::ethercat::parse_input_channel(parser);
    ASSERT_EQ(base(parsed).device, "dev-001");
    const auto ch = make_input(parsed, slave, parser);

    ASSERT_TRUE(parser.ok());
    ASSERT_TRUE(ch->enabled);
    ASSERT_EQ(ch->device_key, "dev-001");
    ASSERT_EQ(ch->synnax_key, 42);
    ASSERT_EQ(ch->slave_position, 3);
    ASSERT_EQ(ch->index, 0x6000);
    ASSERT_EQ(ch->sub_index, 2);
    ASSERT_EQ(ch->bit_length, 32);
    ASSERT_EQ(ch->data_type, x::telem::INT32_T);
    ASSERT_TRUE(ch->is_input);
}

TEST(Input, ReportsErrorWhenPdoNotFound) {
    slave::Properties slave{
        .position = 1,
        .input_pdos = {
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 1,
             .bit_length = 16,
             .is_input = true,
             .name = "Position",
             .data_type = x::telem::INT16_T},
        },
    };

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-001"},
        {"channel", 42},
        {"type", "automatic"},
        {"pdo", "NonExistentPdo"}
    };

    auto parser = x::json::Parser(j);
    const auto parsed = ::synnax::ethercat::parse_input_channel(parser);
    make_input(parsed, slave, parser);

    ASSERT_FALSE(parser.ok());
}

TEST(Input, ParsesManualPdoAddressFromJson) {
    slave::Properties slave{.position = 5};

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-002"},
        {"channel", 100},
        {"type", "manual"},
        {"index", 0x6010},
        {"sub_index", 3},
        {"bit_length", 32},
        {"data_type", "float32"}
    };

    auto parser = x::json::Parser(j);
    const auto parsed = ::synnax::ethercat::parse_input_channel(parser);
    const auto ch = make_input(parsed, slave, parser);

    ASSERT_TRUE(parser.ok());
    ASSERT_TRUE(ch->enabled);
    ASSERT_EQ(ch->device_key, "dev-002");
    ASSERT_EQ(ch->synnax_key, 100);
    ASSERT_EQ(ch->slave_position, 5);
    ASSERT_EQ(ch->index, 0x6010);
    ASSERT_EQ(ch->sub_index, 3);
    ASSERT_EQ(ch->bit_length, 32);
    ASSERT_EQ(ch->data_type, x::telem::FLOAT32_T);
    ASSERT_TRUE(ch->is_input);
}

TEST(MakeInput, ResolvesAutomaticChannel) {
    slave::Properties slave{
        .position = 2,
        .input_pdos = {
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 1,
             .bit_length = 16,
             .is_input = true,
             .name = "Status",
             .data_type = x::telem::UINT16_T},
        },
    };

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-003"},
        {"channel", 200},
        {"type", "automatic"},
        {"pdo", "Status"}
    };

    auto parser = x::json::Parser(j);
    const auto ch = make_input(
        ::synnax::ethercat::parse_input_channel(parser),
        slave,
        parser
    );

    ASSERT_TRUE(parser.ok());
    ASSERT_NE(ch, nullptr);
    ASSERT_EQ(ch->index, 0x6000);
    ASSERT_EQ(ch->data_type, x::telem::UINT16_T);
}

TEST(MakeInput, BindsManualChannel) {
    slave::Properties slave{.position = 4};

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-004"},
        {"channel", 300},
        {"type", "manual"},
        {"index", 0x6020},
        {"sub_index", 5},
        {"bit_length", 64},
        {"data_type", "uint64"}
    };

    auto parser = x::json::Parser(j);
    const auto ch = make_input(
        ::synnax::ethercat::parse_input_channel(parser),
        slave,
        parser
    );

    ASSERT_TRUE(parser.ok());
    ASSERT_NE(ch, nullptr);
    ASSERT_EQ(ch->index, 0x6020);
    ASSERT_EQ(ch->sub_index, 5);
    ASSERT_EQ(ch->bit_length, 64);
    ASSERT_EQ(ch->data_type, x::telem::UINT64_T);
}

TEST(MakeInput, ReportsErrorForUnknownType) {
    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-005"},
        {"channel", 400},
        {"type", "invalid"}
    };

    auto parser = x::json::Parser(j);
    ::synnax::ethercat::parse_input_channel(parser);

    ASSERT_FALSE(parser.ok());
}

TEST(Output, ResolvesAutomaticPdoFromSlaveProperties) {
    slave::Properties slave{
        .position = 7,
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 1,
             .bit_length = 16,
             .is_input = false,
             .name = "TargetPosition",
             .data_type = x::telem::INT16_T},
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 2,
             .bit_length = 8,
             .is_input = false,
             .name = "ControlWord",
             .data_type = x::telem::UINT8_T},
        },
    };

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-006"},
        {"cmd_channel", 500},
        {"state_channel", 501},
        {"type", "automatic"},
        {"pdo", "ControlWord"}
    };

    auto parser = x::json::Parser(j);
    const auto parsed = ::synnax::ethercat::parse_output_channel(parser);
    ASSERT_EQ(base(parsed).device, "dev-006");
    const auto ch = make_output(parsed, slave, parser);

    ASSERT_TRUE(parser.ok());
    ASSERT_TRUE(ch->enabled);
    ASSERT_EQ(ch->device_key, "dev-006");
    ASSERT_EQ(ch->command_key, 500);
    ASSERT_EQ(ch->state_key, 501);
    ASSERT_EQ(ch->slave_position, 7);
    ASSERT_EQ(ch->index, 0x7000);
    ASSERT_EQ(ch->sub_index, 2);
    ASSERT_EQ(ch->bit_length, 8);
    ASSERT_EQ(ch->data_type, x::telem::UINT8_T);
    ASSERT_FALSE(ch->is_input);
}

TEST(Output, ReportsErrorWhenPdoNotFound) {
    slave::Properties slave{
        .position = 1,
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 1,
             .bit_length = 16,
             .is_input = false,
             .name = "TargetPosition",
             .data_type = x::telem::INT16_T},
        },
    };

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-007"},
        {"cmd_channel", 600},
        {"type", "automatic"},
        {"pdo", "NonExistentPdo"}
    };

    auto parser = x::json::Parser(j);
    const auto parsed = ::synnax::ethercat::parse_output_channel(parser);
    make_output(parsed, slave, parser);

    ASSERT_FALSE(parser.ok());
}

TEST(Output, ParsesManualPdoAddressFromJson) {
    slave::Properties slave{.position = 9};

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-008"},
        {"cmd_channel", 700},
        {"state_channel", 701},
        {"type", "manual"},
        {"index", 0x7010},
        {"sub_index", 4},
        {"bit_length", 16},
        {"data_type", "int16"}
    };

    auto parser = x::json::Parser(j);
    const auto ch = make_output(
        ::synnax::ethercat::parse_output_channel(parser),
        slave,
        parser
    );

    ASSERT_TRUE(parser.ok());
    ASSERT_TRUE(ch->enabled);
    ASSERT_EQ(ch->device_key, "dev-008");
    ASSERT_EQ(ch->command_key, 700);
    ASSERT_EQ(ch->state_key, 701);
    ASSERT_EQ(ch->slave_position, 9);
    ASSERT_EQ(ch->index, 0x7010);
    ASSERT_EQ(ch->sub_index, 4);
    ASSERT_EQ(ch->bit_length, 16);
    ASSERT_EQ(ch->data_type, x::telem::INT16_T);
    ASSERT_FALSE(ch->is_input);
}

TEST(MakeOutput, ResolvesAutomaticChannel) {
    slave::Properties slave{
        .position = 2,
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 1,
             .bit_length = 32,
             .is_input = false,
             .name = "TargetVelocity",
             .data_type = x::telem::INT32_T},
        },
    };

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-009"},
        {"cmd_channel", 800},
        {"state_channel", 801},
        {"type", "automatic"},
        {"pdo", "TargetVelocity"}
    };

    auto parser = x::json::Parser(j);
    const auto ch = make_output(
        ::synnax::ethercat::parse_output_channel(parser),
        slave,
        parser
    );

    ASSERT_TRUE(parser.ok());
    ASSERT_NE(ch, nullptr);
    ASSERT_EQ(ch->index, 0x7000);
    ASSERT_EQ(ch->data_type, x::telem::INT32_T);
}

TEST(MakeOutput, BindsManualChannel) {
    slave::Properties slave{.position = 6};

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-010"},
        {"cmd_channel", 900},
        {"state_channel", 901},
        {"type", "manual"},
        {"index", 0x7020},
        {"sub_index", 2},
        {"bit_length", 8},
        {"data_type", "uint8"}
    };

    auto parser = x::json::Parser(j);
    const auto ch = make_output(
        ::synnax::ethercat::parse_output_channel(parser),
        slave,
        parser
    );

    ASSERT_TRUE(parser.ok());
    ASSERT_NE(ch, nullptr);
    ASSERT_EQ(ch->index, 0x7020);
    ASSERT_EQ(ch->sub_index, 2);
    ASSERT_EQ(ch->bit_length, 8);
    ASSERT_EQ(ch->data_type, x::telem::UINT8_T);
}

TEST(MakeOutput, ReportsErrorForUnknownType) {
    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-011"},
        {"cmd_channel", 1000},
        {"type", "unknown"}
    };

    auto parser = x::json::Parser(j);
    ::synnax::ethercat::parse_output_channel(parser);

    ASSERT_FALSE(parser.ok());
}

TEST(SortByPosition, SortsBySlavePositionThenByIndex) {
    slave::Properties slave1{.position = 1};
    slave::Properties slave2{.position = 2};
    slave::Properties slave3{.position = 1};

    nlohmann::json j1 = {
        {"disabled", false},
        {"device", "dev-a"},
        {"channel", 1},
        {"type", "manual"},
        {"index", 0x6020},
        {"sub_index", 1},
        {"bit_length", 16},
        {"data_type", "uint16"}
    };

    nlohmann::json j2 = {
        {"disabled", false},
        {"device", "dev-b"},
        {"channel", 2},
        {"type", "manual"},
        {"index", 0x6000},
        {"sub_index", 1},
        {"bit_length", 16},
        {"data_type", "uint16"}
    };

    nlohmann::json j3 = {
        {"disabled", false},
        {"device", "dev-c"},
        {"channel", 3},
        {"type", "manual"},
        {"index", 0x6010},
        {"sub_index", 1},
        {"bit_length", 16},
        {"data_type", "uint16"}
    };

    auto parser1 = x::json::Parser(j1);
    auto parser2 = x::json::Parser(j2);
    auto parser3 = x::json::Parser(j3);

    std::vector<std::unique_ptr<Input>> channels;
    channels.push_back(
        make_input(::synnax::ethercat::parse_input_channel(parser1), slave1, parser1)
    );
    channels.push_back(
        make_input(::synnax::ethercat::parse_input_channel(parser2), slave2, parser2)
    );
    channels.push_back(
        make_input(::synnax::ethercat::parse_input_channel(parser3), slave3, parser3)
    );

    sort_by_position(channels);

    ASSERT_EQ(channels[0]->slave_position, 1);
    ASSERT_EQ(channels[0]->index, 0x6010);
    ASSERT_EQ(channels[1]->slave_position, 1);
    ASSERT_EQ(channels[1]->index, 0x6020);
    ASSERT_EQ(channels[2]->slave_position, 2);
    ASSERT_EQ(channels[2]->index, 0x6000);
}

TEST(Channel, EnabledDefaultsToTrue) {
    slave::Properties slave{
        .position = 1,
        .input_pdos = {
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 1,
             .bit_length = 16,
             .is_input = true,
             .name = "Value",
             .data_type = x::telem::UINT16_T},
        },
    };

    nlohmann::json j = {
        {"device", "dev-012"},
        {"channel", 1100},
        {"type", "automatic"},
        {"pdo", "Value"}
    };

    auto parser = x::json::Parser(j);
    const auto ch = make_input(
        ::synnax::ethercat::parse_input_channel(parser),
        slave,
        parser
    );

    ASSERT_TRUE(parser.ok());
    ASSERT_TRUE(ch->enabled);
}

TEST(Output, StateKeyDefaultsToZero) {
    slave::Properties slave{
        .position = 1,
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 1,
             .bit_length = 16,
             .is_input = false,
             .name = "Control",
             .data_type = x::telem::UINT16_T},
        },
    };

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-013"},
        {"cmd_channel", 1200},
        {"type", "automatic"},
        {"pdo", "Control"}
    };

    auto parser = x::json::Parser(j);
    const auto ch = make_output(
        ::synnax::ethercat::parse_output_channel(parser),
        slave,
        parser
    );

    ASSERT_TRUE(parser.ok());
    ASSERT_EQ(ch->state_key, 0);
}

TEST(Input, BindRemoteInfoCopiesChannelInformation) {
    slave::Properties slave{
        .position = 1,
        .input_pdos = {
            {.pdo_index = 0x1A00,
             .index = 0x6000,
             .sub_index = 1,
             .bit_length = 16,
             .is_input = true,
             .name = "Value",
             .data_type = x::telem::UINT16_T},
        },
    };

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-014"},
        {"channel", 1300},
        {"type", "automatic"},
        {"pdo", "Value"}
    };

    auto parser = x::json::Parser(j);
    const auto ch = make_input(
        ::synnax::ethercat::parse_input_channel(parser),
        slave,
        parser
    );

    synnax::channel::Channel remote_ch;
    remote_ch.key = 1300;
    remote_ch.name = "test_channel";
    remote_ch.data_type = x::telem::UINT16_T;

    ch->bind_remote_info(remote_ch);

    ASSERT_EQ(ch->ch.key, 1300);
    ASSERT_EQ(ch->ch.name, "test_channel");
    ASSERT_EQ(ch->ch.data_type, x::telem::UINT16_T);
}

TEST(Output, BindRemoteInfoCopiesStateChannelInformation) {
    slave::Properties slave{
        .position = 1,
        .output_pdos = {
            {.pdo_index = 0x1600,
             .index = 0x7000,
             .sub_index = 1,
             .bit_length = 8,
             .is_input = false,
             .name = "Control",
             .data_type = x::telem::UINT8_T},
        },
    };

    nlohmann::json j = {
        {"disabled", false},
        {"device", "dev-015"},
        {"cmd_channel", 1400},
        {"state_channel", 1401},
        {"type", "automatic"},
        {"pdo", "Control"}
    };

    auto parser = x::json::Parser(j);
    const auto ch = make_output(
        ::synnax::ethercat::parse_output_channel(parser),
        slave,
        parser
    );

    synnax::channel::Channel state_ch;
    state_ch.key = 1401;
    state_ch.name = "state_channel";
    state_ch.data_type = x::telem::UINT8_T;

    ch->bind_remote_info(state_ch);

    ASSERT_EQ(ch->state_ch.key, 1401);
    ASSERT_EQ(ch->state_ch.name, "state_channel");
    ASSERT_EQ(ch->state_ch.data_type, x::telem::UINT8_T);
}

}
