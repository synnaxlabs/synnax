// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <chrono>
#include <memory>
#include <set>
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/arc/arc.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/runtime.h"

namespace driver::arc {
/// @brief computes the calculated channels that an Arc reads from their base inputs,
/// so that the Arc does not need the Core to compute them.
class Calculator {
    std::vector<synnax::arc::Calculation> calculations;
    ::arc::runtime::Config cfg;
    ::arc::runtime::scheduler::ErrorHandler on_error;
    std::vector<std::shared_ptr<::arc::runtime::Runtime>> runtimes;
    /// @brief the calculated channels and their indexes.
    std::set<::arc::types::ChannelKey> computed;
    /// @brief the channels that the calculations read.
    std::set<::arc::types::ChannelKey> inputs;
    /// @brief true once a compute runs after the last open.
    bool used = false;

    /// @brief loads a new runtime for each calculation. On error, the calculator keeps
    /// the runtimes it had.
    x::errors::Error open() {
        std::vector<std::shared_ptr<::arc::runtime::Runtime>> loaded;
        std::set<::arc::types::ChannelKey> loaded_computed;
        std::set<::arc::types::ChannelKey> loaded_inputs;
        for (const auto &c: this->calculations) {
            this->cfg.program = c.program;
            auto [rt, err] = ::arc::runtime::load(
                this->cfg,
                ::arc::runtime::errors::noop_handler,
                this->on_error
            );
            if (err) return err;
            loaded_computed.insert(
                rt->write_channels.begin(),
                rt->write_channels.end()
            );
            loaded_inputs.insert(rt->read_channels.begin(), rt->read_channels.end());
            loaded.push_back(std::move(rt));
        }
        this->runtimes = std::move(loaded);
        this->computed = std::move(loaded_computed);
        this->inputs = std::move(loaded_inputs);
        this->used = false;
        return x::errors::NIL;
    }

public:
    /// @brief loads a runtime for each calculation.
    /// @param calculations the compiled calculations, dependencies first.
    /// @param cfg the runtime config. Its program and factories are not used. Its
    /// retrieve_channels runs only here, so that reset needs no Core.
    /// @param on_error receives the errors that the nodes of a calculation raise, at
    /// most one a second, as a failing calculation fails on every frame.
    static std::pair<Calculator, x::errors::Error> load(
        const std::vector<synnax::arc::Calculation> &calculations,
        ::arc::runtime::Config cfg,
        ::arc::runtime::scheduler::ErrorHandler on_error
    ) {
        Calculator calc;
        calc.calculations = calculations;
        calc.on_error =
            [on_error = std::move(on_error),
             last = std::make_shared<std::chrono::steady_clock::time_point>(
                 std::chrono::steady_clock::now() - x::telem::SECOND.chrono()
             )](const ::arc::ir::Node &node, const x::errors::Error &err) {
                const auto now = std::chrono::steady_clock::now();
                if (now - *last < x::telem::SECOND.chrono()) return;
                *last = now;
                on_error(node, err);
            };
        cfg.factories.clear();
        cfg.rt_handle = nullptr;
        auto digests = std::make_shared<std::unordered_map<
            ::arc::types::ChannelKey,
            ::arc::runtime::state::ChannelDigest>>();
        cfg.retrieve_channels = [digests, retrieve = std::move(cfg.retrieve_channels)](
                                    const std::vector<::arc::types::ChannelKey> &keys
                                )
            -> std::pair<
                std::vector<::arc::runtime::state::ChannelDigest>,
                x::errors::Error> {
            std::vector<::arc::runtime::state::ChannelDigest> found;
            for (const auto key: keys)
                if (const auto it = digests->find(key); it != digests->end())
                    found.push_back(it->second);
            if (found.size() == keys.size()) return {found, x::errors::NIL};
            auto [fetched, err] = retrieve(keys);
            if (err) return {{}, err};
            for (const auto &d: fetched)
                digests->emplace(d.key, d);
            return {fetched, x::errors::NIL};
        };
        calc.cfg = std::move(cfg);
        if (const auto err = calc.open()) return {Calculator{}, err};
        return {std::move(calc), x::errors::NIL};
    }

    /// @brief returns the channels to stream for an Arc that reads the given channels.
    /// A calculated channel gives way to the base channels it reads.
    [[nodiscard]] std::vector<::arc::types::ChannelKey>
    streamed(const std::vector<::arc::types::ChannelKey> &reads) const {
        std::set keys(reads.begin(), reads.end());
        keys.insert(this->inputs.begin(), this->inputs.end());
        for (const auto key: this->computed)
            keys.erase(key);
        return {keys.begin(), keys.end()};
    }

    /// @brief loads every calculation again if a compute ran since the last load, so
    /// that the next compute starts from a fresh state. On error, the calculator keeps
    /// its runtimes. Call it only while no compute runs.
    x::errors::Error reset() {
        if (!this->used) return x::errors::NIL;
        return this->open();
    }

    /// @brief adds to frame the series of each calculated channel that its inputs in
    /// frame let it compute. A calculation reads the series that the ones before it
    /// added. A calculation with no input in frame does not run.
    void compute(x::telem::Frame &frame) {
        this->used = true;
        for (const auto &rt: this->runtimes) {
            x::telem::Frame in;
            for (const auto [key, series]: frame)
                if (std::ranges::find(rt->read_channels, key) !=
                    rt->read_channels.end())
                    in.emplace(key, series.shallow_copy());
            if (in.empty()) continue;
            for (auto [key, series]: rt->compute(in))
                frame.emplace(key, std::move(series));
        }
    }
};
}
