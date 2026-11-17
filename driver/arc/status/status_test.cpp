// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <filesystem>
#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <utility>
#include <variant>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/testutil/testutil.h"
#include "driver/arc/status/status.h"

namespace driver::arc::status {
namespace {

::arc::runtime::node::Context make_context() {
    return ::arc::runtime::node::Context{
        .cycle =
            {.elapsed = x::telem::TimeSpan(0),
             .reason = ::arc::runtime::node::RunReason::TimerTick},
        .mark_changed = [](size_t) {},
        .mark_self_changed = [] {},
        .set_deadline = [](x::telem::TimeSpan, x::telem::TimeSpan) {},
        .report_error = [](const x::errors::Error &) {},
    };
}

// make_set_ir_node builds a `set` IR node with the {key_or_name, message, variant}
// config.
::arc::ir::Node make_set_ir_node(
    const std::string &key_or_name,
    const std::string &message,
    const std::string &variant
) {
    ::arc::ir::Node node;
    node.key = "status";
    node.type = "set";
    ::arc::types::Type str_type;
    str_type.kind = ::arc::types::Kind::String;
    ::arc::types::Param key_param;
    key_param.name = "key_or_name";
    key_param.type = str_type;
    key_param.value = key_or_name;
    ::arc::types::Param message_param;
    message_param.name = "message";
    message_param.type = str_type;
    message_param.value = message;
    ::arc::types::Param variant_param;
    variant_param.name = "variant";
    variant_param.type = std::move(str_type);
    variant_param.value = variant;
    node.inputs = ::arc::types::Params{key_param, message_param, variant_param};
    ::arc::types::Type out_type;
    out_type.kind = ::arc::types::Kind::String;
    ::arc::types::Param out;
    out.name = "output";
    out.type = std::move(out_type);
    node.outputs = ::arc::types::Params{out};
    return node;
}

::arc::types::Param str_input(const std::string &name) {
    ::arc::types::Param p;
    p.name = name;
    p.type = ::arc::types::Type{.kind = ::arc::types::Kind::String};
    return p;
}

// set_spec declares the set native's input shape for building test configs.
::arc::stl::testutil::NodeSpec set_spec() {
    ::arc::types::Param out;
    out.name = ::arc::ir::default_output_param;
    out.type = ::arc::types::Type{.kind = ::arc::types::Kind::String};
    ::arc::stl::testutil::NodeSpec spec;
    spec.type = "set";
    spec.outputs.push_back(std::move(out));
    spec.inputs.push_back(str_input("key_or_name"));
    spec.inputs.push_back(str_input("message"));
    spec.inputs.push_back(str_input("variant"));
    return spec;
}

/// @brief a report captured by recordingReporter.
struct Report {
    std::string message;
    std::string description;
};

/// @brief the reports of a recording reporter, which the outbox thread writes.
struct Reports {
    std::mutex mu;
    std::vector<Report> reports;

    std::vector<Report> snapshot() {
        std::lock_guard lock(this->mu);
        return this->reports;
    }
};

/// @brief returns a reporter that appends each report to out.
Reporter recordingReporter(const std::shared_ptr<Reports> &out) {
    return [out](const std::string &message, const std::string &description) {
        std::lock_guard lock(out->mu);
        out->reports.push_back({message, description});
    };
}

Reporter noopReporter() {
    return [](const std::string &, const std::string &) {};
}

// Unique per test to avoid cross-run contamination on a shared cluster.
std::string unique_name(const std::string &prefix) {
    return prefix + std::to_string(
                        static_cast<unsigned>(x::telem::TimeStamp::now().nanoseconds())
                    );
}

// build_ir wraps a node in a minimal ir::IR so state::State can be built.
::arc::ir::IR build_ir(::arc::ir::Node node) {
    ::arc::ir::IR ir;
    ir.nodes.push_back(std::move(node));
    ::arc::ir::Function fn;
    fn.key = "fn";
    ir.functions.push_back(fn);
    return ir;
}

}

class StatusTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::filesystem::path dir;
    std::shared_ptr<::driver::outbox::Outbox> outbox;

    void SetUp() override {
        this->client = std::make_shared<synnax::Synnax>(new_test_client());
        this->dir = std::filesystem::temp_directory_path() /
                    ("status_test_" + x::uuid::create().to_string());
        this->outbox = this->open_outbox(this->client);
    }

    void TearDown() override {
        this->outbox.reset();
        std::filesystem::remove_all(this->dir);
    }

    [[nodiscard]] std::shared_ptr<::driver::outbox::Outbox> open_outbox(
        const std::shared_ptr<synnax::Synnax> &c,
        const std::size_t cap = ::driver::outbox::DEFAULT_CAP,
        const x::telem::TimeSpan retry = x::telem::SECOND
    ) const {
        ::driver::outbox::Config cfg{
            .dir = this->dir,
            .statuses = c->statuses,
            .cap = cap,
        };
        cfg.retry.base_interval = retry;
        cfg.retry.max_interval = retry;
        return ASSERT_NIL_P(::driver::outbox::Outbox::open(cfg));
    }

    void wait_sent() const { ASSERT_EVENTUALLY_EQ(this->outbox->size(), 0); }
};

/// @brief it should handle only the set node.
TEST_F(StatusTest, HandlesSet) {
    Module module(this->outbox, noopReporter());
    EXPECT_TRUE(module.handles("set"));
    EXPECT_FALSE(module.handles("delete"));
    EXPECT_FALSE(module.handles("set_status"));
    EXPECT_FALSE(module.handles("anything_else"));
}

/// @brief it should name the module status.
TEST_F(StatusTest, ModuleNameIsStatus) {
    Module module(this->outbox, noopReporter());
    EXPECT_EQ(module.module_name(), "status");
}

/// @brief it should create a set node from the bare type.
TEST_F(StatusTest, CreatesSetNodeFromBareType) {
    auto node = make_set_ir_node("test_key", "msg", "warning");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    Module module(this->outbox, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );
    ASSERT_NE(created, nullptr);
    EXPECT_NE(dynamic_cast<SetStatus *>(created.get()), nullptr);
}

/// @brief it should create a set node from the qualified type through a multi
/// factory.
TEST_F(StatusTest, CreatesNodeWithQualifiedTypeViaMultiFactory) {
    auto node = make_set_ir_node("test_key", "msg", "warning");
    node.type = "status.set";
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    auto module = std::make_shared<Module>(this->outbox, noopReporter());
    ::arc::runtime::node::MultiFactory multi({module});
    auto created = ASSERT_NIL_P(
        multi.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );
    ASSERT_NE(created, nullptr);
    EXPECT_NE(dynamic_cast<SetStatus *>(created.get()), nullptr);
}

/// @brief it should return NOT_FOUND for an unknown node type.
TEST_F(StatusTest, ReturnsNotFoundForUnknownType) {
    auto node = make_set_ir_node("test_key", "msg", "warning");
    node.type = "not_set_or_delete";
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    Module module(this->outbox, noopReporter());
    const auto created = ASSERT_OCCURRED_AS_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st))),
        x::errors::NOT_FOUND
    );
    EXPECT_EQ(created, nullptr);
}

/// @brief it should read a var-bound message when the node runs.
TEST_F(StatusTest, ReadsAVarBoundMessageAtFireTime) {
    const auto reports = std::make_shared<Reports>();
    const auto name = unique_name("var_msg_");
    const auto f = set_spec().config(
        {name,
         std::shared_ptr<::arc::stl::testutil::VarBinding>(
             ::arc::stl::testutil::var_of<std::string>("live message")
         ),
         "info"}
    );
    Module module(this->outbox, recordingReporter(reports));
    auto created = ASSERT_NIL_P(module.create(f.make_config()));

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));
    this->wait_sent();
    ASSERT_NIL(created->next(ctx));

    const auto node = f.make_node();
    const auto new_key = std::get<std::string>((*node.output(0)).at(0));
    const auto retrieved = ASSERT_NIL_P(this->client->statuses.retrieve(new_key));
    EXPECT_EQ(retrieved.message, "live message");
    EXPECT_TRUE(reports->snapshot().empty());
}

/// @brief it should output an empty key before the first set resolves, and the key
/// of the last resolved set after.
TEST_F(StatusTest, OutputsLastResolvedKey) {
    const auto name = unique_name("set_next_");
    const auto f = set_spec().config({name, "the message", "info"});
    Module module(this->outbox, noopReporter());
    auto created = ASSERT_NIL_P(module.create(f.make_config()));
    const auto node = f.make_node();
    auto ctx = make_context();

    ASSERT_NIL(created->next(ctx));
    EXPECT_EQ(std::get<std::string>((*node.output(0)).at(0)), "");
    this->wait_sent();
    ASSERT_NIL(created->next(ctx));
    const auto key = std::get<std::string>((*node.output(0)).at(0));
    ASSERT_FALSE(key.empty());
    const auto retrieved = ASSERT_NIL_P(this->client->statuses.retrieve(key));
    EXPECT_EQ(retrieved.name, name);
    EXPECT_EQ(retrieved.message, "the message");
    EXPECT_EQ(retrieved.variant, "info");
}

/// @brief it should stamp the status with the time the node runs.
TEST_F(StatusTest, NextStampsTimeWhenItRuns) {
    const auto name = unique_name("set_time_");
    const auto preset_key = ASSERT_NIL_P(this->client->statuses.set_by_key_or_name(
                                             name,
                                             "initial",
                                             "info",
                                             x::telem::TimeStamp(1)
                                         ))
                                .key;
    const auto f = set_spec().config({preset_key, "the message", "info"});
    Module module(this->outbox, noopReporter());
    auto created = ASSERT_NIL_P(module.create(f.make_config()));

    auto ctx = make_context();
    const auto before = x::telem::TimeStamp::now();
    ASSERT_NIL(created->next(ctx));
    const auto after = x::telem::TimeStamp::now();
    this->wait_sent();

    const auto retrieved = ASSERT_NIL_P(this->client->statuses.retrieve(preset_key));
    EXPECT_EQ(retrieved.message, "the message");
    EXPECT_GE(retrieved.time, before);
    EXPECT_LE(retrieved.time, after);
}

/// @brief it should send every set, in order.
TEST_F(StatusTest, NextRepeatedCallsKeepWriting) {
    const auto name = unique_name("set_repeat_");
    const auto preset_key = ASSERT_NIL_P(this->client->statuses.set_by_key_or_name(
                                             name,
                                             "initial",
                                             "info",
                                             x::telem::TimeStamp()
                                         ))
                                .key;
    const auto f = set_spec().config({preset_key, "msg", "info"});
    Module module(this->outbox, noopReporter());
    auto created = ASSERT_NIL_P(module.create(f.make_config()));

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));
    ASSERT_NIL(created->next(ctx));
    ASSERT_NIL(created->next(ctx));
    this->wait_sent();
    const auto retrieved = ASSERT_NIL_P(this->client->statuses.retrieve(preset_key));
    EXPECT_EQ(retrieved.message, "msg");
}

/// @brief it should warn when the Core rejects a set, and drop the set.
TEST_F(StatusTest, NextWarnsOnInvalidVariant) {
    const auto reports = std::make_shared<Reports>();
    const auto f = set_spec().config({unique_name("set_iv_"), "msg", "bogus"});
    Module module(this->outbox, recordingReporter(reports));
    auto created = ASSERT_NIL_P(module.create(f.make_config()));

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));
    this->wait_sent();
    EXPECT_EQ(this->outbox->dropped(), 1);
    const auto calls = reports->snapshot();
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].message, "Failed to set status");
    EXPECT_NE(calls[0].description.find("invalid status variant"), std::string::npos);
}

/// @brief it should queue a set and return at once while the Core is unreachable.
TEST_F(StatusTest, NeverBlocksWithCoreUnreachable) {
    this->outbox.reset();
    auto cfg = test_client_config();
    cfg.port = 1;
    this->outbox = this->open_outbox(std::make_shared<synnax::Synnax>(cfg));
    const auto reports = std::make_shared<Reports>();
    const auto f = set_spec().config({unique_name("set_down_"), "msg", "info"});
    Module module(this->outbox, recordingReporter(reports));
    auto created = ASSERT_NIL_P(module.create(f.make_config()));
    const auto node = f.make_node();

    auto ctx = make_context();
    const auto start = x::telem::TimeStamp::now();
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(created->next(ctx));
    EXPECT_LT(x::telem::TimeStamp::now() - start, x::telem::SECOND);
    EXPECT_EQ(std::get<std::string>((*node.output(0)).at(0)), "");
    EXPECT_EQ(this->outbox->size(), 3);
    EXPECT_EQ(this->outbox->dropped(), 0);
    EXPECT_TRUE(reports->snapshot().empty());
}

/// @brief it should check output truthiness without throwing.
TEST_F(StatusTest, IsOutputTruthyDoesNotThrow) {
    auto node = make_set_ir_node("truthy", "msg", "info");
    auto ir = build_ir(node);
    ::arc::runtime::state::State s(
        ::arc::runtime::state::Config{.ir = ir, .channels = {}},
        ::arc::runtime::errors::noop_handler
    );
    auto st = ASSERT_NIL_P(s.node("status"));
    Module module(this->outbox, noopReporter());
    auto created = ASSERT_NIL_P(
        module.create(::arc::runtime::node::Config(ir, ir.nodes[0], std::move(st)))
    );
    EXPECT_NO_THROW({ (void) created->is_output_truthy(0); });
}

/// @brief it should make its output truthy once it queues a set, before the set
/// resolves, and falsy again after a reset.
TEST_F(StatusTest, OutputIsTruthyOnceQueued) {
    this->outbox.reset();
    auto cfg = test_client_config();
    cfg.port = 1;
    this->outbox = this->open_outbox(std::make_shared<synnax::Synnax>(cfg));
    const auto f = set_spec().config({unique_name("set_truthy_"), "msg", "info"});
    Module module(this->outbox, noopReporter());
    auto created = ASSERT_NIL_P(module.create(f.make_config()));

    auto ctx = make_context();
    EXPECT_FALSE(created->is_output_truthy(0));
    ASSERT_NIL(created->next(ctx));
    EXPECT_EQ(std::get<std::string>((*f.make_node().output(0)).at(0)), "");
    EXPECT_TRUE(created->is_output_truthy(0));
    created->reset(ctx);
    EXPECT_FALSE(created->is_output_truthy(0));
}

/// @brief it should report a set the full outbox refuses from the outbox thread, with
/// no later set, since the caller must never wait on the Core.
TEST_F(StatusTest, ReportsARefusedSetFromTheOutboxThread) {
    this->outbox.reset();
    auto cfg = test_client_config();
    cfg.port = 1;
    this->outbox = this->open_outbox(
        std::make_shared<synnax::Synnax>(cfg),
        1,
        x::telem::MILLISECOND
    );
    const auto reports = std::make_shared<Reports>();
    const auto caller = std::this_thread::get_id();
    std::atomic<bool> on_caller = false;
    const auto record = recordingReporter(reports);
    const auto f = set_spec().config({unique_name("set_full_"), "msg", "info"});
    Module module(
        this->outbox,
        [&, record](const std::string &message, const std::string &description) {
            if (std::this_thread::get_id() == caller) on_caller = true;
            record(message, description);
        }
    );
    auto created = ASSERT_NIL_P(module.create(f.make_config()));

    auto ctx = make_context();
    ASSERT_NIL(created->next(ctx));
    ASSERT_NIL(created->next(ctx));
    EXPECT_EQ(this->outbox->refused(), 1);
    ASSERT_EVENTUALLY_FALSE(reports->snapshot().empty());
    const auto got = reports->snapshot();
    EXPECT_EQ(got[0].message, SET_FAILURE_MSG);
    EXPECT_NE(got[0].description.find("Dropped 1 sets."), std::string::npos);
    EXPECT_FALSE(on_caller.load());
}

/// @brief it should report the refused sets once, with their count and reason.
TEST(SetsTest, ReportsRefusedSetsOnce) {
    const auto reports = std::make_shared<Reports>();
    Sets sets(recordingReporter(reports));
    sets.refuse("outbox is full.");
    sets.refuse("outbox is full.");
    sets.report_refused();
    sets.report_refused();
    const auto calls = reports->snapshot();
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].message, SET_FAILURE_MSG);
    EXPECT_EQ(calls[0].description, "outbox is full. Dropped 2 sets.");
}

}
