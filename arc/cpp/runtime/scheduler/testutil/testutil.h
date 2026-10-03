// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <initializer_list>
#include <memory>
#include <optional>
#include <string>
#include <unordered_map>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"

namespace arc::runtime::scheduler::testutil {
/// @brief configurable mock node used across scheduler tests. Deals
/// exclusively in ordinals — output names live in ir::Node::outputs and
/// are declared at the IR construction layer (program_of + ir_node).
/// Tests construct mocks via the SchedulerTest::mock helper, which takes
/// a per-ordinal truthy slice that drives both is_output_truthy and the
/// auto-mark loop in next.
struct MockNode final : public node::Node {
    int next_called = 0;
    int reset_called = 0;
    std::vector<x::telem::TimeSpan> elapsed_values;
    /// @brief the cycle stamp each reset ran with, in call order.
    std::vector<x::telem::TimeStamp> reset_now;

    /// @brief output_truthy[i] reports whether output ordinal i is
    /// truthy. Drives is_output_truthy and (unless suppress_auto_mark
    /// is set) the auto-mark loop in next. Length need not match the
    /// IR's declared output count — out-of-range ordinals are treated
    /// as non-truthy.
    std::vector<bool> output_truthy;
    /// @brief disables the default behavior of calling MarkChanged for
    /// every currently-truthy output on each next. Tests that want to
    /// model a node whose output stays truthy across cycles but only
    /// announces a change via MarkChanged on specific cycles should set
    /// this and drive MarkChanged manually from on_next.
    bool suppress_auto_mark = false;
    std::function<void(node::Context &)> on_next;

    /// @brief marks the given ordinal as truthy, growing output_truthy
    /// as needed.
    void set_truthy(const size_t ordinal) {
        if (ordinal >= output_truthy.size()) output_truthy.resize(ordinal + 1, false);
        output_truthy[ordinal] = true;
    }

    x::errors::Error next(node::Context &ctx) override {
        next_called++;
        elapsed_values.push_back(ctx.cycle.elapsed);
        if (!suppress_auto_mark)
            for (size_t i = 0; i < output_truthy.size(); ++i)
                if (output_truthy[i]) ctx.mark_changed(i);
        if (on_next) on_next(ctx);
        return x::errors::NIL;
    }

    void reset(node::Context &ctx) override {
        reset_called++;
        reset_now.push_back(ctx.cycle.now);
    }

    [[nodiscard]] bool is_output_truthy(const size_t output_idx) const override {
        if (output_idx >= output_truthy.size()) return false;
        return output_truthy[output_idx];
    }
};

/// @brief returns an on_next callback that calls mark_changed for the
/// given ordinal each time next runs. Replaces the symbolic
/// mark_on_next("name") form — the ordinal comes from the test's IR
/// declaration.
inline std::function<void(node::Context &)> mark_on_next(const size_t ordinal) {
    return [ordinal](node::Context &ctx) { ctx.mark_changed(ordinal); };
}

/// @brief collects scheduler-reported errors for assertion.
struct MockErrorHandler {
    std::vector<x::errors::Error> errors;
    errors::Handler handler = [this](const x::errors::Error &e) {
        errors.push_back(e);
    };
};

inline ir::Members stratum_of(std::vector<ir::Member> members) {
    ir::Members s;
    s.reserve(members.size());
    for (auto &m: members)
        s.push_back(std::move(m));
    return s;
}

inline ir::Scope parallel_scope(std::string key, std::vector<ir::Members> strata) {
    ir::Scope s;
    s.key = std::move(key);
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Gated;
    s.strata = std::move(strata);
    return s;
}

inline ir::Scope always_scope(std::string key, std::vector<ir::Members> strata) {
    ir::Scope s;
    s.key = std::move(key);
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Always;
    s.strata = std::move(strata);
    return s;
}

inline ir::Scope sequential_scope(
    std::string key,
    std::vector<ir::Member> steps,
    std::vector<ir::Transition> transitions = {}
) {
    ir::Scope s;
    s.key = std::move(key);
    s.mode = ir::ScopeMode::Sequential;
    s.liveness = ir::Liveness::Gated;
    s.steps = stratum_of(std::move(steps));
    s.transitions = std::move(transitions);
    return s;
}

inline ir::Scope root_scope(std::vector<ir::Member> members) {
    ir::Scope s;
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Always;
    if (!members.empty()) s.strata.push_back(stratum_of(std::move(members)));
    return s;
}

inline ir::Scope root_with_strata(std::vector<ir::Members> strata) {
    ir::Scope s;
    s.mode = ir::ScopeMode::Parallel;
    s.liveness = ir::Liveness::Always;
    s.strata = std::move(strata);
    return s;
}

inline ir::Edge continuous_edge(
    const std::string &src,
    const std::string &src_param,
    const std::string &tgt,
    const std::string &tgt_param
) {
    return ir::Edge{
        ir::Handle{src, src_param},
        ir::Handle{tgt, tgt_param},
        ir::EdgeKind::Continuous
    };
}

inline ir::Edge conditional_edge(
    const std::string &src,
    const std::string &src_param,
    const std::string &tgt,
    const std::string &tgt_param
) {
    return ir::Edge{
        ir::Handle{src, src_param},
        ir::Handle{tgt, tgt_param},
        ir::EdgeKind::Conditional
    };
}

inline std::optional<std::string> step_key_target(const std::string &key) {
    return key;
}

inline std::optional<std::string> exit_target() {
    return std::nullopt;
}

/// @brief builds an ir::Node with the given key and ordered output
/// names. The IR owns output names; ordinals used by the runtime mock
/// are this list's positions. Pass no names for a node with no outputs.
inline ir::Node
ir_node(const std::string &key, std::initializer_list<std::string> outputs = {}) {
    ir::Node n;
    n.key = key;
    for (const auto &name: outputs)
        n.outputs.push_back(arc::types::Param{.name = name});
    return n;
}

/// @brief builds an IR program from the given nodes, edges, and root
/// scope. Output names are declared per node via ir_node — the
/// scheduler reads them exclusively from ir::Node::outputs.
inline ir::IR program_of(
    std::initializer_list<ir::Node> nodes,
    std::initializer_list<ir::Edge> edges,
    ir::Scope root
) {
    ir::IR ir;
    for (const auto &n: nodes)
        ir.nodes.push_back(n);
    for (const auto &e: edges)
        ir.edges.push_back(e);
    ir.root = std::move(root);
    return ir;
}

class SchedulerTest : public ::testing::Test {
public:
    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes;
    std::unordered_map<std::string, MockNode *> mocks;

    /// @brief registers a MockNode under key with per-ordinal initial
    /// truthy values. Pass nothing for a silent mock; pass true/false
    /// per declared output ordinal otherwise. The corresponding ir::Node
    /// and its output names are declared separately at the IR layer
    /// (program_of + ir_node) — the mock is name-agnostic.
    MockNode &mock(const std::string &key, std::initializer_list<bool> truthy = {}) {
        auto node = std::make_unique<MockNode>();
        auto *ptr = node.get();
        if (truthy.size() > 0) ptr->output_truthy.assign(truthy.begin(), truthy.end());
        this->nodes[key] = std::move(node);
        this->mocks[key] = ptr;
        return *ptr;
    }

    std::unique_ptr<Scheduler> build(ir::IR ir) {
        return std::make_unique<Scheduler>(
            std::move(ir),
            this->nodes,
            x::telem::TimeSpan(0)
        );
    }

    std::unique_ptr<Scheduler> build_with_handler(ir::IR ir, errors::Handler handler) {
        return std::make_unique<Scheduler>(
            std::move(ir),
            this->nodes,
            x::telem::TimeSpan(0),
            std::move(handler)
        );
    }
};
}
