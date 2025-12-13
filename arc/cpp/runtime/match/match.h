// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <memory>
#include <string>
#include <vector>

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::match {

/// CaseMapping stores the output index and output name for a case value.
struct CaseMapping {
    size_t output_index;
    std::string output_name;
};

/// Match is a node that routes input values to corresponding outputs.
/// It receives a string input and fires a u8(1) signal on the output
/// that matches the input value.
class Match : public node::Node {
    state::Node state;
    /// Maps case value -> output mapping (index + name)
    std::map<std::string, CaseMapping> case_map;

public:
    Match(
        state::Node state,
        std::map<std::string, CaseMapping> case_map
    ):
        state(std::move(state)),
        case_map(std::move(case_map)) {}

    xerrors::Error next(node::Context &ctx) override {
        // Check if we have new input
        if (!state.refresh_inputs()) return xerrors::NIL;

        auto &input = state.input(0);
        if (input->size() == 0) return xerrors::NIL;

        // Get the input value as string
        auto input_value = std::get<std::string>(input->at(0));

        // Find matching case
        auto it = case_map.find(input_value);
        if (it != case_map.end()) {
            auto &mapping = it->second;
            auto &output = state.output(mapping.output_index);
            output->resize(1);
            output->set(0, static_cast<std::uint8_t>(1));
            // Mark the output as changed so edges propagate
            ctx.mark_changed(mapping.output_name);
        }

        return xerrors::NIL;
    }
};

/// Factory creates Match nodes for "match" type nodes in the IR.
class Factory : public node::Factory {
public:
    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(const node::Config &cfg) override {
        if (cfg.node.type != "match") return {nullptr, xerrors::NOT_FOUND};

        // Build case_map from config
        std::map<std::string, CaseMapping> case_map;

        auto cases_param = cfg.node.config.get("cases");
        if (cases_param != nullptr) {
            // Cases is an array of {value, output} maps
            auto &cases_value = cases_param->value;
            if (cases_value.is_array()) {
                for (size_t i = 0; i < cases_value.size(); i++) {
                    auto &case_entry = cases_value[i];
                    if (case_entry.contains("value") && case_entry.contains("output")) {
                        auto value = case_entry["value"].get<std::string>();
                        auto output_name = case_entry["output"].get<std::string>();

                        // Find output index by name
                        for (size_t j = 0; j < cfg.node.outputs.size(); j++) {
                            if (cfg.node.outputs[j].name == output_name) {
                                case_map[value] = CaseMapping{j, output_name};
                                break;
                            }
                        }
                    }
                }
            }
        }

        auto node = std::make_unique<Match>(cfg.state, case_map);
        return {std::move(node), xerrors::NIL};
    }
};

}
