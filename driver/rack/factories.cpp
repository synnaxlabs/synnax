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
#include "driver/common/factory.h"
#include "driver/ethercat/ethercat.h"
#include "driver/http/http.h"
#include "driver/rack/rack.h"
#include "driver/rack/status/status.h"

namespace driver::rack {
using FactoryList = std::vector<std::unique_ptr<task::Factory>>;

bool Config::integration_supported(const std::string &i) const {
    return std::ranges::find(this->integrations, i) != this->integrations.end() &&
           this->libraries_loaded(i);
}

template<typename F>
void configure_integration(
    const Config &config,
    FactoryList &factories,
    const std::string &integration_name,
    F factory_creator
) {
    if (std::ranges::find(config.integrations, integration_name) ==
        config.integrations.end()) {
        VLOG(1) << "[" << integration_name << "] integration not enabled";
        return;
    }
    if (!config.libraries_loaded(integration_name)) {
        LOG(WARNING) << "[" << integration_name
                     << "] integration is enabled, but its libraries do not load on "
                        "this computer";
        return;
    }
    VLOG(1) << "[" << integration_name << "] integration supported";
    factories.push_back(factory_creator());
}

void configure_opcua(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, opcua::INTEGRATION_NAME, [&config]() {
        auto [host, _] = x::os::get_hostname();
        return std::make_unique<opcua::Factory>(
            std::make_shared<opcua::certificate::Store>(opcua::certificate::StoreConfig{
                .dir = config.state_dir / "opcua" / "pki",
                .host = host,
            })
        );
    });
}

#ifdef __APPLE__
/// @brief deletes the NI scanner that a Driver built with NI left on the rack, as the
/// macOS build has no NI integration to run it.
class LegacyNIFactory final : public task::Factory {
public:
    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override {
        const auto err = common::delete_legacy_task_by_type(rack, "ni_scanner", "ni");
        if (err) LOG(ERROR) << "[ni] failed to retrieve the legacy NI scanner: " << err;
        return {};
    }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) override {
        return {nullptr, false};
    }

    std::string name() override { return "ni"; }
};

void configure_ni(const Config &, FactoryList &factories) {
    factories.push_back(std::make_unique<LegacyNIFactory>());
}
#else
void configure_ni(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, ni::INTEGRATION_NAME, [&config]() {
        return ni::Factory::create(config.timing);
    });
}
#endif

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
    if (!config.integration_supported(modbus::INTEGRATION_NAME)) return;
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
#ifndef SYNNAX_NILINUXRT
    configure_modbus(*this, factories);
#endif
    return std::make_unique<task::MultiFactory>(std::move(factories));
}
}
