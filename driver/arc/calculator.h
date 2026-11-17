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
#include <set>
#include <utility>
#include <vector>

#include "client/cpp/arc/arc.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"

#include "arc/cpp/runtime/runtime.h"

namespace driver::arc {
/// @brief computes the calculated channels that an Arc reads from their base inputs,
/// so that the Arc does not need the Core to compute them.
class Calculator {
    std::vector<std::shared_ptr<::arc::runtime::Runtime>> runtimes;
    /// @brief the calculated channels and their indexes.
    std::set<::arc::types::ChannelKey> computed;
    /// @brief the channels that the calculations read.
    std::set<::arc::types::ChannelKey> inputs;

public:
    /// @brief loads a runtime for each calculation.
    /// @param calculations the compiled calculations, dependencies first.
    /// @param cfg the runtime config. Its program and factories are not used.
    /// @param on_error receives the errors that the nodes of a calculation raise.
    static std::pair<Calculator, x::errors::Error> load(
        const std::vector<synnax::arc::Calculation> &calculations,
        ::arc::runtime::Config cfg,
        const ::arc::runtime::scheduler::ErrorHandler &on_error
    ) {
        Calculator calc;
        cfg.factories.clear();
        cfg.rt_handle = nullptr;
        for (const auto &c: calculations) {
            cfg.program = c.program;
            auto [rt, err] = ::arc::runtime::load(
                cfg,
                ::arc::runtime::errors::noop_handler,
                on_error
            );
            if (err) return {Calculator{}, err};
            calc.computed.insert(rt->write_channels.begin(), rt->write_channels.end());
            calc.inputs.insert(rt->read_channels.begin(), rt->read_channels.end());
            calc.runtimes.push_back(std::move(rt));
        }
        return {std::move(calc), x::errors::NIL};
    }

    /// @brief returns the channels to stream for an Arc that reads reads. A calculated
    /// channel gives way to the base channels it reads.
    [[nodiscard]] std::vector<::arc::types::ChannelKey>
    streamed(const std::vector<::arc::types::ChannelKey> &reads) const {
        std::set keys(reads.begin(), reads.end());
        keys.insert(this->inputs.begin(), this->inputs.end());
        for (const auto key: this->computed)
            keys.erase(key);
        return {keys.begin(), keys.end()};
    }

    /// @brief adds to frame the series of each calculated channel that its inputs in
    /// frame let it compute. A calculation reads the series that the ones before it
    /// added.
    void compute(x::telem::Frame &frame) const {
        for (const auto &rt: this->runtimes) {
            auto out = rt->compute(frame.shallow_copy());
            for (auto [key, series]: out)
                frame.emplace(key, std::move(series));
        }
    }
};
}
