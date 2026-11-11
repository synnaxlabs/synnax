// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstring>
#include <functional>
#include <memory>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/types/types.h"

namespace arc::stl::stable {

struct StableForInputs {
    x::telem::TimeSpan duration;

    static std::pair<StableForInputs, x::errors::Error>
    create(const types::Params &params) {
        const auto &param = params["duration"];
        auto sv = types::to_sample_value(param.value, param.type);
        if (!sv.has_value())
            return {
                {},
                x::errors::Error(
                    x::errors::VALIDATION,
                    "The stable_for node is missing its duration parameter."
                )
            };
        return {
            {.duration = x::telem::TimeSpan(x::telem::cast<std::int64_t>(*sv))},
            x::errors::NIL
        };
    }
};

/// @brief StableFor debounces a signal: emits a value only after it stays
/// stable for the given duration, suppressing transient fluctuations.
///
/// Stability is measured in sample time at both ends. The window opens at the
/// stamp of the sample where the value last changed and closes at the first
/// later sample that still holds it. One batch can carry several confirmations,
/// each sent with its own stamp.
class StableFor : public runtime::node::Node {
    runtime::state::Node state;
    x::telem::TimeSpan duration;
    size_t input_idx;
    std::vector<uint8_t> value;
    std::vector<uint8_t> last_sent;
    x::telem::TimeStamp last_changed{0};

    /// @brief latches the var-bound live duration at a window start;
    /// an in-flight window completes under the duration it began with.
    void refresh_duration() {
        const auto [i, err] = this->state.resolve_input("duration");
        if (err || !this->state.ref_sourced(i)) return;
        if (const auto v = this->state.ref_input(i); v != nullptr && !v->empty())
            this->duration = x::telem::TimeSpan(
                this->state.numeric_input<int64_t>("duration")
            );
    }

public:
    explicit StableFor(
        const StableForInputs &inputs,
        runtime::state::Node &&state,
        size_t input_idx
    ):
        state(std::move(state)), duration(inputs.duration), input_idx(input_idx) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (!this->state.refresh_inputs()) return x::errors::NIL;
        const auto &input_data = this->state.input(this->input_idx);
        const auto &input_time = this->state.input_time(this->input_idx);
        const auto density = input_data->data_type().density();
        const auto *data = reinterpret_cast<const uint8_t *>(input_data->data());
        std::vector<uint8_t> held;
        std::vector<x::telem::TimeStamp> stamps;
        for (size_t i = 0; i < input_data->size(); i++) {
            const auto *current = data + i * density;
            const auto stamp = x::telem::TimeStamp(input_time->at<int64_t>(i));
            if (this->value.empty() ||
                std::memcmp(this->value.data(), current, density) != 0) {
                this->value.assign(current, current + density);
                this->last_changed = stamp;
                this->refresh_duration();
            }
            if (x::telem::TimeSpan(stamp - this->last_changed) < this->duration ||
                this->last_sent == this->value)
                continue;
            held.insert(held.end(), this->value.begin(), this->value.end());
            stamps.push_back(stamp);
            this->last_sent = this->value;
        }
        if (stamps.empty()) return x::errors::NIL;
        const auto &o = this->state.output(0);
        o->resize(stamps.size());
        std::memcpy(o->data(), held.data(), held.size());
        *this->state.output_time(0) = x::telem::Series(stamps);
        this->state.emit(ctx.mark_changed, 0);
        return x::errors::NIL;
    }

    void reset(runtime::node::Context &) override {
        this->state.absorb_inputs();
        this->value.clear();
        this->last_sent.clear();
        this->last_changed = x::telem::TimeStamp(0);
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return state.is_output_truthy(output_idx);
    }
};

class Module : public stl::Module {
public:
    [[nodiscard]] std::string module_name() const override { return "stable"; }

    [[nodiscard]] bool handles(const std::string &node_type) const override {
        return node_type == "stable_for" || node_type == "for";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        auto [inputs, err] = StableForInputs::create(cfg.node.inputs);
        if (err) return {nullptr, err};
        auto [input_idx, in_err] = cfg.node.resolve_input(ir::default_input_param);
        if (in_err) return {nullptr, in_err};
        return {
            std::make_unique<StableFor>(inputs, std::move(cfg.state), input_idx),
            x::errors::NIL
        };
    }
};

}
