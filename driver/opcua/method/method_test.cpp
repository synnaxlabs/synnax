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

#include "driver/opcua/method/method.h"
#include "driver/opcua/testutil/server.h"
#include "driver/opcua/testutil/testutil.h"

namespace driver::opcua::method {
/// @brief it should read the declared input arguments of each method, none for a method
/// without the property, and nothing for a method that does not exist.
TEST(MethodTest, ReadsInputArguments) {
    auto server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4865;
    server_cfg.pump = testutil::Pump{};
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());
    connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4865";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    connection::Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(cfg, "[test] "));
    const auto inputs = ASSERT_NIL_P(read_inputs(
        conn,
        {
            testutil::PUMP_CONFIGURE,
            testutil::PUMP_FAIL,
            UA_NODEID_STRING(1, const_cast<char *>("Missing")),
        }
    ));
    ASSERT_EQ(inputs.size(), 3);
    ASSERT_TRUE(inputs[0].has_value());
    const auto &configure = inputs[0].value();
    ASSERT_EQ(configure.size(), 2);
    EXPECT_EQ(configure[0].name, "mode");
    EXPECT_EQ(configure[0].data_type.to_string(), "NS=0;I=12");
    EXPECT_EQ(configure[1].name, "speed");
    EXPECT_EQ(configure[1].data_type.to_string(), "NS=0;I=11");
    ASSERT_TRUE(inputs[1].has_value());
    EXPECT_TRUE(inputs[1].value().empty());
    EXPECT_FALSE(inputs[2].has_value());
    server.stop();
}

/// @brief it should leave the inputs of a method unknown when the browse stops before
/// it lists the children of the method.
TEST(MethodTest, LeavesCutShortMethodUnknown) {
    auto server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4865;
    server_cfg.pump = testutil::Pump{};
    server_cfg.generated_variables = connection::MAX_CHILDREN + 1;
    server_cfg.max_nodes_per_browse = 1;
    server_cfg.max_references_per_node = 2500;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());
    connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4865";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    connection::Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(cfg, "[test] "));
    const auto inputs = ASSERT_NIL_P(read_inputs(
        conn,
        {UA_NODEID_STRING(1, const_cast<char *>("Generated")), testutil::PUMP_CONFIGURE}
    ));
    ASSERT_EQ(inputs.size(), 2);
    EXPECT_FALSE(inputs[1].has_value());
    server.stop();
}
}
