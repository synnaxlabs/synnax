// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xmemory/local_shared.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::constant {
/// @brief Constant is a node that outputs a constant value once on initialization,
/// or after reset() has been called.
class Constant : public node::Node {
    state::Node state;
    telem::SampleValue value;
    bool initialized = false;

public:
    Constant(
        state::Node &&state,
        const telem::SampleValue &value,
        const telem::DataType &data_type
    ):
        state(std::move(state)), value(data_type.cast(value)) {}

    xerrors::Error next(node::Context &ctx) override {
        if (this->initialized) return xerrors::NIL;
        this->initialized = true;
        const auto &o = state.output(0);
        const auto &o_time = state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, this->value);
        o_time->set(0, telem::TimeStamp::now());
        ctx.mark_changed(ir::default_output_param);
        return xerrors::NIL;
    }

    void reset() override { this->initialized = false; }

    [[nodiscard]] bool is_output_truthy(const std::string &param) const override {
        return this->state.is_output_truthy(param);
    }
};

class Factory : public node::Factory {
public:
    bool handles(const std::string &node_type) const override {
        return node_type == "constant";
    }

    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, xerrors::NOT_FOUND};
        const auto &param = cfg.node.config["value"];
        assert(param.value.has_value() && "constant node requires a value");
        auto data_type = cfg.node.outputs[0].type.telem();
        return {
            std::make_unique<Constant>(std::move(cfg.state), *param.value, data_type),
            xerrors::NIL
        };
    }
};

}
