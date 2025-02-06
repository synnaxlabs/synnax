// Copyright 2025 Synnax Labs, Inc.
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


///////////////////////////////////////////////////////////////////////////////////
//                                ScannerTask                                    //
///////////////////////////////////////////////////////////////////////////////////
labjack::ScannerTask::ScannerTask(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    std::shared_ptr<labjack::DeviceManager> device_manager
) : ctx(std::move(ctx)), task(std::move(task)), device_manager(device_manager) {
    this->devices["devices"] = nlohmann::json::array();
    this->breaker.start();
    this->thread = std::make_unique<std::thread>(&ScannerTask::run, this);
}

std::unique_ptr<task::Task> labjack::ScannerTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    std::shared_ptr<labjack::DeviceManager> device_manager
) {
    return std::make_unique<ScannerTask>(ctx, task, device_manager);
}


void labjack::ScannerTask::exec(task::Command &cmd) {
    if (cmd.type == SCAN_CMD_TYPE) {
        this->scan();
        return this->create_devices();
    } else if (cmd.type == STOP_CMD_TYPE) {
        return this->stop();
    }
}

void labjack::ScannerTask::scan() {
    scan_for(LJM_dtANY, LJM_ctANY);
}

void labjack::ScannerTask::scan_for(int device_type, int connection_type) {
    int device_types[LJM_LIST_ALL_SIZE];
    int connection_types[LJM_LIST_ALL_SIZE];
    int serial_numbers[LJM_LIST_ALL_SIZE];
    int ip_addresses[LJM_LIST_ALL_SIZE];
    int num_found = 0; {
        check_err(labjack::locked::LJM_ListAll_wrapped(
            device_type,
            connection_type,
            &num_found,
            device_types,
            connection_types,
            serial_numbers,
            ip_addresses
        ));
    }

    for (int i = 0; i < num_found; i++) {
        nlohmann::json device;
        device["device_type"] = NumberToDeviceType(device_types[i]);
        device["connection_type"] = NumberToConnectionType(connection_types[i]);
        device["serial_number"] = serial_numbers[i];
        device["key"] = device["serial_number"];
        device["failed_to_create"] = false;
        if (device_keys.find(device["key"].get<int>()) == device_keys.end()) {
            devices["devices"].push_back(device);
            device_keys.insert(device["key"].get<int>());
        }
    }
}

void labjack::ScannerTask::create_devices() {
    for (auto &device: devices["devices"]) {
        if (device["failed_to_create"] == true) continue;
        std::string key = std::to_string(device["key"].get<int>());
        auto [retrieved_device, err] = this->ctx->client->hardware.retrieve_device(key);

        if (!err) {
            VLOG(1) << "[labjack.scanner] device with key: " << device["key"] << " found";
            continue;
        }

        // in order to differentiate same model devices, we append the last 4 digits of the serial number
        auto ser_num = std::to_string(device["serial_number"].get<int>());
        auto last_four = ser_num.length() >= 4 ? ser_num.substr(ser_num.length() - 4) : ser_num;
        auto name = device["device_type"].get<std::string>() + "-" + last_four;

        auto new_device = synnax::Device(
            key,
            name, // name
            synnax::taskKeyRack(this->task.key), // rack key
            device["connection_type"].get<std::string>(), // location
            std::to_string(device["serial_number"].get<int>()),
            "LabJack",
            device["device_type"].get<std::string>(),
            device.dump()
        );

        if (this->ctx->client->hardware.create_device(new_device) != freighter::NIL) {
            LOG(ERROR) << "[labjack.scanner] failed to create device with key: " << device["key"];
            device["failed_to_create"] = true;
        } else {
            LOG(INFO) << "[labjack.scanner] successfully created device with key: " << device["key"];
        }
        std::string serial_number = std::to_string(device["serial_number"].get<int>());
        int handle = this->device_manager->get_device_handle(serial_number);
    }
}

void labjack::ScannerTask::stop() {
    this->breaker.stop();
    if (this->thread != nullptr && this->thread->joinable() && std::this_thread::get_id() != this->thread->get_id())
        this->thread->join();
}

void labjack::ScannerTask::run() {
    auto scan_cmd = task::Command{task.key, SCAN_CMD_TYPE, {}};
    int i = 0;
    while (this->breaker.running()) {
        i += 1;
        this->breaker.wait_for(this->scan_rate.period().chrono());
        if (i % this->tcp_scan_multiplier == 0)
            this->scan_for(LJM_dtANY, LJM_ctTCP);
        this->scan_for(LJM_dtANY, LJM_ctUSB);
        this->create_devices();
    }
}

labjack::ScannerTask::~ScannerTask() {
    if (this->thread != nullptr && this->thread->joinable() && std::this_thread::get_id() != this->thread->get_id())
        this->thread->join();
}

json labjack::ScannerTask::get_devices() {
    return devices;
}

int labjack::ScannerTask::check_err(const int err) {
    // First check if it is LJME_AUTO_IPS_FILE_NOT_FOUND as this is a known
    // bug on the LJM Library when no devices are connected
    if (err == LJME_AUTO_IPS_FILE_NOT_FOUND || err == LJME_AUTO_IPS_FILE_INVALID) return 0;
    return labjack::check_err_internal(
        err,
        "",
        "scanner",
        this->ctx,
        this->ok_state,
        this->task.key
    );
}

bool labjack::ScannerTask::ok() const {
    return this->ok_state;
}
