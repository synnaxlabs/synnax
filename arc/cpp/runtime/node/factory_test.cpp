// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <set>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::node { namespace {

using Types = std::set<std::string>;

class MockNode final : public Node {
public:
    x::errors::Error next(Context &) override { return x::errors::NIL; }
    [[nodiscard]] bool is_output_truthy(size_t) const override { return false; }
};

/// @brief handles a fixed set of node types under one module name and records
/// every type it was asked to create.
class MockFactory final : public Factory {
    std::string module;
    Types types;
    x::errors::Error err;

public:
    std::vector<std::string> created;

    MockFactory(
        std::string module,
        Types types,
        const x::errors::Error &err = x::errors::NIL
    ):
        module(std::move(module)), types(std::move(types)), err(err) {}

    [[nodiscard]] std::string module_name() const override { return this->module; }

    [[nodiscard]] bool handles(const std::string &node_type) const override {
        return this->types.contains(node_type);
    }

    std::pair<std::unique_ptr<Node>, x::errors::Error> create(Config &&cfg) override {
        const Config owned = std::move(cfg);
        this->created.push_back(owned.node.type);
        if (this->err) return {nullptr, this->err};
        return {std::make_unique<MockNode>(), x::errors::NIL};
    }
};

struct Program {
    ir::IR ir;
    state::State state;

    explicit Program(const std::string &node_type):
        ir(build_ir(node_type)),
        state(state::Config{.ir = ir, .channels = {}}, errors::noop_handler) {}

    static ir::IR build_ir(const std::string &node_type) {
        ir::Node node;
        node.key = "n";
        node.type = node_type;
        ir::IR ir;
        ir.nodes.push_back(node);
        return ir;
    }

    Config config() {
        return Config(this->ir, this->ir.nodes[0], ASSERT_NIL_P(this->state.node("n")));
    }
};

/// @brief it should hand a dotted type to the factory that handles it whole before
/// splitting off a module prefix.
TEST(MultiFactoryTest, MatchesProgramFunctionByFullType) {
    auto host = std::make_shared<MockFactory>("", Types{"f"});
    auto fn = std::make_shared<MockFactory>("", Types{"lib.m.f"});
    MultiFactory multi({host, fn});
    Program program("lib.m.f");
    const auto node = ASSERT_NIL_P(multi.create(program.config()));
    ASSERT_NE(node, nullptr);
    ASSERT_TRUE(host->created.empty());
    ASSERT_EQ(fn->created, std::vector<std::string>{"lib.m.f"});
}

/// @brief it should split the module prefix when no factory takes the full type.
TEST(MultiFactoryTest, SplitsModulePrefixWhenNoFactoryTakesFullType) {
    auto time = std::make_shared<MockFactory>("time", Types{"now"});
    MultiFactory multi({time});
    Program program("time.now");
    const auto node = ASSERT_NIL_P(multi.create(program.config()));
    ASSERT_NE(node, nullptr);
    ASSERT_EQ(time->created, std::vector<std::string>{"now"});
}

/// @brief it should skip a factory that names another module once the prefix is
/// split.
TEST(MultiFactoryTest, SkipsFactoriesOfAnotherModuleAfterSplit) {
    auto control = std::make_shared<MockFactory>("control", Types{"set"});
    auto status = std::make_shared<MockFactory>("status", Types{"set"});
    MultiFactory multi({control, status});
    Program program("status.set");
    const auto node = ASSERT_NIL_P(multi.create(program.config()));
    ASSERT_NE(node, nullptr);
    ASSERT_TRUE(control->created.empty());
    ASSERT_EQ(status->created, std::vector<std::string>{"set"});
}

/// @brief it should return the error of the factory that owns the full type instead
/// of splitting the prefix and trying another.
TEST(MultiFactoryTest, DoesNotSplitModulePrefixAfterProgramFunctionFails) {
    const x::errors::Error failure(x::errors::VALIDATION, "bad function");
    auto fn = std::make_shared<MockFactory>("", Types{"lib.m.f"}, failure);
    auto host = std::make_shared<MockFactory>("", Types{"f"});
    MultiFactory multi({fn, host});
    Program program("lib.m.f");
    ASSERT_OCCURRED_AS_P(multi.create(program.config()), x::errors::VALIDATION);
    ASSERT_TRUE(host->created.empty());
}

/// @brief it should report NOT_FOUND when neither pass finds a factory.
TEST(MultiFactoryTest, ReportsNotFoundForUnknownType) {
    auto host = std::make_shared<MockFactory>("", Types{"f"});
    MultiFactory multi({host});
    Program program("lib.m.g");
    ASSERT_OCCURRED_AS_P(multi.create(program.config()), x::errors::NOT_FOUND);
}
}}
