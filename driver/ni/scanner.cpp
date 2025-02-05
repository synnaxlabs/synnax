// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <algorithm>

#include "driver/ni/ni.h"
#include "nisyscfg/nisyscfg_api.h"

#include "nlohmann/json.hpp"

ni::Scanner::Scanner(
    const std::shared_ptr<SysCfg> &syscfg,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) : syscfg(syscfg), task(task), ctx(ctx) {
    // initialize syscfg session for the scanner (TODO: Error Handling for status)
    NISysCfgStatus status = NISysCfg_OK;
    status = syscfg->InitializeSession(
        "localhost", // target (ip, mac or dns name)
        nullptr, // username (NULL for local system)
        nullptr, // password (NULL for local system)
        NISysCfgLocaleDefault, // language
        NISysCfgBoolTrue,
        // force properties to be queried everytime rather than cached
        10000, // timeout (ms)
        nullptr, // expert handle
        &this->session // session handle
    );

    if (status != NISysCfg_OK) {
        log_err("failed to initialize scanner");
        return;
    }

    // create a filter to only identify NI devices rather than chassis and devices which are connected (which includes simulated devices)
    this->filter = nullptr;
    syscfg->CreateFilter(this->session, &this->filter);
    syscfg->SetFilterProperty(
        this->filter,
        NISysCfgFilterPropertyIsDevice,
        NISysCfgBoolTrue
    );
    this->syscfg->SetFilterProperty(
        this->filter,
        NISysCfgFilterPropertyIsPresent,
        NISysCfgIsPresentTypePresent
    );
    this->syscfg->SetFilterProperty(
        this->filter,
        NISysCfgFilterPropertyIsChassis,
        NISysCfgBoolFalse
    );
    VLOG(1) << "[ni.scanner] successfully configured scanner for task " << this->task.
            name;

    this->devices["devices"] = json::array();
}

void ni::Scanner::set_scan_thread(const std::shared_ptr<std::thread> &scan_thread) {
    this->scan_thread = scan_thread;
}

void ni::Scanner::join_scan_thread() const {
    if (this->scan_thread && this->scan_thread->joinable()) this->scan_thread->join();
}

ni::Scanner::~Scanner() {
    if (this->scan_thread && scan_thread->joinable()) scan_thread->join();
    this->syscfg->CloseHandle(this->filter);
    this->syscfg->CloseHandle(this->session);
}

void ni::Scanner::scan() {
    if (!this->ok_state) return;
    NISysCfgResourceHandle resource = nullptr;

    auto err = this->syscfg->FindHardware(
        this->session, NISysCfgFilterModeAll,
        this->filter, nullptr,
        &this->resources_handle
    );
    if (err != NISysCfg_OK) return log_err("failed to find hardware");

    while (this->syscfg->NextResource(
               this->session,
               this->resources_handle,
               &resource
           ) == NISysCfg_OK) {
        auto device = get_device_properties(resource);
        if (device["key"] != "" && device_keys.find(device["key"]) == device_keys.
            end()) {
            device["failed_to_create"] = false;
            devices["devices"].push_back(device);
            device_keys.insert(device["key"]);
        }
        this->syscfg->CloseHandle(resource);
    }
    this->syscfg->CloseHandle(this->resources_handle);
}

json ni::Scanner::get_device_properties(NISysCfgResourceHandle resource) {
    json device;

    char propertyValue[1024] = "";

    int status = this->syscfg->GetResourceProperty(
        resource,
        NISysCfgResourcePropertySerialNumber,
        propertyValue
    );
    if (status != NISysCfg_OK) log_err("failed to get serial number");
    device["serial_number"] = propertyValue;

    status = this->syscfg->GetResourceProperty(
        resource,
        NISysCfgResourcePropertyProductName,
        propertyValue
    );
    if (status != NISysCfg_OK) log_err("failed to get product name");
    std::string model = propertyValue;
    if (model.size() > 3) model = model.substr(3);
    device["model"] = model;

    status = this->syscfg->GetResourceIndexedProperty(
        resource,
        NISysCfgIndexedPropertyExpertUserAlias,
        0,
        propertyValue
    );
    if (status != NISysCfg_OK) log_err("failed to get location");
    device["location"] = propertyValue;

    status = this->syscfg->GetResourceIndexedProperty(
        resource,
        NISysCfgIndexedPropertyExpertResourceName,
        0,
        propertyValue
    );
    if (status != NISysCfg_OK) log_err("failed to get resource name");
    std::string rsrc_name = propertyValue;
    if (rsrc_name.size() > 2) rsrc_name = rsrc_name.substr(1, rsrc_name.size() - 2);
    else log_err("resource name too short to extract name");
    device["resource_name"] = rsrc_name;

    double temp = 0;
    // status = this->syscfg->GetResourceProperty(
    //     resource,
    //     NISysCfgResourcePropertyCurrentTemp,
    //     &temp
    // );
    // if (status != NISysCfg_OK) log_err("failed to get currentTemp");
    device["temperature"] = temp;

    NISysCfgBool isSimulated;
    status = this->syscfg->GetResourceProperty(
        resource,
        NISysCfgResourcePropertyIsSimulated,
        &isSimulated
    );
    if (status != NISysCfg_OK) log_err("failed to get isSimulated");
    device["is_simulated"] = isSimulated ? true : false;
    device["key"] = isSimulated ? device["resource_name"] : device["serial_number"];

    return device;
}

void ni::Scanner::create_devices() {
    if (!this->ok_state) return;
    for (auto &device: devices["devices"]) {
        // If model is not found or failed to create previously, skip
        if (device["model"] == "" || device["failed_to_create"] == true) continue;
        // first try to retrieve the device and if found, do not create a new device,
        // simply continue
        auto [retrieved_device, err] = this->ctx->client->hardware.retrieve_device(
            device["key"]);
        if (!err) {
            VLOG(1) << "[ni.scanner] device " << device["model"] << " and key " <<
                    device["key"] << "at location: " << device["location"] <<
                    " found for task " << this->task.name;
            continue;
        }
        auto new_device = synnax::Device(
            device["key"].get<std::string>(), // key
            device["model"].get<std::string>(), // name
            synnax::taskKeyRack(this->task.key), // rack key
            device["location"].get<std::string>(), // location
            device["serial_number"].get<std::string>(), // serial number
            "NI", // make
            device["model"].get<std::string>(), // model
            device.dump() // device properties
        );
        if (this->ctx->client->hardware.create_device(new_device) != freighter::NIL) {
            LOG(ERROR) << "[ni.scanner] failed to create device " << device["model"] <<
                    " with key " << device["key"] << " for task " << this->task.name;
            device["failed_to_create"] = true;
        }

        VLOG(1) << "[ni.scanner] successfully created device " << device["model"] <<
                " with key " << device["key"] << " for task " << this->task.name;
    }
}

bool ni::Scanner::ok() const { return this->ok_state; }

json ni::Scanner::get_devices() {
    if (!this->ok_state) return json::array();
    return devices;
}

void ni::Scanner::log_err(std::string err_msg) {
    LOG(ERROR) << "[ni.scanner] " << err_msg;
    json j = {
        {"error", err_msg}
    };
    this->ctx->set_state({
        .task = this->task.key,
        .variant = "error",
        .details = j
    });
    this->ok_state = false;
    LOG(ERROR) << "[ni.scanner] scanner in error state. Disabling.";
}
