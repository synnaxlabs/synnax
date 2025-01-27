// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/synnax.h"
#include "driver/ni/ni.h"
#include "driver/testutil/testutil.h"

#include <gtest/gtest.h>
#include "glog/logging.h"
#include "nidaqmx/nidaqmx_prod.h"
#include "nisyscfg/nisyscfg_prod.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;

/* 
Devices Identifiers in NI MAX

Dev1 : NI USB-6289 (simulated device)
Dev2 : NI USB-9211A (simulated device)
Dev3 : NI USB-9219 (simulated device)
Dev4 : NI USB-6000 (physical device)
Dev5 : NI USB-9234 (simulated device)

PXI1Slot2 : NI PXIe-4302 (simulated device)
PXI1Slot3 : NI PXIe-4357 (simulated device)
*/

TEST(scanner_tests, test_valid_scan) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto task = synnax::Task("scanner_task", "niScanner", "");
    auto mockCtx = std::make_shared<task::MockContext>(client);

    auto [sys_cfg, load_err] = SysCfgProd::load();
    ASSERT_FALSE(load_err) << load_err.message();

    ni::Scanner scanner(sys_cfg, mockCtx, task);
    
    // First scan
    scanner.scan();
    ASSERT_TRUE(scanner.ok());

    if (scanner.ok()) {
        json devices = scanner.get_devices();
        VLOG(1) << "Number of devices: " << devices["devices"].size();
        VLOG(1) << "Devices: " << devices.dump(4);
    } else {
        FAIL() << "Scanner failed to retrieve devices";
    }

    // Second scan
    scanner.scan();
    ASSERT_TRUE(scanner.ok());

    if (scanner.ok()) {
        json devices = scanner.get_devices();
        VLOG(1) << "Number of devices: " << devices["devices"].size();
        VLOG(1) << "Devices: " << devices.dump(4);
    } else {
        FAIL() << "Scanner failed to retrieve devices";
    }
}
