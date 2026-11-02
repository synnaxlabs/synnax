// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
// Disable GCC 13 false positive warning in <regex> header
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wmaybe-uninitialized"
#endif
#include <regex>
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic pop
#endif
#include <sstream>
#include <string>
#include <unordered_map>

#include "absl/log/log.h"

#include "driver/common/scan_task.h"
#include "driver/ni/probe.h"
#include "driver/ni/scan_task.h"
#include "errors/errors.h"

namespace driver::ni {
Scanner::Scanner(
    std::shared_ptr<x::lib::Lazy<syscfg::SugaredAPI>> lib,
    std::shared_ptr<x::lib::Lazy<daqmx::SugaredAPI>> dmx,
    ScanTaskConfig cfg,
    synnax::task::Task task
):
    cfg(cfg), task(std::move(task)), lib(std::move(lib)), dmx(std::move(dmx)) {}

const auto SKIP_DEVICE_ERR = x::errors::Error("ni.skip_device", "");
const std::size_t NO_DEVICES_LOG_MULTIPLIER = 12;
const std::string MODEL_PREFIX = MAKE + " ";
const std::regex IGNORED_MODEL("(cRIO|Unknown).*");

std::pair<Device, x::errors::Error>
Scanner::parse_device(NISysCfgResourceHandle resource) {
    char property_value_buf[1024];
    Device dev;
    dev.make = MAKE;
    dev.rack = this->task.rack;
    dev.configured = false;
    NISysCfgBool is_simulated = NISysCfgBoolFalse;
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
    if (dev.model.starts_with(MODEL_PREFIX))
        dev.model = dev.model.substr(MODEL_PREFIX.size());
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
        .key = synnax::device::status_key(dev),
        .name = dev.name,
        .variant = synnax::status::VARIANT_SUCCESS,
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
    if (std::regex_match(dev.model, IGNORED_MODEL)) {
        if (this->ignored_keys.insert(dev.key).second)
            VLOG(1) << SCAN_LOG_PREFIX << "skipping " << dev.model << " (" << dev.key
                    << "): model matches an ignored pattern";
        err = SKIP_DEVICE_ERR;
    } else {
        VLOG(1) << SCAN_LOG_PREFIX << "device validated successfully: " << dev.key
                << " (model: " << dev.model << ")";
    }
    return {dev, err};
}

std::pair<std::vector<synnax::device::Device>, x::errors::Error>
Scanner::scan(const common::ScannerContext &ctx) {
    if (this->filter == nullptr) {
        if (const auto err = this->open()) {
            this->close();
            return {{}, err};
        }
    }
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
    // For each module, resolve its parent chassis key.
    std::unordered_map<std::string, std::string> parent_keys;
    for (const auto &dev: ni_devices) {
        if (dev.is_chassis || dev.connects_to_link_name.empty()) continue;
        if (auto it = link_to_chassis.find(dev.connects_to_link_name);
            it != link_to_chassis.end())
            parent_keys[dev.key] = it->second;
        else
            VLOG(1) << SCAN_LOG_PREFIX << "module " << dev.key << " connects to link '"
                    << dev.connects_to_link_name << "' but no chassis provides it";
    }

    // Stage 3: Sort chassis before modules for creation ordering, then convert.
    std::ranges::stable_sort(ni_devices, [](const ni::Device &a, const ni::Device &b) {
        return a.is_chassis > b.is_chassis;
    });

    std::vector<synnax::device::Device> devices;
    devices.reserve(ni_devices.size());
    for (auto &dev: ni_devices) {
        this->read_hardware(dev, ctx);
        auto sy_dev = dev.to_synnax();
        if (auto it = parent_keys.find(dev.key); it != parent_keys.end())
            sy_dev.parent = synnax::device::ontology_id(it->second);
        devices.push_back(std::move(sy_dev));
    }
    return {devices, close_err};
}

namespace {
/// @brief false when the stored hardware is absent, does not parse, or is incomplete.
bool has_complete_hardware(const x::json::json::object_t &properties) {
    const auto it = properties.find(HARDWARE_KEY);
    if (it == properties.end()) return false;
    x::json::Parser parser(it->second);
    const auto hardware = ::synnax::ni::Hardware::parse(parser);
    return parser.ok() && is_complete(hardware);
}
}

void Scanner::read_hardware(ni::Device &dev, const common::ScannerContext &ctx) {
    if (ctx.devices != nullptr)
        if (const auto it = ctx.devices->find(dev.key);
            it != ctx.devices->end() && has_complete_hardware(it->second.properties))
            return;
    auto [dmx, err] = this->dmx->get();
    if (!err) {
        auto [hardware, probe_err] = probe(*dmx, dev.location);
        err = probe_err;
        if (!err) {
            const bool complete = is_complete(hardware);
            dev.hardware = std::move(hardware);
            if (complete) {
                this->unread_keys.erase(dev.key);
                return;
            }
        }
    }
    std::ostringstream msg;
    msg << SCAN_LOG_PREFIX << "failed to read the ";
    if (err)
        msg << "hardware of " << dev.location << ": " << err;
    else
        msg << "default terminal configuration of some ports of " << dev.location
            << ". The next scan reads it again";
    if (this->unread_keys.insert(dev.key).second)
        LOG(WARNING) << msg.str();
    else
        VLOG(1) << msg.str();
}

x::errors::Error Scanner::stop() {
    return this->close();
}

x::errors::Error Scanner::open() {
    auto [syscfg, load_err] = this->lib->get();
    if (load_err) return load_err;
    this->syscfg = std::move(syscfg);
    if (const auto err = this->syscfg->InitializeSession(
            nullptr,
            nullptr,
            nullptr,
            NISysCfgLocaleDefault,
            NISysCfgBoolTrue,
            static_cast<unsigned int>(
                (this->cfg.rate.period() - x::telem::SECOND).milliseconds()
            ),
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

x::errors::Error Scanner::close() {
    if (this->filter != nullptr) {
        this->syscfg->CloseHandle(this->filter);
        this->filter = nullptr;
    }
    if (this->session == nullptr) return x::errors::NIL;
    const auto err = this->syscfg->CloseHandle(this->session);
    this->session = nullptr;
    return err;
}
}
