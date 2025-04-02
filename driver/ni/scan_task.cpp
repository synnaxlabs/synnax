// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <regex>
#include <string>

/// internal
#include "driver/ni/scan_task.h"

ni::Scanner::Scanner(
    const std::shared_ptr<::syscfg::SugaredAPI> &syscfg,
    ScanTaskConfig cfg,
    synnax::Task task
):
    cfg(std::move(cfg)), task(std::move(task)), syscfg(syscfg) {}

const auto SKIP_DEVICE_ERR = xerrors::Error("ni.skip_device", "");

std::pair<ni::Device, xerrors::Error>
ni::Scanner::parse_device(NISysCfgResourceHandle resource) const {
    char property_value_buf[1024];
    Device dev;
    dev.make = MAKE;
    dev.rack = synnax::task_key_rack(this->task.key);
    dev.configured = false;

    NISysCfgBool is_simulated;
    if (const auto err = this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertyIsSimulated,
            &is_simulated
        ))
        return {dev, err};
    dev.is_simulated = is_simulated;

    if (!is_simulated) {
        if (const auto err = this->syscfg->GetResourceProperty(
                resource,
                NISysCfgResourcePropertySerialNumber,
                property_value_buf
            ))
            return {Device(), err};
        dev.key = property_value_buf;
    }

    if (const auto err = this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertyProductName,
            property_value_buf
        ))
        return {Device(), err};
    dev.model = property_value_buf;
    if (dev.model.size() > 3) dev.model = dev.model.substr(3);
    dev.name = MAKE + " " + dev.model;

    if (const auto err = this->syscfg->GetResourceIndexedProperty(
            resource,
            NISysCfgIndexedPropertyExpertUserAlias,
            0,
            property_value_buf
        ))
        return {Device(), err};
    dev.location = property_value_buf;

    if (const auto err = this->syscfg->GetResourceIndexedProperty(
            resource,
            NISysCfgIndexedPropertyExpertResourceName,
            0,
            property_value_buf
        ))
        return {dev, err};
    dev.resource_name = property_value_buf;
    if (dev.resource_name.size() > 2)
        dev.resource_name = dev.resource_name.substr(1, dev.resource_name.size() - 2);
    if (is_simulated) dev.key = dev.resource_name;

    auto err = xerrors::NIL;
    if (this->cfg.should_ignore(dev.model)) err = SKIP_DEVICE_ERR;
    return {dev, err};
}

std::pair<DeviceState, xerrors::Error>
ni::Scanner::get_device_state(NISysCfgResourceHandle resource, const ni::Device& device) const {
    DeviceState state;
    state.key = device.key;
    state.rack = device.rack;
    
    // Get presence status
    NISysCfgIsPresentType is_present;
    if (const auto err = this->syscfg->GetResourceProperty(
        resource,
        NISysCfgResourcePropertyIsPresent,
        &is_present
    )) {
        return {DeviceState{}, err};
    }
    state.is_connected = (is_present == NISysCfgIsPresentTypePresent);
    
    // Get firmware status for operational check
    NISysCfgFirmwareStatus firmware_status;
    if (const auto err = this->syscfg->GetResourceProperty(
        resource,
        NISysCfgResourcePropertyFirmwareStatus,
        &firmware_status
    )) {
        state.error_code = err.code();
        state.is_operational = false;
    } else {
        state.is_operational = (firmware_status == NISysCfgFirmwareStatusInstalled);
        state.error_code = 0;
    }

    // Try to get temperature if available
    double temp;
    if (const auto err = this->syscfg->GetResourceProperty(
        resource,
        NISysCfgResourcePropertyCurrentTemp,
        &temp
    ); !err) {
        state.temperature_celsius = temp;
    }

    // Set variant and details based on state
    if (state.is_operational && state.is_connected) {
        state.variant = "operational";
        state.details = {{"temperature_celsius", state.temperature_celsius}};
    } else if (!state.is_connected) {
        state.variant = "disconnected";
        state.details = {{"error_code", state.error_code}};
    } else {
        state.variant = "error";
        state.details = {
            {"error_code", state.error_code},
            {"temperature_celsius", state.temperature_celsius}
        };
    }

    return {state, xerrors::NIL};
}

std::pair<common::ScanResult, xerrors::Error>
ni::Scanner::scan(const common::ScannerContext &ctx) {
    common::ScanResult result;
    NISysCfgEnumResourceHandle resources = nullptr;
    NISysCfgResourceHandle curr_resource = nullptr;
    
    auto err = this->syscfg->FindHardware(
        this->session,
        NISysCfgFilterModeAll,
        this->filter,
        nullptr,
        &resources
    );
    if (err) return {result, err};

    while (true) {
        if (const auto next_err = this->syscfg->NextResource(
            this->session,
            resources,
            &curr_resource
        )) break;

        auto [dev, parse_err] = this->parse_device(curr_resource);
        if (parse_err) {
            if (parse_err == SKIP_DEVICE_ERR) continue;
            this->syscfg->CloseHandle(curr_resource);
            continue;
        }

        auto [state, state_err] = this->get_device_state(curr_resource, dev);
        if (!state_err) {
            result.devices.push_back(dev.to_synnax());
            result.states.push_back(state.to_synnax());
        }

        this->syscfg->CloseHandle(curr_resource);
    }

    auto close_err = this->syscfg->CloseHandle(resources);
    if (err.skip(SKIP_DEVICE_ERR)) return {result, err};
    return {result, close_err};
}

xerrors::Error ni::Scanner::stop() {
    this->syscfg->CloseHandle(this->filter);
    return this->syscfg->CloseHandle(this->session);
}

xerrors::Error ni::Scanner::start() {
    if (const auto err = this->syscfg->InitializeSession(
            nullptr, // target (ip, mac or dns name)
            nullptr, // username (NULL for local system)
            nullptr, // password (NULL for local system)
            NISysCfgLocaleDefault, // language
            NISysCfgBoolTrue,
            // force properties to be queried everytime rather than cached
            (this->cfg.rate.period() - telem::SECOND).milliseconds(),
            nullptr, // expert handle
            &this->session // session handle
        ))
        return err;

    if (const auto err = this->syscfg->CreateFilter(this->session, &this->filter))
        return err;
    if (const auto err = this->syscfg->SetFilterProperty(
            this->filter,
            NISysCfgFilterPropertyIsDevice,
            NISysCfgBoolTrue
        ))
        return err;
    if (const auto err = this->syscfg->SetFilterProperty(
            this->filter,
            NISysCfgFilterPropertyIsPresent,
            NISysCfgIsPresentTypePresent
        ))
        return err;
    if (const auto err = this->syscfg->SetFilterProperty(
            this->filter,
            NISysCfgFilterPropertyIsChassis,
            NISysCfgBoolFalse
        ))
        return err;
    if (const auto err = this->syscfg->SetFilterProperty(
            this->filter,
            NISysCfgFilterPropertyIsNIProduct,
            NISysCfgBoolTrue
        ))
        return err;
    return xerrors::NIL;
}
