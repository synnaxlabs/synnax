// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/nodes/interval/factory.h"

namespace arc { namespace interval {

std::pair<std::unique_ptr<arc::Node>, xerrors::Error>
Factory::create(const NodeFactoryConfig &cfg) {
    // Check if this is an interval node
    if (cfg.ir_node.type != "interval") {
        // Not an interval node - let another factory handle it
        return {nullptr, xerrors::Error("NOT_FOUND")};
    }

    // Extract period from config_values
    if (cfg.ir_node.config_values.count("period") == 0) {
        return {
            nullptr,
            xerrors::Error(
                "arc.factory.interval_missing_period",
                "Interval node '" + cfg.ir_node.key +
                    "' missing 'period' in config_values"
            )
        };
    }

    uint64_t period_ns;
    try {
        period_ns = cfg.ir_node.config_values.at("period").get<uint64_t>();
    } catch (const std::exception &e) {
        return {
            nullptr,
            xerrors::Error(
                "arc.factory.interval_invalid_period",
                "Invalid period value for node '" + cfg.ir_node.key + "': " + e.what()
            )
        };
    }

    // Extract output channel
    if (cfg.ir_node.channels.write.count("output") == 0) {
        return {
            nullptr,
            xerrors::Error(
                "arc.factory.interval_missing_output",
                "Interval node '" + cfg.ir_node.key +
                    "' missing 'output' in channels.write"
            )
        };
    }

    ChannelKey output_ch = cfg.ir_node.channels.write.at("output");

    // Create interval::Node
    auto interval_node = std::make_unique<Node>(
        cfg.ir_node.key,
        &cfg.state,
        output_ch,
        period_ns
    );

    return {std::move(interval_node), xerrors::NIL};
}

} // namespace interval
} // namespace arc
