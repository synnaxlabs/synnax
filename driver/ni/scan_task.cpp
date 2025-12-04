// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <regex>
#include <string>

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
    dev.rack = synnax::rack_key_from_task_key(this->task.key);
    dev.configured = false;
    NISysCfgBool is_simulated;
    if (const auto err = this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertyIsSimulated,
            &is_simulated
        ))
        return {dev, err};
    dev.is_simulated = is_simulated;
    VLOG(1) << SCAN_LOG_PREFIX << "processing device resource: " << resource;
    VLOG(1) << SCAN_LOG_PREFIX << "device rack: " << dev.rack;

    if (!is_simulated) {
        VLOG(1) << SCAN_LOG_PREFIX << "physical device detected";
        if (const auto err = this->syscfg->GetResourceProperty(
                resource,
                NISysCfgResourcePropertySerialNumber,
                property_value_buf
            )) {
            LOG(WARNING) << SCAN_LOG_PREFIX
                         << "physical device missing serial number, skipping: "
                         << err.message();
            return {Device(), SKIP_DEVICE_ERR};
        }
        dev.key = property_value_buf;
        VLOG(1) << SCAN_LOG_PREFIX << "physical device serial number: " << dev.key;
    } else
        VLOG(1) << SCAN_LOG_PREFIX << "simulated device detected";

    if (const auto err = this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertyProductName,
            property_value_buf
        )) {
        LOG(WARNING) << SCAN_LOG_PREFIX
                     << "device missing product name, skipping: " << err.message();
        return {Device(), SKIP_DEVICE_ERR};
    }
    dev.model = property_value_buf;
    if (dev.model.size() > 3) dev.model = dev.model.substr(3);
    dev.name = MAKE + " " + dev.model;

    if (const auto err = this->syscfg->GetResourceIndexedProperty(
            resource,
            NISysCfgIndexedPropertyExpertUserAlias,
            0,
            property_value_buf
        )) {
        LOG(WARNING) << SCAN_LOG_PREFIX
                     << "device missing user alias, using empty location: "
                     << err.message();
        return {Device(), SKIP_DEVICE_ERR};
    }
    dev.location = property_value_buf;

    if (const auto err = this->syscfg->GetResourceIndexedProperty(
            resource,
            NISysCfgIndexedPropertyExpertResourceName,
            0,
            property_value_buf
        )) {
        LOG(WARNING) << SCAN_LOG_PREFIX
                     << "device missing resource name, skipping: " << err.message();
        return {Device(), SKIP_DEVICE_ERR};
    }
    VLOG(1) << SCAN_LOG_PREFIX << "resource name: " << property_value_buf;
    dev.resource_name = property_value_buf;
    if (dev.resource_name.size() > 2)
        dev.resource_name = dev.resource_name.substr(1, dev.resource_name.size() - 2);
    if (is_simulated) dev.key = dev.resource_name;

    dev.status = synnax::DeviceStatus{
        .key = dev.status_key(),
        .name = dev.name,
        .variant = status::variant::SUCCESS,
        .message = "Device present",
        .time = telem::TimeStamp::now(),
        .details = synnax::DeviceStatusDetails{
            .rack = dev.rack,
            .device = dev.key,
        }
    };

    auto err = xerrors::NIL;
    if (this->cfg.should_ignore(dev.model)) {
        LOG(WARNING) << SCAN_LOG_PREFIX << "device ignored by filter: " << dev.key
                     << " (model: " << dev.model << ")";
        err = SKIP_DEVICE_ERR;
    } else {
        VLOG(1) << SCAN_LOG_PREFIX << "device validated successfully: " << dev.key
                << " (model: " << dev.model << ")";
    }
    return {dev, err};
}

std::pair<std::vector<synnax::Device>, xerrors::Error>
ni::Scanner::scan(const common::ScannerContext &ctx) {
    std::vector<synnax::Device> devices;
    NISysCfgEnumResourceHandle resources = nullptr;
    NISysCfgResourceHandle curr_resource = nullptr;

    auto err = this->syscfg->FindHardware(
        this->session,
        NISysCfgFilterModeAll,
        this->filter,
        nullptr,
        &resources
    );
    if (err) return {devices, err};

    while (true) {
        if (const auto next_err = this->syscfg->NextResource(
                this->session,
                resources,
                &curr_resource
            ))
            break;

        auto [dev, parse_err] = this->parse_device(curr_resource);
        if (parse_err) {
            if (parse_err == SKIP_DEVICE_ERR) continue;
            this->syscfg->CloseHandle(curr_resource);
            continue;
        }
        devices.push_back(dev.to_synnax());
        this->syscfg->CloseHandle(curr_resource);
    }

    auto close_err = this->syscfg->CloseHandle(resources);
    if (err.skip(SKIP_DEVICE_ERR)) return {devices, err};
    return {devices, close_err};
}

xerrors::Error ni::Scanner::stop() {
    this->syscfg->CloseHandle(this->filter);
    return this->syscfg->CloseHandle(this->session);
}

xerrors::Error ni::Scanner::start() {
    if (const auto err = this->syscfg->InitializeSession(
            nullptr,
            nullptr,
            nullptr,
            NISysCfgLocaleDefault,
            NISysCfgBoolTrue,
            (this->cfg.rate.period() - telem::SECOND).milliseconds(),
            nullptr,
            &this->session
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
