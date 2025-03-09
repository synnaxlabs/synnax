// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// external
#include "gtest/gtest.h"

/// module
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/opc/util/util.h"
#include "driver/opc/mock/server.h"

// TEST(ConnTest, testBasicConn) {
//     mock::Server server{mock::ServerConfig()};
//     server.start();
//     std::this_thread::sleep_for(std::chrono::seconds(3));
//     util::ConnectionConfig cfg;
//     cfg.endpoint = "opc.tcp://0.0.0.0:4840";
//
//     auto v = ASSERT_NIL_P(util::connect(cfg, "opc"));
//     ASSERT_NE(v, nullptr);
// }
