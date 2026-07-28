// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <tuple>
#include <unordered_map>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/mem/local_shared.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/stl/channels/state.h"
#include "arc/cpp/stl/series/state.h"
#include "arc/cpp/stl/stateful/state.h"
#include "arc/cpp/stl/strings/state.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime::state {
using Series = x::mem::local_shared<x::telem::Series>;

struct AuthorityChange {
    std::optional<types::ChannelKey> channel_key;
    uint8_t authority;
};

struct Value {
    Series data;
    Series time;
};

using ChannelDigest = stl::channels::Digest;

struct Config {
    ir::IR ir;
    std::vector<ChannelDigest> channels;
};

class State;

class Node {
    friend class State;
    State &state;
    std::vector<ir::Edge> inputs;
    std::vector<ir::Handle> outputs;
    std::vector<size_t> input_source_idx;
    std::vector<size_t> output_idx;

    /// @brief marks an input whose source output is absent from program state.
    static constexpr size_t NO_SOURCE = ~size_t{0};

    struct InputEntry {
        size_t source{NO_SOURCE};
        Series data;
        Series time;
        x::telem::TimeStamp last_timestamp{0};
        bool consumed{true};
    };

    /// @brief selects when a consumed input re-arms and fires again.
    enum class Rearm : uint8_t {
        /// @brief re-arms on scope Reset and on fresh data.
        Always,
        /// @brief re-arms only on fresh data: reads fire on unseen values.
        OnFresh,
        /// @brief re-arms only on scope Reset: self-writes fire once per entry.
        OnReset,
        /// @brief absorbs pending data on Reset: only post-entry values fire.
        OnArrival,
    };

    std::vector<InputEntry> accumulated;
    std::vector<Series> aligned_data;
    std::vector<Series> aligned_time;
    // is_reference marks inputs that are channel references rather than value
    // streams. Reference inputs carry no data series and never gate execution.
    std::vector<bool> is_reference;
    /// @brief rearm[i] selects when a consumed input i fires again.
    std::vector<Rearm> rearm;
    /// @brief params holds the node's input params with their configured values.
    types::Params params;

    Node(
        State &state,
        std::vector<ir::Edge> inputs,
        std::vector<ir::Handle> outputs,
        std::vector<size_t> input_source_idx,
        std::vector<size_t> output_idx,
        std::vector<InputEntry> accumulated,
        std::vector<Series> aligned_data,
        std::vector<Series> aligned_time,
        std::vector<bool> is_reference,
        std::vector<Rearm> rearm,
        types::Params params
    ):
        state(state),
        inputs(std::move(inputs)),
        outputs(std::move(outputs)),
        input_source_idx(std::move(input_source_idx)),
        output_idx(std::move(output_idx)),
        accumulated(std::move(accumulated)),
        aligned_data(std::move(aligned_data)),
        aligned_time(std::move(aligned_time)),
        is_reference(std::move(is_reference)),
        rearm(std::move(rearm)),
        params(std::move(params)) {}

    /// @brief marks input i consumed at its current source timestamp.
    void absorb_input(size_t i);

public:
    Node(const Node &) = delete;
    Node &operator=(const Node &) = delete;
    Node &operator=(Node &&) = delete;
    Node(Node &&) noexcept = default;

    bool refresh_inputs();

    [[nodiscard]] const Series &input(const size_t param_index) const {
        return this->aligned_data[param_index];
    }

    [[nodiscard]] const Series &input_time(size_t param_index) const;

    [[nodiscard]] Series &output(size_t param_index) const;
    [[nodiscard]] Series &output_time(size_t param_index) const;

    /// Reads buffered data and time series from a channel. Returns (data, index_data,
    /// ok). If the channel has an associated index, both data and time are returned.
    std::tuple<x::telem::MultiSeries, x::telem::MultiSeries, bool>
    read_series(types::ChannelKey key) const;

    /// Writes data and time series to a channel buffer.
    void
    write_series(types::ChannelKey key, const Series &data, const Series &time) const;

    /// @brief reports whether the cached output at the given 0-based
    /// ordinal is truthy. Out-of-range ordinals report false.
    [[nodiscard]] bool is_output_truthy(size_t output_idx) const;

    /// @brief Checks if a series is truthy by examining its last element.
    /// Empty series are falsy. A series with a last element of zero is falsy.
    [[nodiscard]] static bool is_series_truthy(const x::telem::Series &series) {
        if (series.size() == 0) return false;
        const auto last_value = series.at(-1);
        return std::visit(
            []<typename T0>(const T0 &v) -> bool {
                using T = std::decay_t<T0>;
                if constexpr (std::is_same_v<T, std::string>)
                    return !v.empty();
                else
                    return v != 0;
            },
            last_value
        );
    }

    /// @brief Initializes an input's source with the given data and time series.
    /// Used to seed optional inputs (e.g., reset signals) so that
    /// refresh_inputs does not block on them before they receive real data.
    void init_input(size_t param_index, const Series &data, const Series &time);

    /// @brief reports whether the reference input at param_index is edge-fed.
    [[nodiscard]] bool ref_sourced(size_t param_index) const;

    /// @brief returns the current data of an edge-fed reference input, or a null
    /// series when the input is unedged.
    [[nodiscard]] Series ref_input(size_t param_index) const;

    /// @brief returns the named input's current value: the referenced
    /// variable's value when var-bound (its declared initial until first written),
    /// else the configured value.
    [[nodiscard]] std::string string_input(const std::string &name) const;

    /// @brief returns the named input's current value: the referenced
    /// variable's value when var-bound (its declared initial until first written),
    /// else the configured value.
    template<typename T>
    [[nodiscard]] T numeric_input(const std::string &name) const {
        const auto [i, err] = this->resolve_input(name);
        if (err) return 0;
        if (const auto s = this->ref_input(i); s != nullptr && s->size() > 0)
            return s->at<T>(-1);
        const auto &p = this->params[i];
        if (const auto v = types::to_sample_value(p.value, p.type))
            return x::telem::cast<T>(*v);
        return 0;
    }

    /// @brief marks every data input consumed at its current source timestamp,
    /// so only writes after this call re-fire the node.
    void absorb_inputs();

    /// @brief returns input i's unconsumed data, marking it consumed. ok is
    /// false when input i is a reference or has no new data.
    [[nodiscard]] std::pair<Series, bool> consume_input(size_t i);

    /// @brief reports whether input i has unconsumed data, without consuming it.
    [[nodiscard]] bool input_fresh(size_t i) const;

    /// @brief returns the series of the most-recently-changed input, marking it
    /// consumed for last-write-wins. ok is false when no input has new data.
    [[nodiscard]] std::pair<Series, bool> last_changed();

    /// @brief returns the position of the named input, or a NOT_FOUND error if
    /// the node has no such param. Resolve at construction so wiring mistakes fail
    /// at load.
    [[nodiscard]] std::pair<size_t, x::errors::Error>
    resolve_input(const std::string &name) const;

    /// @brief Re-arms every input when the node's stage is (re)activated, so a node
    /// whose gating inputs are all literal-valued re-runs instead of staying consumed.
    void reset() {
        for (size_t i = 0; i < this->accumulated.size(); i++) {
            switch (this->rearm[i]) {
                case Rearm::OnFresh:
                    break;
                case Rearm::OnArrival:
                    this->absorb_input(i);
                    break;
                case Rearm::Always:
                case Rearm::OnReset:
                    this->accumulated[i].consumed = false;
                    this->accumulated[i].last_timestamp = x::telem::TimeStamp(0);
                    break;
            }
        }
    }

    /// @brief Sets the current node key on the parent state for stateful variable
    /// isolation.
    void set_current_node_key(const std::string &key);
};

class State {
    friend class Node;
    Config cfg;
    std::vector<Value> values;
    std::unordered_map<ir::Handle, size_t> value_index;
    /// @brief Per-module state slices.
    std::shared_ptr<stl::channels::State> channel;
    std::shared_ptr<stl::strings::State> strings;
    std::shared_ptr<stl::series::State> series;
    std::shared_ptr<stl::stateful::Variables> vars;

    /// @brief Callback for reporting warnings (e.g., data drops).
    errors::Handler error_handler;

    /// @brief Buffered authority changes from set_authority nodes.
    std::vector<AuthorityChange> authority_changes;

public:
    explicit State(
        const Config &cfg,
        errors::Handler error_handler = errors::noop_handler
    );
    State(
        const Config &cfg,
        std::shared_ptr<stl::channels::State> channel,
        std::shared_ptr<stl::strings::State> strings,
        std::shared_ptr<stl::series::State> series,
        std::shared_ptr<stl::stateful::Variables> vars,
        errors::Handler error_handler = errors::noop_handler
    );
    std::pair<Node, x::errors::Error> node(const std::string &key);
    void ingest(const x::telem::Frame &frame);
    /// @brief flushes channel state directly into the provided frame, avoiding
    /// intermediate allocations.
    void flush_into(x::telem::Frame &out);

    /// @brief Buffers an authority change request for later flushing.
    /// If channel_key is nullopt, the change applies to all write channels.
    void set_authority(std::optional<types::ChannelKey> channel_key, uint8_t authority);

    /// @brief Returns and clears all buffered authority changes.
    std::vector<AuthorityChange> flush_authority_changes();

    /// @brief Clears all persistent state, resetting the runtime to initial conditions.
    void reset();

    /// @brief Sets the current node key for stateful variable isolation.
    /// Must be called before each WASM function invocation.
    void set_current_node_key(const std::string &key) {
        this->vars->set_current_node_key(key);
    }
};
}
