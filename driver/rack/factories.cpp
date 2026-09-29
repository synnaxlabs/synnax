// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#ifndef SYNNAX_NILINUXRT
#include "driver/modbus/modbus.h"
#endif

#include "driver/arc/arc.h"
#include "driver/arinc429/arinc429.h"
#include "driver/can/backends/backends.h"
#include "driver/can/factory.h"
#include "driver/ethercat/ethercat.h"
#include "driver/http/http.h"
#include "driver/rack/rack.h"
#include "driver/rack/status/status.h"
#include "driver/serial/serial.h"
#include "driver/tcp/tcp.h"
#include "driver/udp/udp.h"

namespace driver::rack {
using FactoryList = std::vector<std::unique_ptr<task::Factory>>;

bool Config::integration_enabled(const std::string &i) const {
    return std::ranges::find(integrations, i) != integrations.end();
}

template<typename F>
void configure_integration(
    const Config &config,
    FactoryList &factories,
    const std::string &integration_name,
    F factory_creator
) {
    if (!config.integration_enabled(integration_name)) {
        VLOG(1) << "[" << integration_name << "] integration disabled";
        return;
    }
    VLOG(1) << "[" << integration_name << "] integration enabled";
    factories.push_back(factory_creator());
}

void configure_opcua(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, opcua::INTEGRATION_NAME, []() {
        return std::make_unique<opcua::Factory>();
    });
}

void configure_ni(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, ni::INTEGRATION_NAME, [&config]() {
        return ni::Factory::create(config.timing);
    });
}

void configure_labjack(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, labjack::INTEGRATION_NAME, [&config]() {
        return labjack::Factory::create(config.timing);
    });
}

void configure_state(FactoryList &factories) {
    factories.push_back(std::make_unique<status::Factory>());
}

#ifndef SYNNAX_NILINUXRT
void configure_modbus(const Config &config, FactoryList &factories) {
    if (!config.integration_enabled(modbus::INTEGRATION_NAME)) return;
    factories.push_back(std::make_unique<modbus::Factory>());
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

void configure_can(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, can::INTEGRATION_NAME, []() {
        return std::make_unique<can::Factory>(can::backends::load());
    });
}

void configure_serial(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, serial::INTEGRATION_NAME, []() {
        return std::make_unique<serial::Factory>();
    });
}

void configure_tcp(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, tcp::INTEGRATION_NAME, []() {
        return std::make_unique<tcp::Factory>();
    });
}

void configure_udp(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, udp::INTEGRATION_NAME, []() {
        return std::make_unique<udp::Factory>();
    });
}

void configure_arinc429(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, arinc429::INTEGRATION_NAME, []() {
        return std::make_unique<arinc429::Factory>(arinc429::create_backends());
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
    configure_can(*this, factories);
    configure_serial(*this, factories);
    configure_tcp(*this, factories);
    configure_udp(*this, factories);
    configure_arinc429(*this, factories);
#ifndef SYNNAX_NILINUXRT
    configure_modbus(*this, factories);
#endif
    return std::make_unique<task::MultiFactory>(std::move(factories));
}
}
