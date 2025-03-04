// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <memory>

/// external
#include "nlohmann/json.hpp"

/// module
#include "x/cpp/xjson/xjson.h"

///internal
#include "driver/labjack/scan_task.h"
#include "driver/labjack/ljm/LJM_Utilities.h"

namespace labjack {

ScanTask::ScanTask(
    const std::shared_ptr<task::Context> &ctx,
    synnax::Task task,
    ScanTaskConfig cfg,
    std::shared_ptr<ljm::DeviceManager> device_manager
) : task(std::move(task)),
    cfg(std::move(cfg)),
    ctx(ctx),
    device_manager(std::move(device_manager)) {
    state.key = task.key;
}

std::pair<std::unique_ptr<task::Task>, xerrors::Error> ScanTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    std::shared_ptr<ljm::DeviceManager> device_manager
) {
    auto parser = xjson::Parser(task.config);
    auto cfg = ScanTaskConfig(parser);
    if (parser.error()) return {nullptr, parser.error()};
    auto tsk = std::make_unique<ScanTask>(ctx, task, cfg, device_manager);
    if (cfg.enabled) tsk->start();
    return {std::move(tsk), xerrors::NIL};
}

xerrors::Error ScanTask::scan_for(const int device_type, const int connection_type) {
    int device_types[LJM_LIST_ALL_SIZE];
    int connection_types[LJM_LIST_ALL_SIZE];
    int serial_numbers[LJM_LIST_ALL_SIZE];
    int ip_addresses[LJM_LIST_ALL_SIZE];
    int num_found = 0;

    if (const auto err = device_manager->ListAll(
        device_type,
        connection_type,
        &num_found,
        device_types,
        connection_types,
        serial_numbers,
        ip_addresses
    )) return err;

    for (int i = 0; i < num_found; i++) {
        const auto serial_str = std::to_string(serial_numbers[i]);
        const auto device_type_str = std::string(NumberToDeviceType(device_types[i]));
        const auto conn_type_str = std::string(NumberToConnectionType(connection_types[i]));
        
        // Create base synnax device
        auto last_four = serial_str.length() >= 4 ? 
            serial_str.substr(serial_str.length() - 4) : serial_str;
        auto name = device_type_str + "-" + last_four;

        auto sy_dev = synnax::Device(
            serial_str,
            name,
            synnax::task_key_rack(this->task.key),
            conn_type_str,
            serial_str,
            "LabJack",
            device_type_str,
            ""  // Properties will be set in Device constructor
        );

        // Create extended device
        auto dev = Device(
            sy_dev,
            serial_numbers[i],
            device_type_str,
            conn_type_str
        );

        this->devices[dev.key] = dev;
    }

    return xerrors::NIL;
}

xerrors::Error ScanTask::update_remote() {
    for (const auto &[key, device] : devices) {
        auto [retrieved_device, err] = this->ctx->client->hardware.retrieve_device(key);
        if (err.matches(xerrors::NOT_FOUND)) {
            auto sy_dev = device.to_synnax();
            if (const auto c_err = this->ctx->client->hardware.create_device(sy_dev))
                return c_err;
        } else if (err)
            return err;
    }
    return xerrors::NIL;
}

xerrors::Error ScanTask::scan() {
    if (const auto err = scan_for(LJM_dtANY, LJM_ctUSB)) return err;
    if (const auto err = scan_for(LJM_dtANY, LJM_ctTCP)) return err;
    return update_remote();
}

void ScanTask::start() {
    if (!this->breaker.start()) return;
    this->thread = std::thread(&ScanTask::run, this);
}

void ScanTask::stop(bool will_reconfigure) {
    if (!this->breaker.stop()) return;
    this->thread.join();
}

void ScanTask::exec(task::Command &cmd) {
    this->state.key = cmd.key;
    if (cmd.type == "stop") return this->stop(false);
    
    xerrors::Error err = xerrors::NIL;
    if (cmd.type == "start") this->start();
    else if (cmd.type == "scan") err = this->scan();
    
    if (!err) return;
    this->state.variant = "error";
    this->state.details["message"] = err.message();
    this->ctx->set_state(this->state);
}

void ScanTask::run() {
    this->state.variant = "success";
    this->state.details["message"] = "scan task started";
    this->ctx->set_state(this->state);
    int tcp_counter = 0;
    while (this->breaker.running()) {
        tcp_counter++;
        if (const auto err = scan_internal(tcp_counter)) {
            this->state.variant = "warning";
            this->state.details["message"] = err.message();
            this->ctx->set_state(this->state);
            break;
        }
        this->breaker.wait_for(this->cfg.rate.period());
    }
    this->state.variant = "success";
    this->state.details["message"] = "scan task stopped";
    this->ctx->set_state(this->state);
}
}
