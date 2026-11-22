// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/ethercat/slave/json.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::ethercat::slave {
namespace {
x::json::json valid_slave() {
    return x::json::json{
        {"key", "22222222-2222-4222-8222-222222222222"},
        {"rack", 1},
        {"name", "EL3102"},
        {"vendor_id", 2},
        {"product_code", 0x0C1E3052},
        {"revision", 0x00110000},
        {"master", "33333333-3333-4333-8333-333333333333"},
    };
}
}

/// @brief it should parse an unconfigured slave and apply the defaults.
TEST(Slave, Parses) {
    x::json::Parser parser(valid_slave());
    const auto parsed = Slave::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.vendor_id, 2);
    ASSERT_EQ(parsed.serial, 0);
    ASSERT_EQ(parsed.identifier, "");
    ASSERT_FALSE(parsed.disabled);
    ASSERT_TRUE(parsed.pdos.inputs.empty());
    ASSERT_TRUE(parsed.read_bindings.empty());
}

/// @brief it should reject a slave without a master.
TEST(Slave, RequiresMaster) {
    auto j = valid_slave();
    j.erase("master");
    x::json::Parser parser(j);
    Slave::parse(parser);
    ASSERT_FALSE(parser.ok());
    ASSERT_EQ(parser.errors->size(), 1);
    ASSERT_EQ(parser.errors->at(0)["path"], "master");
}

/// @brief it should round trip PDO entries and channel bindings through JSON.
TEST(Slave, RoundTrips) {
    x::json::Parser in(valid_slave());
    auto s = Slave::parse(in);
    ASSERT_NIL(in.error());
    s.pdos.inputs.push_back({
        .name = "Value",
        .index = 0x6000,
        .sub_index = 0x11,
        .bit_length = 16,
        .data_type = x::telem::INT16_T,
    });
    s.read_bindings.push_back({.index = 0x6000, .sub_index = 0x11, .channel = 7});
    s.write_bindings.push_back({.index = 0x7000, .sub_index = 1, .command = 8});
    x::json::Parser parser(s.to_json());
    const auto parsed = Slave::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.pdos.inputs.size(), 1);
    ASSERT_EQ(parsed.pdos.inputs[0].data_type, x::telem::INT16_T);
    ASSERT_EQ(parsed.read_bindings[0].channel, 7);
    ASSERT_EQ(parsed.write_bindings[0].command, 8);
    ASSERT_EQ(parsed.write_bindings[0].state, 0);
}
}
