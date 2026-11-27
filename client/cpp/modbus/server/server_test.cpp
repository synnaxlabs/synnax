// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/modbus/server/json.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::modbus::server {
namespace {
x::json::json valid_server() {
    return x::json::json{
        {"key", "22222222-2222-4222-8222-222222222222"},
        {"rack", 1},
        {"name", "PLC"},
        {"connection", {{"transport", "tcp"}, {"host", "plc"}}},
    };
}
}

/// @brief it should parse a TCP server and apply the defaults.
TEST(Server, AppliesTCPDefaults) {
    x::json::Parser parser(valid_server());
    const auto server = Server::parse(parser);
    ASSERT_NIL(parser.error());
    const auto &tcp = std::get<TcpConnection>(server.connection);
    ASSERT_EQ(tcp.host, "plc");
    ASSERT_EQ(tcp.port, 502);
    ASSERT_EQ(tcp.unit_id, 255);
    ASSERT_EQ(tcp.timeout, x::telem::SECOND);
    ASSERT_EQ(server.request_policy.write_function, WRITE_FUNCTION_MULTIPLE);
    ASSERT_EQ(server.request_policy.max_write_registers, 123);
    ASSERT_TRUE(server.reads.empty());
    ASSERT_TRUE(server.writes.empty());
    ASSERT_FALSE(server.status.has_value());
}

/// @brief it should parse an RTU server and apply its line defaults.
TEST(Server, AppliesRTUDefaults) {
    auto j = valid_server();
    j["connection"] = {{"transport", "rtu"}, {"serial_port", "COM3"}};
    x::json::Parser parser(j);
    const auto server = Server::parse(parser);
    ASSERT_NIL(parser.error());
    const auto &rtu = std::get<RtuConnection>(server.connection);
    ASSERT_EQ(rtu.baud_rate, BaudRate::B19200);
    ASSERT_EQ(rtu.parity, PARITY_EVEN_);
    ASSERT_EQ(rtu.stop_bits, StopBits::One);
    ASSERT_EQ(rtu.unit_id, 1);
}

/// @brief it should round trip a server through JSON.
TEST(Server, RoundTrips) {
    x::json::Parser parser(valid_server());
    auto server = Server::parse(parser);
    ASSERT_NIL(parser.error());
    HoldingRegisterLocation holding;
    holding.address = 5;
    WholeValue whole_value;
    whole_value.data_type = x::telem::NUMERIC_DATA_TYPE_FLOAT_32;
    whole_value.words_swapped = true;
    holding.value = whole_value;
    server.reads.push_back(Read{.location = holding, .index = 3, .channel = 4});
    HoldingRegisterWriteLocation bit;
    bit.address = 1;
    BitWriteValue bit_value;
    bit_value.bit = 3;
    bit.value = bit_value;
    server.writes.push_back(Write{.location = bit, .channel = 5});
    x::json::Parser round(server.to_json());
    const auto parsed = Server::parse(round);
    ASSERT_NIL(round.error());
    ASSERT_EQ(std::get<TcpConnection>(parsed.connection).host, "plc");
    const auto &read = std::get<HoldingRegisterLocation>(parsed.reads.at(0).location);
    ASSERT_EQ(read.address, 5);
    const auto &whole = std::get<WholeValue>(read.value);
    ASSERT_EQ(whole.data_type, x::telem::NUMERIC_DATA_TYPE_FLOAT_32);
    ASSERT_EQ(whole.words_swapped, true);
    ASSERT_FALSE(whole.bytes_swapped.has_value());
    ASSERT_EQ(parsed.reads.at(0).index, 3);
    ASSERT_EQ(parsed.reads.at(0).channel, 4);
    const auto &write = std::get<HoldingRegisterWriteLocation>(
        parsed.writes.at(0).location
    );
    ASSERT_EQ(std::get<BitWriteValue>(write.value).bit, 3);
    ASSERT_EQ(parsed.writes.at(0).channel, 5);
}

/// @brief it should keep the field group of an update through JSON.
TEST(Update, RoundTripsGroup) {
    IdentityUpdate identity;
    identity.rack = 1;
    identity.name = "PLC";
    TcpConnection tcp;
    tcp.host = "plc";
    tcp.port = 502;
    identity.connection = tcp;
    const auto j = to_json(Update(identity));
    ASSERT_EQ(j["group"], "identity");
    x::json::Parser parser(j);
    const auto parsed = parse_update(parser);
    ASSERT_NIL(parser.error());
    const auto &got = std::get<IdentityUpdate>(parsed);
    ASSERT_EQ(got.name, "PLC");
    ASSERT_EQ(std::get<TcpConnection>(got.connection).port, 502);
    ASSERT_EQ(got.request_policy.max_read_registers, 125);
}

/// @brief it should apply the defaults of a configuration update.
TEST(Update, AppliesConfigurationDefaults) {
    x::json::Parser parser(x::json::json{{"group", "configuration"}});
    const auto parsed = parse_update(parser);
    ASSERT_NIL(parser.error());
    const auto &got = std::get<ConfigurationUpdate>(parsed);
    ASSERT_TRUE(got.reads.empty());
    ASSERT_TRUE(got.writes.empty());
}

/// @brief it should reject an update with an unknown group.
TEST(Update, RejectsUnknownGroup) {
    x::json::Parser parser(x::json::json{{"group", "other"}});
    parse_update(parser);
    ASSERT_FALSE(parser.ok());
}
}
