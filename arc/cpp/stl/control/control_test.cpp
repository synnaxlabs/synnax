// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/mem/indirect.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/control/control.h"
#include "arc/cpp/stl/testutil/testutil.h"

namespace arc::stl::control {
struct TestSetup {
    ir::IR ir;
    std::shared_ptr<runtime::state::State> state;

    TestSetup(const uint8_t auth_value, const uint32_t channel):
        ir(build_ir(auth_value, channel)),
        state(
            std::make_shared<runtime::state::State>(
                runtime::state::Config{.ir = ir, .channels = {}},
                runtime::errors::noop_handler
            )
        ) {}

    runtime::state::Node make_node() const {
        return ASSERT_NIL_P(this->state->node("set_auth"));
    }

private:
    static ir::IR build_ir(const uint8_t auth_value, const uint32_t channel) {
        types::Param authority_param;
        authority_param.name = "value";
        authority_param.type.kind = types::Kind::U8;
        authority_param.value = auth_value;

        types::Param channel_param;
        channel_param.name = "channel";
        channel_param.type.kind = types::Kind::U32;
        channel_param.value = channel;

        ir::Node ir_node;
        ir_node.key = "set_auth";
        ir_node.type = "set_authority";
        ir_node.inputs.push_back(authority_param);
        ir_node.inputs.push_back(channel_param);

        ir::Function fn;
        fn.key = "test";

        ir::IR ir;
        ir.nodes.push_back(ir_node);
        ir.functions.push_back(fn);
        return ir;
    }
};

runtime::node::Context make_context() {
    return runtime::node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed = [](size_t) {},
        .report_error = [](const x::errors::Error &) {},
    };
}

/// @brief declares the native's input shape for building test configs.
stl::testutil::NodeSpec set_authority() {
    types::Param value;
    value.name = "value";
    value.type = types::Type{.kind = types::Kind::U8};
    types::Param channel;
    channel.name = "channel";
    channel.type = types::Type{
        .kind = types::Kind::Chan,
        .elem = x::mem::indirect<types::Type>(types::Type{.kind = types::Kind::U8}),
        .chan_direction = types::ChanDirection::Write
    };
    stl::testutil::NodeSpec spec;
    spec.type = "set_authority";
    spec.inputs.push_back(std::move(value));
    spec.inputs.push_back(std::move(channel));
    return spec;
}

/// @brief builds a set_authority config whose inputs are all consts.
stl::testutil::Fixture const_config(const uint8_t value, const uint32_t channel) {
    return set_authority().config({x::json::json(value), x::json::json(channel)});
}

/// @brief returns a state used only to buffer authority changes.
std::shared_ptr<runtime::state::State> auth_state() {
    return std::make_shared<runtime::state::State>(
        runtime::state::Config{.ir = ir::IR{}, .channels = {}},
        runtime::errors::noop_handler
    );
}

TEST(SetAuthorityModuleTest, ReturnsErrorForNullAuthorityValue) {
    TestSetup setup(100, 42);
    auto ir_node = setup.ir.nodes[0];
    for (auto &p: ir_node.inputs)
        if (p.name == "value") p.value = nullptr;

    control::Module module(setup.state);
    ASSERT_OCCURRED_AS_P(
        module.create(runtime::node::Config(setup.ir, ir_node, setup.make_node())),
        x::errors::VALIDATION
    );
}

TEST(SetAuthorityModuleTest, CreatesNode) {
    control::Module module(auth_state());
    auto const_cfg = const_config(200, 42);
    ASSERT_NE(ASSERT_NIL_P(module.create(const_cfg.make_config())), nullptr);
    const std::shared_ptr<stl::testutil::VarBinding>
        v = std::make_shared<stl::testutil::VarInput<uint8_t>>(200);
    auto var_cfg = set_authority().config(
        {v, x::json::json(static_cast<uint32_t>(42))}
    );
    ASSERT_NE(ASSERT_NIL_P(module.create(var_cfg.make_config())), nullptr);
}

TEST(SetAuthorityModuleTest, CreatesNodeWithQualifiedTypeViaMultiFactory) {
    auto f = const_config(200, 42);
    auto ir_node = f.ir_node();
    ir_node.type = "control.set_authority";

    auto module = std::make_shared<control::Module>(auth_state());
    runtime::node::MultiFactory multi({module});
    auto node = ASSERT_NIL_P(
        multi.create(runtime::node::Config(f.program(), ir_node, f.make_node()))
    );
    ASSERT_NE(node, nullptr);
}

TEST(SetAuthorityModuleTest, ReturnsNotFoundForWrongType) {
    TestSetup setup(100, 42);
    auto ir_node = setup.ir.nodes[0];
    ir_node.type = "not_set_authority";

    control::Module module(setup.state);
    ASSERT_OCCURRED_AS_P(
        module.create(runtime::node::Config(setup.ir, ir_node, setup.make_node())),
        x::errors::NOT_FOUND
    );
}

/// @brief runs each spec twice: once with every input a const, once with the value
/// input read live from a variable.
class SetAuthorityInputsTest : public testing::TestWithParam<bool> {
protected:
    std::shared_ptr<runtime::state::State> auth = auth_state();

    stl::testutil::Fixture config(const uint8_t value, const uint32_t channel) const {
        stl::testutil::InputValue v = x::json::json(value);
        if (GetParam())
            v = std::shared_ptr<stl::testutil::VarBinding>(
                stl::testutil::var_of<uint8_t>(value)
            );
        return set_authority().config({v, x::json::json(channel)});
    }
};

INSTANTIATE_TEST_SUITE_P(
    Inputs,
    SetAuthorityInputsTest,
    testing::Values(false, true),
    [](const testing::TestParamInfo<bool> &info) {
        return info.param ? "var_value" : "const_inputs";
    }
);

TEST_P(SetAuthorityInputsTest, BuffersAPerChannelAuthorityChange) {
    auto f = this->config(200, 42);
    control::Module module(this->auth);
    auto n = ASSERT_NIL_P(module.create(f.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(n->next(ctx));
    const auto changes = this->auth->flush_authority_changes();
    ASSERT_EQ(changes.size(), 1);
    EXPECT_EQ(changes[0].authority, 200);
    ASSERT_TRUE(changes[0].channel_key.has_value());
    EXPECT_EQ(*changes[0].channel_key, 42);
}

TEST_P(SetAuthorityInputsTest, BuffersAGlobalAuthorityChange) {
    auto f = this->config(150, 0);
    control::Module module(this->auth);
    auto n = ASSERT_NIL_P(module.create(f.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(n->next(ctx));
    const auto changes = this->auth->flush_authority_changes();
    ASSERT_EQ(changes.size(), 1);
    EXPECT_EQ(changes[0].authority, 150);
    EXPECT_FALSE(changes[0].channel_key.has_value());
}

TEST_P(SetAuthorityInputsTest, FiresOnlyOnceBeforeReset) {
    auto f = this->config(200, 42);
    control::Module module(this->auth);
    auto n = ASSERT_NIL_P(module.create(f.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(n->next(ctx));
    ASSERT_NIL(n->next(ctx));
    ASSERT_NIL(n->next(ctx));
    const auto changes = this->auth->flush_authority_changes();
    ASSERT_EQ(changes.size(), 1);
    EXPECT_EQ(changes[0].authority, 200);
}

TEST_P(SetAuthorityInputsTest, DoesNotCallMarkChanged) {
    auto f = this->config(200, 42);
    control::Module module(this->auth);
    auto n = ASSERT_NIL_P(module.create(f.make_config()));
    std::vector<std::string> outputs;
    auto ctx = make_context();
    // set_authority declares no outputs; mark_changed should never fire.
    ctx.mark_changed = [&outputs](size_t) { outputs.emplace_back("called"); };
    ASSERT_NIL(n->next(ctx));
    EXPECT_TRUE(outputs.empty());
}

TEST_P(SetAuthorityInputsTest, AllowsReFireAfterReset) {
    auto f = this->config(200, 42);
    control::Module module(this->auth);
    auto n = ASSERT_NIL_P(module.create(f.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(this->auth->flush_authority_changes().size(), 1);
    n->reset();
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(this->auth->flush_authority_changes().size(), 1);
}

TEST_P(SetAuthorityInputsTest, ProducesSameAuthorityOnReFire) {
    auto f = this->config(200, 42);
    control::Module module(this->auth);
    auto n = ASSERT_NIL_P(module.create(f.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(n->next(ctx));
    const auto first = this->auth->flush_authority_changes();
    ASSERT_EQ(first.size(), 1);
    EXPECT_EQ(first[0].authority, 200);
    n->reset();
    ASSERT_NIL(n->next(ctx));
    const auto second = this->auth->flush_authority_changes();
    ASSERT_EQ(second.size(), 1);
    EXPECT_EQ(second[0].authority, first[0].authority);
    EXPECT_EQ(*second[0].channel_key, *first[0].channel_key);
}

TEST(SetAuthorityVarTest, UsesTheVarsDeclaredInitialBeforeAnyWrite) {
    const auto auth = auth_state();
    control::Module module(auth);
    const std::shared_ptr<stl::testutil::VarBinding>
        v = std::make_shared<stl::testutil::VarInput<uint8_t>>(5);
    auto f = set_authority().config({v, x::json::json(static_cast<uint32_t>(42))});
    auto n = ASSERT_NIL_P(module.create(f.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(n->next(ctx));
    const auto changes = auth->flush_authority_changes();
    ASSERT_EQ(changes.size(), 1);
    EXPECT_EQ(changes[0].authority, 5);
}

TEST(SetAuthorityVarTest, DoesNotReFireWhenTheVarChangesWithoutAReset) {
    const auto auth = auth_state();
    control::Module module(auth);
    const auto v = std::make_shared<stl::testutil::VarInput<uint8_t>>(1);
    auto f = set_authority().config(
        {std::shared_ptr<stl::testutil::VarBinding>(v),
         x::json::json(static_cast<uint32_t>(42))}
    );
    v->set(77);
    auto n = ASSERT_NIL_P(module.create(f.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(n->next(ctx));
    v->set(33);
    ASSERT_NIL(n->next(ctx));
    const auto changes = auth->flush_authority_changes();
    ASSERT_EQ(changes.size(), 1);
    EXPECT_EQ(changes[0].authority, 77);
}

TEST(SetAuthorityVarTest, ReadsTheLatestVarValueOnReFireAfterReset) {
    const auto auth = auth_state();
    control::Module module(auth);
    const auto v = std::make_shared<stl::testutil::VarInput<uint8_t>>(1);
    auto f = set_authority().config(
        {std::shared_ptr<stl::testutil::VarBinding>(v),
         x::json::json(static_cast<uint32_t>(42))}
    );
    v->set(77);
    auto n = ASSERT_NIL_P(module.create(f.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(n->next(ctx));
    EXPECT_EQ(auth->flush_authority_changes()[0].authority, 77);
    n->reset();
    v->set(33);
    ASSERT_NIL(n->next(ctx));
    const auto changes = auth->flush_authority_changes();
    ASSERT_EQ(changes.size(), 1);
    EXPECT_EQ(changes[0].authority, 33);
}
}
