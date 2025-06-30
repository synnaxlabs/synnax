// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/rack/rack.h"

using FactoryList = std::vector<std::unique_ptr<task::Factory>>;

bool rack::Config::integration_enabled(const std::string &i) const {
    return std::find(integrations.begin(), integrations.end(), i) != integrations.end();
}

template<typename F>
void configure_integration(
    const rack::Config &config,
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

void configure_opc(const rack::Config &config, FactoryList &factories) {
    configure_integration(config, factories, opc::INTEGRATION_NAME, []() {
        return std::make_unique<opc::Factory>();
    });
}

void configure_ni(const rack::Config &config, FactoryList &factories) {
    configure_integration(config, factories, ni::INTEGRATION_NAME, [&config]() {
        return ni::Factory::create(config.timing);
    });
}

void configure_sequences(const rack::Config &config, FactoryList &factories) {
    configure_integration(config, factories, sequence::INTEGRATION_NAME, []() {
        return std::make_unique<sequence::Factory>();
    });
}

void configure_labjack(const rack::Config &config, FactoryList &factories) {
    configure_integration(config, factories, labjack::INTEGRATION_NAME, [&config]() {
        return labjack::Factory::create(config.timing);
    });
}

void configure_state(FactoryList &factories) {
    factories.push_back(std::make_unique<rack::status::Factory>());
}

std::unique_ptr<task::Factory> rack::Config::new_factory() const {
    FactoryList factories;
    configure_state(factories);
    configure_opc(*this, factories);
    configure_ni(*this, factories);
    configure_sequences(*this, factories);
    configure_labjack(*this, factories);
    return std::make_unique<task::MultiFactory>(std::move(factories));
}
