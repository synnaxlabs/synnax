//
// Created by Synnax on 4/1/2024.
//

#include <include/gtest/gtest.h>
#include "driver/ni/ni.h"
#include "client/cpp/synnax.h"
#include <stdio.h>
#include "nlohmann/json.hpp"
#include "driver/testutil/testutil.h"


TEST(NiScannerTests, testGetDevices){
    LOG(INFO) << "test_ni_scanner: "; //<< std::endl;
    // create properties json
    nlohmann::json config;
    config["properties"] = nlohmann::json::array();
    config["properties"].push_back("SerialNumber");
    config["properties"].push_back("DeviceName");

    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto task = synnax::Task(
        "my_task",
        "NI_digitalRead",
        to_string(config)
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
}