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

#include "driver/ni/capability_mapper.h"
#include "driver/ni/scan_task.h"

// DAQmx capability queries are not available on macOS
#ifndef __APPLE__
#include "driver/ni/daqmx/nidaqmx.h"
#endif

ni::Scanner::Scanner(
    const std::shared_ptr<::syscfg::SugaredAPI> &syscfg,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    ScanTaskConfig cfg,
    synnax::Task task
):
    cfg(std::move(cfg)), task(std::move(task)), syscfg(syscfg), dmx(dmx) {}

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
    VLOG(1) << "Processing device resource: " << resource;
    VLOG(1) << "Device Rack: " << dev.rack;

    if (!is_simulated) {
        VLOG(1) << "Physical device detected";
        if (const auto err = this->syscfg->GetResourceProperty(
                resource,
                NISysCfgResourcePropertySerialNumber,
                property_value_buf
            )) {
            LOG(WARNING) << "Physical device missing serial number, skipping: "
                         << err.message();
            return {Device(), SKIP_DEVICE_ERR};
        }
        dev.key = property_value_buf;
        VLOG(1) << "Physical device serial number: " << dev.key;
    } else {
        VLOG(1) << "Simulated device detected";
    }

    if (const auto err = this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertyProductName,
            property_value_buf
        )) {
        LOG(WARNING) << "Device missing product name, skipping: " << err.message();
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
        LOG(WARNING) << "Device missing user alias, using empty location: "
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
        LOG(WARNING) << "Device missing resource name, skipping: " << err.message();
        return {Device(), SKIP_DEVICE_ERR};
    }
    VLOG(1) << "Resource name: " << property_value_buf;
    dev.resource_name = property_value_buf;
    if (dev.resource_name.size() > 2)
        dev.resource_name = dev.resource_name.substr(1, dev.resource_name.size() - 2);
    if (is_simulated) dev.key = dev.resource_name;

    // Query device capabilities using DAQmx (only if DAQmx is available)
    // Note: DAQmx is not available on macOS, so we skip capability querying there
#ifndef __APPLE__
    if (this->dmx != nullptr) {
        // AI capabilities
        int32 ai_types[50]; // Max AI measurement types
        auto ai_status = DAQmxGetDevAISupportedMeasTypes(
            dev.resource_name.c_str(),
            ai_types,
            50
        );
        if (ai_status >= 0) { // Positive value = number of types
            dev.supported_ai_types.assign(ai_types, ai_types + ai_status);
            VLOG(1) << "Device " << dev.key << " supports " << ai_status << " AI types";
        }

        // AO capabilities
        int32 ao_types[10]; // Max AO output types
        auto ao_status = DAQmxGetDevAOSupportedOutputTypes(
            dev.resource_name.c_str(),
            ao_types,
            10
        );
        if (ao_status >= 0) {
            dev.supported_ao_types.assign(ao_types, ao_types + ao_status);
            VLOG(1) << "Device " << dev.key << " supports " << ao_status << " AO types";
        }

        // CI capabilities
        int32 ci_types[20]; // Max CI measurement types
        auto ci_status = DAQmxGetDevCISupportedMeasTypes(
            dev.resource_name.c_str(),
            ci_types,
            20
        );
        if (ci_status >= 0) {
            dev.supported_ci_types.assign(ci_types, ci_types + ci_status);
            VLOG(1) << "Device " << dev.key << " supports " << ci_status << " CI types";
        }

        // CO capabilities
        int32 co_types[10]; // Max CO output types
        auto co_status = DAQmxGetDevCOSupportedOutputTypes(
            dev.resource_name.c_str(),
            co_types,
            10
        );
        if (co_status >= 0) {
            dev.supported_co_types.assign(co_types, co_types + co_status);
            VLOG(1) << "Device " << dev.key << " supports " << co_status << " CO types";
        }
    }
#endif

    dev.status = synnax::DeviceStatus{
        .key = dev.key,
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
        LOG(WARNING) << "Device ignored by filter: " << dev.key
                     << " (model: " << dev.model << ")";
        err = SKIP_DEVICE_ERR;
    } else {
        VLOG(1) << "Device validated successfully: " << dev.key
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

synnax::Device ni::Device::to_synnax() {
    // Convert DAQmx capability constants to Synnax channel type strings
    auto synnax_ai_types = capability::convert_ai_types_to_synnax(
        this->supported_ai_types
    );
    auto synnax_ao_types = capability::convert_ao_types_to_synnax(
        this->supported_ao_types
    );
    auto synnax_ci_types = capability::convert_ci_types_to_synnax(
        this->supported_ci_types
    );
    auto synnax_co_types = capability::convert_co_types_to_synnax(
        this->supported_co_types
    );

    auto dev = synnax::Device(
        this->key,
        this->name,
        this->rack,
        this->location,
        this->make,
        this->model,
        nlohmann::to_string(
            json{
                {"is_simulated", this->is_simulated},
                {"resource_name", this->resource_name},
                {"supportedAITypes", synnax_ai_types},
                {"supportedAOTypes", synnax_ao_types},
                {"supportedCITypes", synnax_ci_types},
                {"supportedCOTypes", synnax_co_types}
            }
        )
    );
    dev.status = this->status;
    return dev;
}
