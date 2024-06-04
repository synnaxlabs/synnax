// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Synnax on 4/1/2024.
//

#include <include/gtest/gtest.h>
#include "driver/ni/ni.h"
#include "client/cpp/synnax.h"
#include <stdio.h>
#include "nlohmann/json.hpp"
#include "driver/testutil/testutil.h"

//TODO: add asserts to eliminate manual checking of terminal output

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                          Functional Tests                                                    //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
TEST(NiScannerTests, test_valid_scan){
    LOG(INFO) << "test_ni_scanner: "; //<< std::endl;
    // create properties json
    
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto task = synnax::Task(
        "my_task",
        "niScanner",
        ""
    );
    auto mockCtx = std::make_shared<task::MockContext>(client);

    //create a scanner
    ni::Scanner scanner = ni::Scanner(mockCtx, task);
    scanner.scan();
    if(scanner.ok()){
        nlohmann::json devices = scanner.getDevices();
        // print size of devices
        std::cout << "Number of devices: " << devices["devices"].size() << std::endl;
        std::cout << devices.dump(4) << std::endl;
    } else {
        std::cout << "Scanner failed to retreive devices" << std::endl;
    }

    // scan a second time
    scanner.scan();
    if(scanner.ok()){
        nlohmann::json devices = scanner.getDevices();
        // print size of devices
        std::cout << "Number of devices: " << devices["devices"].size() << std::endl;
        std::cout << devices.dump(4) << std::endl;
    } else {
        std::cout << "Scanner failed to retreive devices" << std::endl;
    }
}

