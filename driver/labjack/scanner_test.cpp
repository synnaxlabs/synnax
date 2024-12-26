// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.tx

#pragma once

#include <stdio.h>

#include "driver/testutil/testutil.h"
#include "client/cpp/synnax.h"
#include "driver/labjack/scanner.h"

#include "nlohmann/json.hpp"
#include <include/gtest/gtest.h>

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                          Functional Tests                                                    //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
TEST(LabjackScannerTests, test_valid_scan) {
    std::cout << "Running test_valid_scan" << std::endl;
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto task = synnax::Task(
        "my_task",
        "labjackScanner",
        ""
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);
    labjack::ScannerTask scanner = labjack::ScannerTask(mockCtx, task);
    // sleep for 3 seconds
    std::this_thread::sleep_for(std::chrono::seconds(3));

    nlohmann::json devices = scanner.get_devices();
    // print json
    std::cout << devices.dump(4) << std::endl;
}
