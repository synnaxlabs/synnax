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

void configure_state(FactoryList &factories) {
    factories.push_back(std::make_unique<status::Factory>());
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
    const std::shared_ptr<x::thread::rt::Manager> &rt_manager
) {
    configure_integration(
        config,
        factories,
        ethercat::INTEGRATION_NAME,
        [&rt_manager]() { return std::make_unique<ethercat::Factory>(rt_manager); }
    );
}

void configure_http(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, http::INTEGRATION_NAME, []() {
        return std::make_unique<http::Factory>();
    });
}

std::unique_ptr<task::Factory>
Config::new_factory(const std::shared_ptr<x::thread::rt::Manager> &rt_manager) const {
    FactoryList factories;
    configure_state(factories);
    configure_opcua(*this, factories);
    configure_ni(*this, factories);
    configure_labjack(*this, factories);
    configure_arc(*this, factories, rt_manager);
    configure_ethercat(*this, factories, rt_manager);
    configure_http(*this, factories);
    configure_modbus(*this, factories);
    return std::make_unique<task::MultiFactory>(std::move(factories));
}
}
