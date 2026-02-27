// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <regex>
#include <string>
#include <unordered_map>

#include "driver/ni/scan_task.h"
#include "errors/errors.h"

namespace driver::ni {
Scanner::Scanner(
    const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
    ScanTaskConfig cfg,
    synnax::task::Task task
):
    cfg(std::move(cfg)), task(std::move(task)), syscfg(syscfg) {}

const auto SKIP_DEVICE_ERR = x::errors::Error("ni.skip_device", "");
const std::size_t NO_DEVICES_LOG_MULTIPLIER = 12;

std::pair<Device, x::errors::Error>
Scanner::parse_device(NISysCfgResourceHandle resource) const {
    char property_value_buf[1024];
    Device dev;
    dev.make = MAKE;
    dev.rack = synnax::task::rack_key_from_task_key(this->task.key);
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

    dev.status = synnax::device::Status{
        .key = dev.status_key(),
        .name = dev.name,
        .variant = x::status::VARIANT_SUCCESS,
        .message = "Device present",
        .time = x::telem::TimeStamp::now(),
        .details = synnax::device::StatusDetails{
            .rack = dev.rack,
            .device = dev.key,
        }
    };

    // Chassis/link properties (non-fatal; missing = empty/false).
    NISysCfgBool is_chassis_val = NISysCfgBoolFalse;
    if (!this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertyIsChassis,
            &is_chassis_val
        ))
        dev.is_chassis = is_chassis_val;
    if (!this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertyConnectsToLinkName,
            property_value_buf
        ))
        dev.connects_to_link_name = property_value_buf;
    if (!this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertyProvidesLinkName,
            property_value_buf
        ))
        dev.provides_link_name = property_value_buf;

    VLOG(1) << SCAN_LOG_PREFIX << "device " << dev.key
            << " is_chassis=" << dev.is_chassis
            << " connects_to=" << dev.connects_to_link_name
            << " provides=" << dev.provides_link_name;

    auto err = x::errors::NIL;
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

std::pair<std::vector<synnax::device::Device>, x::errors::Error>
Scanner::scan(const common::ScannerContext &ctx) {
    NISysCfgEnumResourceHandle resources = nullptr;
    NISysCfgResourceHandle curr_resource = nullptr;

    auto err = this->syscfg->FindHardware(
        this->session,
        NISysCfgFilterModeAll,
        this->filter,
        nullptr,
        &resources
    );
    if (err) {
        if (err.matches(errors::END_OF_ENUM)) {
            if (ctx.count % NO_DEVICES_LOG_MULTIPLIER == 0)
                LOG(INFO) << SCAN_LOG_PREFIX << "no devices found.";
            return {{}, x::errors::NIL};
        }
        return {{}, err};
    }

    // Stage 1: Parse NISysCfg resources into ni::Device objects.
    std::vector<ni::Device> ni_devices;
    while (true) {
        if (const auto next_err = this->syscfg->NextResource(
                this->session,
                resources,
                &curr_resource
            ))
            break;
        auto [dev, parse_err] = this->parse_device(curr_resource);
        this->syscfg->CloseHandle(curr_resource);
        if (parse_err) continue;
        ni_devices.push_back(std::move(dev));
    }
    auto close_err = this->syscfg->CloseHandle(resources);
    if (err.skip(SKIP_DEVICE_ERR)) return {{}, err};

    // Stage 2: Resolve parent links (O(N) total, O(1) per device).
    // Build map: provides_link_name -> device key (chassis only).
    std::unordered_map<std::string, std::string> link_to_chassis;
    for (const auto &dev: ni_devices)
        if (dev.is_chassis && !dev.provides_link_name.empty())
            link_to_chassis[dev.provides_link_name] = dev.key;
    // For each module, look up its parent chassis.
    for (auto &dev: ni_devices) {
        if (dev.is_chassis || dev.connects_to_link_name.empty()) continue;
        if (auto it = link_to_chassis.find(dev.connects_to_link_name);
            it != link_to_chassis.end())
            dev.parent_device = it->second;
        else
            VLOG(1) << SCAN_LOG_PREFIX << "module " << dev.key << " connects to link '"
                    << dev.connects_to_link_name << "' but no chassis provides it";
    }

    // Stage 3: Sort chassis before modules for creation ordering, then convert.
    std::stable_sort(
        ni_devices.begin(),
        ni_devices.end(),
        [](const ni::Device &a, const ni::Device &b) {
            return a.is_chassis > b.is_chassis;
        }
    );

    std::vector<synnax::device::Device> devices;
    devices.reserve(ni_devices.size());
    for (auto &dev: ni_devices)
        devices.push_back(dev.to_synnax());
    return {devices, close_err};
}

x::errors::Error Scanner::stop() {
    this->syscfg->CloseHandle(this->filter);
    return this->syscfg->CloseHandle(this->session);
}

x::errors::Error Scanner::start() {
    if (const auto err = this->syscfg->InitializeSession(
            nullptr,
            nullptr,
            nullptr,
            NISysCfgLocaleDefault,
            NISysCfgBoolTrue,
            (this->cfg.scan_rate.period() - x::telem::SECOND).milliseconds(),
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
            NISysCfgFilterPropertyIsNIProduct,
            NISysCfgBoolTrue
        ))
        return err;
    return x::errors::NIL;
}
}
