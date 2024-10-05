// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <utility>
#include "nlohmann/json.hpp"

#include "scanner.h"
#include "glog/logging.h"
#include "driver/config/config.h"

using namespace labjack;

///////////////////////////////////////////////////////////////////////////////////
//                                ScannerTask                                    //
///////////////////////////////////////////////////////////////////////////////////
ScannerTask::ScannerTask (
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
) : ctx(std::move(ctx)), task(std::move(task)) {
    this->devices["devices"] = nlohmann::json::array();
}

std::unique_ptr<task::Task> ScannerTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    return std::make_unique<ScannerTask>(ctx, task);
}

void ScannerTask::exec(task::Command &cmd) {
    if (cmd.type == SCAN_CMD_TYPE) return scan();
    LOG(ERROR) << "[labjack] Scanner received unknown command type: " << cmd.type;
}

void ScannerTask::scan() {
    int DeviceType = LJM_dtANY;
    int ConnectionType = LJM_ctANY;

    int aDeviceTypes[LJM_LIST_ALL_SIZE];
    int aConnectionTypes[LJM_LIST_ALL_SIZE];
    int aSerialNumbers[LJM_LIST_ALL_SIZE];
    int aIPAddresses[LJM_LIST_ALL_SIZE];
    int NumFound = 0;

    // Get the device keys
    int err = LJM_ListAll(
            DeviceType,
            ConnectionType,
            &NumFound,
            aDeviceTypes,
            aConnectionTypes,
            aSerialNumbers,
            aIPAddresses
        );
    ErrorCheck(
            err,
            "LJM_ListAll with device type: %s, connection type: %s",
            NumberToDeviceType(DeviceType),
            NumberToConnectionType(ConnectionType)
       );
    LOG(INFO) << "[labjack.scanner] Found " << NumFound << " devices";

    for(int i= 0; i < NumFound; i++) {
        nlohmann::json device;
        device["device_type"] = NumberToDeviceType(aDeviceTypes[i]);
        device["connection_type"] = NumberToConnectionType(aConnectionTypes[i]);
        device["serial_number"] = aSerialNumbers[i];
        devices["devices"].push_back(device);
    }

    LOG(INFO) << "devices json: "  << devices.dump(4);

}
