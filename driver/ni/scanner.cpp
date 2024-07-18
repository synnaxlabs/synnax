// Copyright 2024 Synnax Labs, Inc.
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
#include "driver/ni/nisyscfg_api.h"

#include "nlohmann/json.hpp"

ni::Scanner::Scanner(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
): task(task), ctx(ctx) {
    // initialize syscfg session for the scanner (TODO: Error Handling for status)
    NISysCfgStatus status = NISysCfg_OK;
    status = ni::NiSysCfgInterface::InitializeSession(
        "localhost", // target (ip, mac or dns name)
        NULL, // username (NULL for local system)
        NULL, // password (NULL for local system)
        NISysCfgLocaleDefault, // language
        NISysCfgBoolTrue,
        // force pproperties to be queried everytime rather than cached
        10000, // timeout (ms)
        NULL, // expert handle
        &this->session // session handle
    );

    if (status != NISysCfg_OK) {
        this->ok_state = false;
        LOG(ERROR) << "[ni.scanner] failed to initialize scanner for task " << this->
                task.name;
        return;
    }

    // create a filter to only identify NI devices rather than chassis and devices which are connected (which includes simulated devices)
    this->filter = NULL;
    ni::NiSysCfgInterface::CreateFilter(this->session, &this->filter);
    ni::NiSysCfgInterface::SetFilterProperty(
        this->filter,
        NISysCfgFilterPropertyIsDevice,
        NISysCfgBoolTrue
    );
    ni::NiSysCfgInterface::SetFilterProperty(
        filter,
        NISysCfgFilterPropertyIsPresent,
        NISysCfgIsPresentTypePresent
    );
    ni::NiSysCfgInterface::SetFilterProperty(
        filter,
        NISysCfgFilterPropertyIsChassis,
        NISysCfgBoolFalse
    );
    VLOG(1) << "[ni.scanner] successfully configured scanner for task " << this->task.
            name;
}

ni::Scanner::~Scanner() {
    // TODO: Error Handling
    ni::NiSysCfgInterface::CloseHandle(this->filter);
    ni::NiSysCfgInterface::CloseHandle(this->resources_handle);
    ni::NiSysCfgInterface::CloseHandle(this->session);
}

void ni::Scanner::scan() {
    if(!this->ok_state) return;
    NISysCfgResourceHandle resource = NULL;

    // first find hardware
    auto err = ni::NiSysCfgInterface::FindHardware(
        this->session, NISysCfgFilterModeAll,
        this->filter, NULL,
        &this->resources_handle
    );
    if (err != NISysCfg_OK) {
        this->ok_state = false;
        LOG(ERROR) << "[ni.scanner] failed to find hardware for task " << this->task.
                name;
        return;
    }

    // Now iterate through found devices and get requested properties
    devices["devices"] = json::array();

    while (ni::NiSysCfgInterface::NextResource(
               this->session,
               this->resources_handle,
               &resource
           ) == NISysCfg_OK) {
        auto device = get_device_properties(resource);
        devices["devices"].push_back(device);
    }
}


json ni::Scanner::get_device_properties(NISysCfgResourceHandle resource) {
    json device;

    char propertyValue[1024] = "";

    ni::NiSysCfgInterface::GetResourceProperty(
        resource,
        NISysCfgResourcePropertySerialNumber,
        propertyValue
    );
    device["serial_number"] = propertyValue;

    ni::NiSysCfgInterface::GetResourceProperty(
        resource,
        NISysCfgResourcePropertyProductName,
        propertyValue
    );
    std::string model = propertyValue;
    model = model.substr(3, model.size());
    device["model"] = model;

    ni::NiSysCfgInterface::GetResourceIndexedProperty(
        resource,
        NISysCfgIndexedPropertyExpertUserAlias,
        0,
        propertyValue
    );
    device["location"] = propertyValue;

    ni::NiSysCfgInterface::GetResourceIndexedProperty(
        resource,
        NISysCfgIndexedPropertyExpertResourceName,
        0,
        propertyValue
    );
    std::string rsrc_name = propertyValue;
    rsrc_name = rsrc_name.substr(1, rsrc_name.size() - 2);
    device["resource_name"] = rsrc_name;

    double temp;
    ni::NiSysCfgInterface::GetResourceProperty(
        resource,
        NISysCfgResourcePropertyCurrentTemp,
        &temp
    );
    device["temperature"] = temp;

    NISysCfgBool isSimulated;
    ni::NiSysCfgInterface::GetResourceProperty(
        resource,
        NISysCfgResourcePropertyIsSimulated,
        &isSimulated
    );
    if (isSimulated) {
        device["is_simulated"] = true;
        device["key"] = device["resource_name"];
    } else {
        device["is_simulated"] = false;
        device["key"] = device["serial_number"];
    }
    return device;
}

void ni::Scanner::create_devices() {
    for (auto &device: devices["devices"]) {
        // first  try to rereive the device and if found, do not create a new device, simply continue
        auto [retrieved_device, err] = this->ctx->client->hardware.retrieveDevice(
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
        if (this->ctx->client->hardware.createDevice(new_device) != freighter::NIL)
            LOG(ERROR) << "[ni.scanner] failed to create device " << device["model"] <<
                    " with key " << device["key"] << " for task " << this->task.name;
        VLOG(1) << "[ni.scanner] successfully created device " << device["model"] <<
                " with key " << device["key"] << " for task " << this->task.name;
    }
}

bool ni::Scanner::ok() {
    return this->ok_state;
}

json ni::Scanner::get_devices() {
    return devices;
}
