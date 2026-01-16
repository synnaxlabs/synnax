// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <unordered_map>

#include "driver/ni/errors.h"
#include "driver/ni/scan_task.h"

ni::Scanner::Scanner(
    const std::shared_ptr<::syscfg::SugaredAPI> &syscfg,
    ScanTaskConfig cfg,
    synnax::Task task
):
    cfg(std::move(cfg)), task(std::move(task)), syscfg(syscfg) {}

const auto SKIP_DEVICE_ERR = xerrors::Error("ni.skip_device", "");
const std::size_t NO_DEVICES_LOG_MULTIPLIER = 12;

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

    // Get the link name this device connects to (modules connect to a chassis link)
    if (const auto link_err = this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertyConnectsToLinkName,
            property_value_buf
        )) {
        VLOG(1) << SCAN_LOG_PREFIX << "device " << dev.key
                << " has no ConnectsToLinkName: " << link_err.message();
    } else {
        dev.connects_to_link_name = property_value_buf;
        LOG(INFO) << SCAN_LOG_PREFIX << "device " << dev.key << " (" << dev.model
                  << ") connects to link: '" << dev.connects_to_link_name << "'";
    }

    // Get the link name this device provides (chassis provide a link for modules)
    if (const auto link_err = this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertyProvidesLinkName,
            property_value_buf
        )) {
        VLOG(1) << SCAN_LOG_PREFIX << "device " << dev.key
                << " has no ProvidesLinkName: " << link_err.message();
    } else {
        dev.provides_link_name = property_value_buf;
        LOG(INFO) << SCAN_LOG_PREFIX << "device " << dev.key << " (" << dev.model
                  << ") provides link: '" << dev.provides_link_name << "'";
    }

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
    std::vector<ni::Device> ni_devices;
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
        if (err.matches(ni::END_OF_ENUM)) {
            if (ctx.count % NO_DEVICES_LOG_MULTIPLIER == 0)
                LOG(INFO) << SCAN_LOG_PREFIX << "no devices found.";
            return {devices, xerrors::NIL};
        }
        return {devices, err};
    }

    // First pass: collect all devices and build a map of link providers (chassis)
    std::unordered_map<std::string, std::string> link_to_device_key;
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

        // If this device provides a link (i.e., it's a chassis), record the mapping
        if (!dev.provides_link_name.empty()) {
            link_to_device_key[dev.provides_link_name] = dev.key;
            VLOG(1) << SCAN_LOG_PREFIX << "device " << dev.key
                    << " provides link: " << dev.provides_link_name;
        }

        ni_devices.push_back(std::move(dev));
        this->syscfg->CloseHandle(curr_resource);
    }

    // Log the link provider map for debugging
    LOG(INFO) << SCAN_LOG_PREFIX << "link provider map has " << link_to_device_key.size()
              << " entries";
    for (const auto &[link, key] : link_to_device_key) {
        LOG(INFO) << SCAN_LOG_PREFIX << "  link '" << link << "' -> device " << key;
    }

    // Second pass: resolve parent relationships and convert to synnax devices
    for (auto &dev : ni_devices) {
        if (!dev.connects_to_link_name.empty()) {
            auto it = link_to_device_key.find(dev.connects_to_link_name);
            if (it != link_to_device_key.end()) {
                dev.parent_device = it->second;
                LOG(INFO) << SCAN_LOG_PREFIX << "device " << dev.key << " (" << dev.model
                          << ") parent resolved to: " << dev.parent_device;
            } else {
                LOG(WARNING) << SCAN_LOG_PREFIX << "device " << dev.key << " (" << dev.model
                             << ") connects to link '" << dev.connects_to_link_name
                             << "' but no provider found";
            }
        }
        devices.push_back(dev.to_synnax());
    }

    LOG(INFO) << SCAN_LOG_PREFIX << "scan complete: " << devices.size() << " devices found";

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
            (this->cfg.scan_rate.period() - telem::SECOND).milliseconds(),
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
    return xerrors::NIL;
}
