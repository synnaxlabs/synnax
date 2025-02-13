// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/rack/rack.h"

typedef std::vector<std::unique_ptr<task::Factory> > FactoryList;

bool rack::Config::integration_enabled(const std::string &i) const {
    return std::find(integrations.begin(), integrations.end(), i) != integrations.end();
}

void configure_opc(const rack::Config &config, FactoryList &factories) {
    if (!config.integration_enabled(opc::INTEGRATION_NAME)) return;
    factories.push_back(std::make_unique<opc::Factory>());
}

void configure_ni(const rack::Config &config, FactoryList &factories) {
    if (!config.integration_enabled(ni::INTEGRATION_NAME)) return;
    factories.push_back(ni::Factory::create());
}

void configure_sequences(const rack::Config &config, FactoryList &factories) {
    if (!config.integration_enabled(sequence::INTEGRATION_NAME)) return;
    factories.push_back(std::make_unique<sequence::Factory>());
}

void configure_heartbeat(const rack::Config &config, FactoryList &factories) {
    if (!config.integration_enabled(heartbeat::INTEGRATION_NAME)) return;
    factories.push_back(std::make_unique<heartbeat::Factory>());
}

void configure_labjack(const rack::Config &config, FactoryList &factories) {
#ifdef _WIN32
    if (
        !config.integration_enabled(labjack::INTEGRATION_NAME) ||
        !labjack::dlls_available()
    ) return;
    factories.push_back(std::make_unique<labjack::Factory>());
    return;
#endif
}

std::unique_ptr<task::Factory> rack::Config::new_factory() const {
    FactoryList factories;
    configure_heartbeat(*this, factories);
    configure_opc(*this, factories);
    configure_ni(*this, factories);
    configure_sequences(*this, factories);
    configure_labjack(*this, factories);
    return std::make_unique<task::MultiFactory>(std::move(factories));
}
