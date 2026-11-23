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
        {"connection", {{"transport", "tcp"}, {"host", "plc"}, {"port", 502}}},
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
    ASSERT_EQ(server.requests.write_codes, WRITE_CODES_MULTIPLE);
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
    ASSERT_EQ(rtu.parity, PARITY_EVEN);
    ASSERT_EQ(rtu.unit_id, 1);
}

/// @brief it should round trip a server through JSON.
TEST(Server, RoundTrips) {
    x::json::Parser parser(valid_server());
    auto server = Server::parse(parser);
    ASSERT_NIL(parser.error());
    server.reads.push_back(
        Read{
            .register_ =
                Register{
                    .type = REGISTER_TYPE_HOLDING_REGISTER,
                    .address = 5,
                    .data_type = DATA_TYPE_FLOAT_32,
                },
            .index = 3,
            .channel = 4,
        }
    );
    server.writes.push_back(
        Write{
            .register_ = Register{.type = REGISTER_TYPE_COIL, .address = 1, .bit = 3},
            .channel = 5,
        }
    );
    x::json::Parser round(server.to_json());
    const auto parsed = Server::parse(round);
    ASSERT_NIL(round.error());
    ASSERT_EQ(std::get<TcpConnection>(parsed.connection).host, "plc");
    ASSERT_EQ(parsed.reads.at(0).register_.data_type, DATA_TYPE_FLOAT_32);
    ASSERT_EQ(parsed.reads.at(0).index, 3);
    ASSERT_EQ(parsed.reads.at(0).channel, 4);
    ASSERT_EQ(parsed.writes.at(0).register_.bit, 3);
    ASSERT_EQ(parsed.writes.at(0).channel, 5);
}
}
