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
#include "driver/labjack/util.h"

using namespace labjack;

///////////////////////////////////////////////////////////////////////////////////
//                                ScannerTask                                    //
///////////////////////////////////////////////////////////////////////////////////
ScannerTask::ScannerTask (
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
) : ctx(std::move(ctx)), task(std::move(task)) {
    this->devices["devices"] = nlohmann::json::array();
    this->breaker.start();
    this->thread = std::make_unique<std::thread>(&ScannerTask::run, this);
}

std::unique_ptr<task::Task> ScannerTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    return std::make_unique<ScannerTask>(ctx, task);
}

void ScannerTask::exec(task::Command &cmd) {
    if (cmd.type == SCAN_CMD_TYPE) {
        this->scan();
        return this->create_devices();
    } else if (cmd.type == STOP_CMD_TYPE){
        return this->stop();
    }
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

    int err;
    {
        std::lock_guard<std::mutex> lock(labjack::device_mutex);
        err = LJM_ListAll(
                DeviceType,
                ConnectionType,
                &NumFound,
                aDeviceTypes,
                aConnectionTypes,
                aSerialNumbers,
                aIPAddresses
        );
    }
    char err_name[LJM_MAX_NAME_SIZE];
    LJM_ErrorToString(err, err_name);
    LOG(ERROR) << "LJM_ListAll error: " << err_name;
    // TODO: deal with error checks which will cause exit
    ErrorCheck(
            err,
            "LJM_ListAll with device type: %s, connection type: %s",
            NumberToDeviceType(DeviceType),
            NumberToConnectionType(ConnectionType)
       );
    for(int i= 0; i < NumFound; i++) {
        nlohmann::json device;
        device["device_type"] = NumberToDeviceType(aDeviceTypes[i]);
        device["connection_type"] = NumberToConnectionType(aConnectionTypes[i]);
        device["serial_number"] = aSerialNumbers[i];
        device["key"] = device["serial_number"];
        device["failed_to_create"] = false;
        // check if device is already in set, else pushback
        if(device_keys.find(device["key"].get<int>()) == device_keys.end()) {
            devices["devices"].push_back(device);
            device_keys.insert(device["key"].get<int>());
        }
    }

    LOG(INFO) << "devices json: "  << devices.dump(4); // TODO: remove once labjack dev is done
}

void ScannerTask::create_devices() {
    for(auto &device : devices["devices"]) {
        if(device["failed_to_create"] == true) continue;
        std::string key = std::to_string(device["key"].get<int>());
        auto [retrieved_device, err] = this->ctx->client->hardware.retrieveDevice(key);

        if(!err) {
            VLOG(1) << "[labjack.scanner] device with key: " << device["key"] << " found";
            continue;
        }

        auto new_device = synnax::Device(
            key,
            device["device_type"].get<std::string>(),           // name
            synnax::taskKeyRack(this->task.key),                // rack key
            device["connection_type"].get<std::string>(),       // location
            std::to_string(device["serial_number"].get<int>()),
            "LabJack",
            device["device_type"].get<std::string>(),
            device.dump()
        );

        LOG(INFO) << "[labjack.scanner] creating device with key: " << device["key"];
        if (this->ctx->client->hardware.createDevice(new_device) != freighter::NIL) {
            LOG(ERROR) << "[labjack.scanner] failed to create device with key: " << device["key"];
            device["failed_to_create"] = true;
        } else {
            LOG(INFO) << "[labjack.scanner] successfully created device with key: " << device["key"];
        }
    }
}

void ScannerTask::stop(){
    this->breaker.stop();
    if (this->thread != nullptr && this->thread->joinable() && std::this_thread::get_id() != this->thread->get_id())
        this->thread->join();
}

void ScannerTask::run(){
    auto scan_cmd = task::Command{task.key, SCAN_CMD_TYPE, {}};
    while (this->breaker.running()) {
        this->breaker.waitFor(this->scan_rate.period().chrono());
        this->exec(scan_cmd);
    }
}

ScannerTask::~ScannerTask() {
    if (this->thread != nullptr && this->thread->joinable() && std::this_thread::get_id() != this->thread->get_id())
        this->thread->join();
}

json ScannerTask::get_devices() {
    return devices;
}