// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/ethercat/master/json.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::ethercat::master {
namespace {
x::json::json valid_master() {
    return x::json::json{
        {"key", "22222222-2222-4222-8222-222222222222"},
        {"rack", 1},
        {"name", "EtherCAT eth0"},
        {"interface", "eth0"},
    };
}
}

/// @brief it should parse a master with an empty bus by default.
TEST(Master, Parses) {
    x::json::Parser parser(valid_master());
    const auto parsed = Master::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.interface, "eth0");
    ASSERT_TRUE(parsed.slaves.empty());
    ASSERT_FALSE(parsed.status.has_value());
}

/// @brief it should reject a master without an interface.
TEST(Master, RequiresInterface) {
    auto j = valid_master();
    j.erase("interface");
    x::json::Parser parser(j);
    Master::parse(parser);
    ASSERT_FALSE(parser.ok());
    ASSERT_EQ(parser.errors->size(), 1);
    ASSERT_EQ(parser.errors->at(0)["path"], "interface");
}

/// @brief it should round trip a master and its bus order through JSON.
TEST(Master, RoundTrips) {
    Master m;
    m.key = x::uuid::create();
    m.rack = 1;
    m.name = "EtherCAT eth1";
    m.interface = "eth1";
    m.slaves = {x::uuid::create(), x::uuid::create()};
    x::json::Parser parser(m.to_json());
    const auto parsed = Master::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(parsed.key, m.key);
    ASSERT_EQ(parsed.slaves, m.slaves);
}
}
