// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xmemory/local_shared.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::constant {

/// Constant is a node that outputs a constant value once on initialization.
/// After the initial output, it does nothing on subsequent Next() calls.
class Constant : public node::Node {
    state::Node state;
    nlohmann::json value;
    telem::DataType data_type;
    bool initialized = false;

public:
    Constant(state::Node state, nlohmann::json value, telem::DataType data_type):
        state(std::move(state)), value(std::move(value)), data_type(data_type) {}

    xerrors::Error next(node::Context &ctx) override {
        if (initialized) return xerrors::NIL;
        initialized = true;

        auto &o = state.output(0);
        auto &o_time = state.output_time(0);

        o->resize(1);
        o_time->resize(1);

        // Set the value based on data type
        if (data_type == telem::INT64_T) {
            o->set(0, value.get<int64_t>());
        } else if (data_type == telem::INT32_T) {
            o->set(0, value.get<int32_t>());
        } else if (data_type == telem::INT16_T) {
            o->set(0, value.get<int16_t>());
        } else if (data_type == telem::INT8_T) {
            o->set(0, value.get<int8_t>());
        } else if (data_type == telem::UINT64_T) {
            o->set(0, value.get<uint64_t>());
        } else if (data_type == telem::UINT32_T) {
            o->set(0, value.get<uint32_t>());
        } else if (data_type == telem::UINT16_T) {
            o->set(0, value.get<uint16_t>());
        } else if (data_type == telem::UINT8_T) {
            o->set(0, value.get<uint8_t>());
        } else if (data_type == telem::FLOAT64_T) {
            o->set(0, value.get<double>());
        } else if (data_type == telem::FLOAT32_T) {
            o->set(0, value.get<float>());
        }

        o_time->set(0, telem::TimeStamp::now().nanoseconds());
        ctx.mark_changed(ir::default_output_param);
        return xerrors::NIL;
    }

    void reset() override { initialized = false; }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

/// Factory creates Constant nodes for "constant" type nodes in the IR.
class Factory : public node::Factory {
public:
    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(const node::Config &cfg) override {
        if (cfg.node.type != "constant") return {nullptr, xerrors::NOT_FOUND};

        auto value_param = cfg.node.config.get("value");
        if (value_param == nullptr)
            return {nullptr, xerrors::Error("constant node missing value config")};

        // Get the output type from the node's outputs
        if (cfg.node.outputs.empty())
            return {nullptr, xerrors::Error("constant node missing output definition")};

        auto data_type = cfg.node.outputs[0].type.telem();

        return {
            std::make_unique<Constant>(cfg.state, value_param->value, data_type),
            xerrors::NIL
        };
    }
};

} // namespace arc::runtime::constant