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
#include <optional>
#include <stdexcept>
#include <string>
#include <type_traits>
#include <utility>
#include <variant>
#include <vector>

#include "x/cpp/json/json.h"
#include "x/cpp/mem/indirect.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/types/types.h"

namespace arc::stl::testutil {
template<typename>
inline constexpr bool always_false = false;

/// @brief returns the Arc type for the sample type T.
template<typename T>
types::Type sample_type() {
    types::Type t;
    if constexpr (std::is_same_v<T, std::uint8_t>)
        t.kind = types::Kind::U8;
    else if constexpr (std::is_same_v<T, std::uint16_t>)
        t.kind = types::Kind::U16;
    else if constexpr (std::is_same_v<T, std::uint32_t>)
        t.kind = types::Kind::U32;
    else if constexpr (std::is_same_v<T, std::uint64_t>)
        t.kind = types::Kind::U64;
    else if constexpr (std::is_same_v<T, std::int8_t>)
        t.kind = types::Kind::I8;
    else if constexpr (std::is_same_v<T, std::int16_t>)
        t.kind = types::Kind::I16;
    else if constexpr (std::is_same_v<T, std::int32_t>)
        t.kind = types::Kind::I32;
    else if constexpr (std::is_same_v<T, std::int64_t>)
        t.kind = types::Kind::I64;
    else if constexpr (std::is_same_v<T, float>)
        t.kind = types::Kind::F32;
    else if constexpr (std::is_same_v<T, double>)
        t.kind = types::Kind::F64;
    else if constexpr (std::is_same_v<T, std::string>)
        t.kind = types::Kind::String;
    else
        static_assert(always_false<T>, "unsupported sample type");
    return t;
}

/// @brief VarBinding is the type-erased view config uses to wire a VarInput.
class VarBinding {
public:
    virtual ~VarBinding() = default;

    [[nodiscard]] virtual x::json::json initial_value() const = 0;

    [[nodiscard]] virtual types::Type elem_type() const = 0;

    virtual void bind(runtime::state::State &state, const std::string &node_key) = 0;
};

/// @brief VarInput is a variable input: declare it, pass it to NodeSpec::config, then
/// write it through set at any point.
template<typename T>
class VarInput final : public VarBinding {
    T initial;
    std::optional<T> written;
    runtime::state::State *state = nullptr;
    std::string node_key;

public:
    /// @brief returns a VarInput holding initial as its declared value.
    explicit VarInput(T initial): initial(std::move(initial)) {}

    /// @brief writes the variable's current value.
    void set(T value) {
        if (this->state == nullptr)
            throw std::runtime_error(
                "variable input was not passed to NodeSpec::config"
            );
        auto [node, err] = this->state->node(this->node_key);
        if (err) throw std::runtime_error(err.message());
        *node.output(0) = x::telem::Series(std::move(value));
    }

    /// @brief writes value the moment this input is bound, rather than at its
    /// declared initial.
    void set_on_bind(T value) { this->written = std::move(value); }

    [[nodiscard]] x::json::json initial_value() const override {
        return x::json::json(this->initial);
    }

    [[nodiscard]] types::Type elem_type() const override { return sample_type<T>(); }

    void bind(runtime::state::State &s, const std::string &key) override {
        this->state = &s;
        this->node_key = key;
        if (this->written.has_value()) this->set(*this->written);
    }
};

/// @brief returns a VarInput whose value arrives via a write the
/// moment it is bound, not via its declared initial (the type's zero value).
template<typename T>
std::shared_ptr<VarInput<T>> var_of(T value) {
    auto v = std::make_shared<VarInput<T>>(T{});
    v->set_on_bind(std::move(value));
    return v;
}

/// @brief the value for one declared input: a constant, or a VarInput the node reads
/// live.
using InputValue = std::variant<x::json::json, std::shared_ptr<VarBinding>>;

/// @brief owns the IR and state a node config is built from. Both outlive every config
/// and state node handed out, so a fixture must stay alive for the test's duration.
class Fixture {
    ir::IR prog;
    std::shared_ptr<runtime::state::State> state;
    std::string node_key;

public:
    Fixture(
        ir::IR prog,
        std::shared_ptr<runtime::state::State> state,
        std::string node_key
    ):
        prog(std::move(prog)), state(std::move(state)), node_key(std::move(node_key)) {}

    [[nodiscard]] const ir::IR &program() const { return this->prog; }

    [[nodiscard]] const ir::Node &ir_node() const {
        return this->prog.node(this->node_key);
    }

    [[nodiscard]] runtime::state::Node make_node() const {
        auto [node, err] = this->state->node(this->node_key);
        if (err) throw std::runtime_error(err.message());
        return std::move(node);
    }

    [[nodiscard]] runtime::node::Config make_config() const {
        return runtime::node::Config(this->prog, this->ir_node(), this->make_node());
    }
};

/// @brief NodeSpec declares a native's shape for building test configs: its type,
/// outputs, and input params (names and types; values come from config).
struct NodeSpec {
    std::string type;
    types::Params outputs;
    types::Params inputs;

    /// @brief zips the declared inputs with values, each given as a const or a
    /// VarInput read live by the node. The State is built from the same IR the
    /// config carries.
    [[nodiscard]] Fixture config(const std::vector<InputValue> &values) const {
        if (values.size() != this->inputs.size())
            throw std::runtime_error(
                "NodeSpec::config: one value per declared input required"
            );
        ir::IR prog;
        types::Params params;
        std::vector<std::pair<std::shared_ptr<VarBinding>, std::string>> binds;
        for (size_t i = 0; i < this->inputs.size(); i++) {
            const auto &in = this->inputs[i];
            const auto *vb = std::get_if<std::shared_ptr<VarBinding>>(&values[i]);
            if (vb == nullptr) {
                types::Param p;
                p.name = in.name;
                p.type = in.type;
                p.value = std::get<x::json::json>(values[i]);
                params.push_back(std::move(p));
                continue;
            }
            const auto key = "v_" + in.name;
            const auto elem = (*vb)->elem_type();
            ir::Node var_node;
            var_node.key = key;
            var_node.type = "variable";
            types::Param out;
            out.name = ir::default_output_param;
            out.type = elem;
            var_node.outputs.push_back(std::move(out));
            prog.nodes.push_back(std::move(var_node));
            types::Param p;
            p.name = in.name;
            p.type.kind = types::Kind::VarRef;
            p.type.name = key;
            p.type.elem = x::mem::indirect<types::Type>(elem);
            p.value = (*vb)->initial_value();
            params.push_back(std::move(p));
            binds.emplace_back(*vb, key);
        }
        ir::Node consumer;
        consumer.key = "n";
        consumer.type = this->type;
        consumer.inputs = std::move(params);
        consumer.outputs = this->outputs;
        prog.nodes.push_back(std::move(consumer));
        auto state = std::make_shared<runtime::state::State>(
            runtime::state::Config{.ir = prog, .channels = {}},
            runtime::errors::noop_handler
        );
        for (const auto &[binding, key]: binds)
            binding->bind(*state, key);
        return Fixture(std::move(prog), std::move(state), "n");
    }
};

}
