// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "x/cpp/telem/telem.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/memory/local_shared.h"

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
    x::telem::DataType data_type;
    bool initialized = false;

public:
    Constant(state::Node state, nlohmann::json value, const x::telem::DataType &data_type):
        state(std::move(state)), value(std::move(value)), data_type(data_type) {}

    x::errors::Error next(node::Context &ctx) override {
        if (this->initialized) return x::errors::NIL;
        this->initialized = true;
        const auto &o = state.output(0);
        const auto &o_time = state.output_time(0);

        o->resize(1);
        o_time->resize(1);

        if (data_type == x::telem::INT64_T)
            o->set(0, value.get<int64_t>());
        else if (data_type == x::telem::INT32_T)
            o->set(0, value.get<int32_t>());
        else if (data_type == x::telem::INT16_T)
            o->set(0, value.get<int16_t>());
        else if (data_type == x::telem::INT8_T)
            o->set(0, value.get<int8_t>());
        else if (data_type == x::telem::UINT64_T)
            o->set(0, value.get<uint64_t>());
        else if (data_type == x::telem::UINT32_T)
            o->set(0, value.get<uint32_t>());
        else if (data_type == x::telem::UINT16_T)
            o->set(0, value.get<uint16_t>());
        else if (data_type == x::telem::UINT8_T)
            o->set(0, value.get<uint8_t>());
        else if (data_type == x::telem::FLOAT64_T)
            o->set(0, value.get<double>());
        else if (data_type == x::telem::FLOAT32_T)
            o->set(0, value.get<float>());

        o_time->set(0, x::telem::TimeStamp::now());
        ctx.mark_changed(ir::default_output_param);
        return x::errors::NIL;
    }

    void reset() override { this->initialized = false; }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return this->state.is_output_truthy(param_name);
    }
};

/// Factory creates Constant nodes for "constant" type nodes in the IR.
class Factory : public node::Factory {
public:
    std::pair<std::unique_ptr<node::Node>, x::errors::Error>
    create(const node::Config &cfg) override {
        if (cfg.node.type != "constant") return {nullptr, x::errors::NOT_FOUND};

        const auto value_param = cfg.node.config.get("value");
        if (value_param == nullptr)
            return {nullptr, x::errors::Error("constant node missing value config")};

        if (cfg.node.outputs.empty())
            return {nullptr, x::errors::Error("constant node missing output definition")};

        auto data_type = cfg.node.outputs[0].type.telem();

        return {
            std::make_unique<Constant>(cfg.state, value_param->value, data_type),
            x::errors::NIL
        };
    }
};

}
