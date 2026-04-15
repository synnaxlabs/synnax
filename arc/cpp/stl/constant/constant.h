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

#include "x/cpp/errors/errors.h"
#include "x/cpp/mem/local_shared.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/types/types.h"

namespace arc::stl::constant {
/// @brief Node that outputs a constant value once on initialization,
/// or after reset() has been called.
class Constant : public runtime::node::Node {
    runtime::state::Node state;
    x::telem::SampleValue value;
    bool initialized = false;

public:
    Constant(
        runtime::state::Node &&state,
        const x::telem::SampleValue &value,
        const x::telem::DataType &data_type
    ):
        state(std::move(state)), value(data_type.cast(value)) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (this->initialized) return x::errors::NIL;
        this->initialized = true;
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        if (o->data_type().is_variable())
            *o = x::telem::Series(this->value);
        else {
            o->resize(1);
            o->set(0, this->value);
        }
        o_time->resize(1);
        o_time->set(0, x::telem::TimeStamp::now());
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

    void reset() override { this->initialized = false; }

    [[nodiscard]] std::vector<std::string> outputs() const override {
        return {ir::default_output_param};
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param) const override {
        return this->state.is_output_truthy(param);
    }
};

class Module : public stl::Module {
public:
    bool handles(const std::string &node_type) const override {
        return node_type == "constant";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        const auto &param = cfg.node.config["value"];
        auto sample_value = types::to_sample_value(param.value, param.type);
        if (!sample_value.has_value())
            return {
                nullptr,
                x::errors::Error(
                    x::errors::VALIDATION,
                    "constant node missing required value parameter"
                )
            };
        auto data_type = cfg.node.outputs[0].type.telem();
        return {
            std::make_unique<Constant>(std::move(cfg.state), *sample_value, data_type),
            x::errors::NIL
        };
    }
};

}
