// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <include/gtest/gtest.h>
#include "nlohmann/json.hpp"
#include "driver/opc/scanner.h"
#include "client/cpp/synnax.h"

using json = nlohmann::json;

const synnax::Config test_client_config = {
    "localhost",
    9090,
    "synnax",
    "seldon"
};

synnax::Synnax new_test_client() {
    return synnax::Synnax(test_client_config);
}

TEST(OPCUAScnannerTest, testScannerCmdParseOnlyEdnpoint) {
   json cmd = {
       {"endpoint", "opc.tcp://localhost:4840"},
   };
   json err = {
       {"errors", std::vector<json>()}
   };
   bool ok = true;
   auto parsedScanCmd = opcua::ScannerScanCommand(cmd, err, ok);
   EXPECT_TRUE(ok);
   EXPECT_EQ(parsedScanCmd.endpoint, "opc.tcp://localhost:4840");
   EXPECT_EQ(parsedScanCmd.username, "");
   EXPECT_EQ(parsedScanCmd.password, "");
}

TEST(OPCUAScannerTest, testScannerCmdParseEndpointUsernamePassword) {
   json cmd = {
       {"endpoint", "opc.tcp://localhost:4840"},
       {"username", "user"},
       {"password", "password"}
   };
   json err = {
       {"errors", std::vector<json>()}
   };
   bool ok = true;
   auto parsedScanCmd = opcua::ScannerScanCommand(cmd, err, ok);
   EXPECT_TRUE(ok);
   EXPECT_EQ(parsedScanCmd.endpoint, "opc.tcp://localhost:4840");
   EXPECT_EQ(parsedScanCmd.username, "user");
   EXPECT_EQ(parsedScanCmd.password, "password");
}

TEST(OPCUAScannerTest, testScannerCmdParseNoEndpoint) {
   json cmd = {
       {"username", "user"},
       {"password", "password"}
   };
   json err = {
       {"errors", std::vector<json>()}
   };
   bool ok = true;
   auto parsedScanCmd = opcua::ScannerScanCommand(cmd, err, ok);
   EXPECT_FALSE(ok);
   auto field_err = err["errors"][0];
   EXPECT_EQ(field_err["path"], "endpoint");
   EXPECT_EQ(field_err["message"], "required");
}


TEST(OPCUAScannerTest, testScannerCmdParseNoAccessControl) {
   json cmd = {
           {"connection", {
               "client_certificate" "",
                       "client_private_key": "",
                       "endpoint": "opc.tcp://0.0.0.0:4840",
                       "password": "",
                       "security_mode": "None",
                       "security_policy": "None",
                       "server_certificate": "",
                       "username": ""
           }},
           {"node_id", ""}
   }
   json err = {
       {"errors", std::vector<json>()}
   };
   bool ok = true;
   auto parsedScanCmd = opcua::ScannerScanCommand(cmd, err, ok);
   EXPECT_FALSE(ok);
   auto field_err = err["errors"][0];
   EXPECT_EQ(field_err["path"], "username");
   EXPECT_EQ(field_err["message"], "required");
}