// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#include "x/cpp/os/os.h"

#ifndef SYNNAX_NILINUXRT
#include "driver/modbus/modbus.h"
#endif
#include "driver/arc/arc.h"
#include "driver/ethercat/ethercat.h"
#include "driver/http/http.h"
#include "driver/modbus/names.h"
#include "driver/rack/factories.h"
#include "driver/rack/rack.h"
#include "driver/rack/status/status.h"
#include "driver/version/version.h"

namespace driver::rack {
bool Config::integration_enabled(const std::string &i) const {
    return std::ranges::find(integrations, i) != integrations.end();
}

void configure_opcua(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, opcua::INTEGRATION_NAME, [&config]() {
        auto [host, _] = x::os::get_hostname();
        return std::make_unique<opcua::Factory>(
            std::make_shared<opcua::certificate::Store>(opcua::certificate::StoreConfig{
                .dir = config.state_dir / OPCUA_STATE_DIR / "pki",
                .host = host,
            })
        );
    });
}

constexpr auto MISMATCH_DESCRIPTION = "Install the Driver release that matches the "
                                      "Core, then restart the Driver.";

std::string mismatch_message(const std::string &core_version) {
    return "Driver " + std::string(SYNNAX_DRIVER_VERSION) + " cannot work with Core " +
           core_version;
}

/// @brief adds the rack status, which reports an error when the Core runs another
/// minor.
void configure_state(const Config &config, FactoryList &factories) {
    if (!config.incompatible_core_version) {
        factories.push_back(std::make_unique<status::Factory>());
        return;
    }
    const auto message = mismatch_message(*config.incompatible_core_version);
    LOG(ERROR) << message << ". " << MISMATCH_DESCRIPTION;
    factories.push_back(
        std::make_unique<status::Factory>(status::Health{
            .variant = synnax::status::VARIANT_ERROR,
            .message = message,
            .description = MISMATCH_DESCRIPTION,
        })
    );
}

void configure_mismatch(
    const std::string &core_version,
    const std::string &integration_name,
    FactoryList &factories
) {
    // A task can still hold the running status that an earlier Driver wrote.
    factories.push_back(
        std::make_unique<UnavailableFactory>(
            integration_name,
            mismatch_message(core_version) + ". " + MISMATCH_DESCRIPTION,
            "",
            true
        )
    );
}

#ifdef SYNNAX_NILINUXRT
void configure_modbus(const Config &, FactoryList &factories) {
    factories.push_back(
        std::make_unique<UnavailableFactory>(
            modbus::INTEGRATION_NAME,
            "Modbus tasks do not run on NI Linux RT. Connect the device from a Driver on "
            "Windows, macOS, or Linux."
        )
    );
}
#else
void configure_modbus(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, modbus::INTEGRATION_NAME, []() {
        return std::make_unique<modbus::Factory>();
    });
}
#endif

void configure_arc(
    const Config &config,
    FactoryList &factories,
    const std::shared_ptr<x::thread::rt::Manager> &rt_manager
) {
    configure_integration(config, factories, arc::INTEGRATION_NAME, [&rt_manager]() {
        return std::make_unique<arc::Factory>(rt_manager);
    });
}

void configure_ethercat(
    const Config &config,
    FactoryList &factories,
    const std::shared_ptr<synnax::Synnax> &client,
    const std::shared_ptr<x::thread::rt::Manager> &rt_manager
) {
    configure_integration(
        config,
        factories,
        ethercat::INTEGRATION_NAME,
        [&config, &client, &rt_manager]() {
            return std::make_unique<ethercat::Factory>(
                client,
                config.rack.key,
                rt_manager,
                config.manager.op_timeout
            );
        }
    );
}

void configure_http(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, http::INTEGRATION_NAME, []() {
        return std::make_unique<http::Factory>();
    });
}

std::unique_ptr<task::Factory> Config::new_factory(
    const std::shared_ptr<synnax::Synnax> &client,
    const std::shared_ptr<x::thread::rt::Manager> &rt_manager
) const {
    FactoryList factories;
    configure_state(*this, factories);
    configure_opcua(*this, factories);
    configure_ni(*this, factories);
    configure_labjack(*this, factories);
    configure_arc(*this, factories, rt_manager);
    configure_ethercat(*this, factories, client, rt_manager);
    configure_http(*this, factories);
    configure_modbus(*this, factories);
    return std::make_unique<task::MultiFactory>(std::move(factories));
}
}
