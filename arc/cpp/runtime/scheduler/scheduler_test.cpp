// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <functional>
#include <memory>
#include <string>
#include <unordered_map>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/ir/testutil/testutil.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/time/time.h"

namespace arc::runtime::scheduler {
/// @brief Configurable mock node for testing scheduler behavior.
struct MockNode final : public node::Node {
    int next_called = 0;
    int reset_called = 0;
    std::vector<x::telem::TimeSpan> elapsed_values;

    std::unordered_map<std::string, bool> param_truthy;
    std::function<void(node::Context &)> on_next;
    x::errors::Error next_error = x::errors::NIL;

    x::errors::Error next(node::Context &ctx) override {
        next_called++;
        elapsed_values.push_back(ctx.elapsed);
        if (on_next) on_next(ctx);
        return next_error;
    }

    void reset() override { reset_called++; }

    [[nodiscard]] bool is_output_truthy(const std::string &param) const override {
        const auto it = param_truthy.find(param);
        return it != param_truthy.end() && it->second;
    }

    /// @brief Configure node to mark a parameter as changed when next() is called.
    void mark_on_next(const std::string &param) {
        on_next = [param](const node::Context &ctx) { ctx.mark_changed(param); };
    }

    /// @brief Configure node to activate stage when next() is called.
    void activate_on_next() {
        on_next = [](const node::Context &ctx) { ctx.activate_stage(); };
    }

    /// @brief Configure node to report an error when next() is called.
    void error_on_next(const x::errors::Error &err) {
        on_next = [err](const node::Context &ctx) { ctx.report_error(err); };
    }
};

class SchedulerTest : public ::testing::Test {
protected:
    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes_;
    std::unordered_map<std::string, MockNode *> mocks_;

    MockNode &mock(const std::string &key) {
        auto node = std::make_unique<MockNode>();
        auto *ptr = node.get();
        nodes_[key] = std::move(node);
        mocks_[key] = ptr;
        return *ptr;
    }

    std::unique_ptr<Scheduler> build(ir::IR ir) {
        return std::make_unique<Scheduler>(ir, nodes_, x::telem::TimeSpan(0));
    }
};

/// @brief it should construct with an empty program
TEST_F(SchedulerTest, testConstructsWithEmptyProgram) {
    ir::IR ir;
    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
}

/// @brief it should construct with a single stratum and execute all nodes
TEST_F(SchedulerTest, testConstructsWithSingleStratum) {
    mock("A");
    mock("B");
    mock("C");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .node("C")
                  .strata({{"A", "B", "C"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(mocks_["A"]->next_called, 1);
    ASSERT_EQ(mocks_["B"]->next_called, 1);
    ASSERT_EQ(mocks_["C"]->next_called, 1);
}

/// @brief it should build a transition table for stage activation
TEST_F(SchedulerTest, testBuildsTransitionTable) {
    // Trigger nodes at stratum 0 activate entry nodes at stratum 1
    auto &trigger_a = mock("trigger_a");
    mock("trigger_b"); // Trigger for stage_b (not activated in this test)
    auto &entry_a = mock("entry_seq_stage_a");
    mock("entry_seq_stage_b"); // Entry for stage_b
    mock("A");
    mock("B");

    trigger_a.mark_on_next("activate");
    trigger_a.param_truthy["activate"] = true;
    entry_a.activate_on_next();

    auto ir = ir::testutil::Builder()
                  .node("trigger_a")
                  .node("trigger_b")
                  .node("entry_seq_stage_a")
                  .node("entry_seq_stage_b")
                  .node("A")
                  .node("B")
                  .oneshot("trigger_a", "activate", "entry_seq_stage_a", "input")
                  .oneshot("trigger_b", "activate", "entry_seq_stage_b", "input")
                  .strata(
                      {{"trigger_a", "trigger_b"},
                       {"entry_seq_stage_a", "entry_seq_stage_b"}}
                  )
                  .sequence("seq", {{"stage_a", {{"A"}}}, {"stage_b", {{"B"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    // If transition table built correctly, stage_a should be active
    ASSERT_EQ(mocks_["A"]->next_called, 1);
}

/// @brief it should always execute stratum 0 on every next() call
TEST_F(SchedulerTest, testStratum0AlwaysExecutes) {
    const auto &nodeA = mock("A");

    auto ir = ir::testutil::Builder().node("A").strata({{"A"}}).build();

    const auto scheduler = build(std::move(ir));

    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    scheduler->next(x::telem::MILLISECOND * 3, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 3);
}

/// @brief it should skip higher strata when no changes are propagated
TEST_F(SchedulerTest, testHigherStrataSkipWithoutChanges) {
    const auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .edge("A", "output", "B", "input")
                  .strata({{"A"}, {"B"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 0);
}

/// @brief it should pass the correct elapsed time to the node context
TEST_F(SchedulerTest, testElapsedTimePassedToContext) {
    const auto &nodeA = mock("A");

    auto ir = ir::testutil::Builder().node("A").strata({{"A"}}).build();

    const auto scheduler = build(std::move(ir));

    scheduler->next(x::telem::MILLISECOND * 5, node::RunReason::TimerTick);
    scheduler->next(x::telem::MILLISECOND * 10, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.elapsed_values.size(), 2);
    ASSERT_EQ(nodeA.elapsed_values[0], x::telem::MILLISECOND * 5);
    ASSERT_EQ(nodeA.elapsed_values[1], x::telem::MILLISECOND * 10);
}

TEST_F(SchedulerTest, testRunReasonPassedToNode) {
    auto &nodeA = mock("A");

    std::vector<node::RunReason> received_reasons;
    nodeA.on_next = [&received_reasons](const node::Context &ctx) {
        received_reasons.push_back(ctx.reason);
    };

    auto ir = ir::testutil::Builder().node("A").strata({{"A"}}).build();
    const auto scheduler = build(std::move(ir));

    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::ChannelInput);
    scheduler->next(x::telem::MILLISECOND * 3, node::RunReason::TimerTick);

    ASSERT_EQ(received_reasons.size(), 3);
    ASSERT_EQ(received_reasons[0], node::RunReason::TimerTick);
    ASSERT_EQ(received_reasons[1], node::RunReason::ChannelInput);
    ASSERT_EQ(received_reasons[2], node::RunReason::TimerTick);
}

/// @brief it should accumulate execution counts across multiple next() calls
TEST_F(SchedulerTest, testMultipleNextCallsAccumulate) {
    const auto &nodeA = mock("A");

    auto ir = ir::testutil::Builder().node("A").strata({{"A"}}).build();

    const auto scheduler = build(std::move(ir));

    for (int i = 0; i < 100; i++)
        scheduler->next(x::telem::MILLISECOND * i, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 100);
}

/// @brief it should handle empty strata without crashing
TEST_F(SchedulerTest, testEmptyStrataDoesNotCrash) {
    mock("A");
    mock("B");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .strata({{"A"}, {}, {"B"}}) // Empty middle stratum
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    // A executes (stratum 0), B doesn't execute (stratum 2, not changed)
    ASSERT_EQ(mocks_["A"]->next_called, 1);
    ASSERT_EQ(mocks_["B"]->next_called, 0);
}

/// @brief it should clear the changed set between strata executions
TEST_F(SchedulerTest, testChangedSetClearsPerStrataExecution) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");
    mock("C");

    // A marks changed on first call only
    bool first_call = true;
    nodeA.on_next = [&first_call](const node::Context &ctx) {
        if (first_call) {
            ctx.mark_changed("output");
            first_call = false;
        }
    };

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .node("C")
                  .edge("A", "output", "B", "input")
                  .strata({{"A"}, {"B"}, {"C"}})
                  .build();

    const auto scheduler = build(std::move(ir));

    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);

    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);
}

/// @brief it should propagate changes through continuous edges
TEST_F(SchedulerTest, testMarkChangedPropagatesContinuousEdge) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    nodeA.mark_on_next("output");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .edge("A", "output", "B", "input")
                  .strata({{"A"}, {"B"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 1);
}

/// @brief it should not propagate changes when there is no edge
TEST_F(SchedulerTest, testMarkChangedDoesNotPropagateWithoutEdge) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    nodeA.mark_on_next("output");

    // No edge between A and B
    auto
        ir = ir::testutil::Builder().node("A").node("B").strata({{"A"}, {"B"}}).build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 0);
}

/// @brief it should only propagate to nodes connected to the marked output parameter
TEST_F(SchedulerTest, testMultipleOutputsFromSingleNode) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");
    const auto &nodeC = mock("C");

    // A marks only "output_x"
    nodeA.mark_on_next("output_x");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .node("C")
                  .edge("A", "output_x", "B", "input")
                  .edge("A", "output_y", "C", "input")
                  .strata({{"A"}, {"B", "C"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeB.next_called, 1); // Connected to output_x
    ASSERT_EQ(nodeC.next_called, 0); // Connected to output_y (not marked)
}

/// @brief it should execute a node when any of its inputs are marked changed
TEST_F(SchedulerTest, testMultipleInputsToSingleNode) {
    auto &nodeA = mock("A");
    auto &nodeB = mock("B");
    const auto &nodeC = mock("C");

    nodeA.mark_on_next("output");
    nodeB.mark_on_next("output");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .node("C")
                  .edge("A", "output", "C", "input_a")
                  .edge("B", "output", "C", "input_b")
                  .strata({{"A", "B"}, {"C"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeC.next_called, 1);
}

/// @brief it should respect parameter-specific edge connections
TEST_F(SchedulerTest, testParameterSpecificEdges) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");
    const auto &nodeC = mock("C");

    // A marks "param_a", not "param_b"
    nodeA.mark_on_next("param_a");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .node("C")
                  .edge("A", "param_a", "B", "input")
                  .edge("A", "param_b", "C", "input")
                  .strata({{"A"}, {"B", "C"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeB.next_called, 1);
    ASSERT_EQ(nodeC.next_called, 0);
}

/// @brief it should propagate changes through a chain of nodes
TEST_F(SchedulerTest, testChainedPropagation) {
    auto &nodeA = mock("A");
    auto &nodeB = mock("B");
    const auto &nodeC = mock("C");

    nodeA.mark_on_next("output");
    nodeB.mark_on_next("output");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .node("C")
                  .edge("A", "output", "B", "input")
                  .edge("B", "output", "C", "input")
                  .strata({{"A"}, {"B"}, {"C"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 1);
    ASSERT_EQ(nodeC.next_called, 1);
}

/// @brief it should handle diamond dependency patterns correctly
TEST_F(SchedulerTest, testDiamondDependency) {
    auto &nodeA = mock("A");
    auto &nodeB = mock("B");
    auto &nodeC = mock("C");
    const auto &nodeD = mock("D");

    nodeA.mark_on_next("output");
    nodeB.mark_on_next("output");
    nodeC.mark_on_next("output");

    // Diamond: A -> B -> D, A -> C -> D
    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .node("C")
                  .node("D")
                  .edge("A", "output", "B", "input")
                  .edge("A", "output", "C", "input")
                  .edge("B", "output", "D", "input_b")
                  .edge("C", "output", "D", "input_c")
                  .strata({{"A"}, {"B", "C"}, {"D"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeD.next_called, 1);
}

/// @brief it should execute many nodes in parallel within a single stratum
TEST_F(SchedulerTest, testWideGraph) {
    // 10 nodes in stratum 0, all independent
    for (int i = 0; i < 10; i++)
        mock("N" + std::to_string(i));

    std::vector<std::string> stratum0;
    for (int i = 0; i < 10; i++)
        stratum0.push_back("N" + std::to_string(i));

    auto builder = ir::testutil::Builder();
    for (int i = 0; i < 10; i++)
        builder.node("N" + std::to_string(i));
    auto ir = builder.strata({stratum0}).build();

    auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    for (int i = 0; i < 10; i++)
        ASSERT_EQ(mocks_["N" + std::to_string(i)]->next_called, 1);
}

/// @brief it should fire one-shot edges when the output is truthy
TEST_F(SchedulerTest, testOneShotFiresWhenTruthy) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    nodeA.mark_on_next("output");
    nodeA.param_truthy["output"] = true;

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .oneshot("A", "output", "B", "input")
                  .strata({{"A"}, {"B"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeB.next_called, 1);
}

/// @brief it should not fire one-shot edges when the output is falsy
TEST_F(SchedulerTest, testOneShotDoesNotFireWhenFalsy) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    nodeA.mark_on_next("output");
    nodeA.param_truthy["output"] = false;

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .oneshot("A", "output", "B", "input")
                  .strata({{"A"}, {"B"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeB.next_called, 0);
}

/// @brief it should fire one-shot edges only once per stage activation
TEST_F(SchedulerTest, testOneShotFiresOnlyOncePerStage) {
    // Trigger at stratum 0, entry at stratum 1
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();
    nodeA.mark_on_next("output");
    nodeA.param_truthy["output"] = true;

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("A")
                  .node("B")
                  .oneshot("trigger", "activate", "entry_seq_stage", "input")
                  .oneshot("A", "output", "B", "input")
                  .strata({{"trigger"}, {"entry_seq_stage"}})
                  .sequence("seq", {{"stage", {{"A"}, {"B"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));

    // First call: trigger→entry one-shot fires, stage activates, A→B one-shot fires
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);

    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);
}

/// @brief it should fire one-shot edges only once in global strata
TEST_F(SchedulerTest, testOneShotFiresOnceInGlobalStrata) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    nodeA.mark_on_next("output");
    nodeA.param_truthy["output"] = true;

    // One-shot in global strata fires once ever (no reset mechanism)
    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .oneshot("A", "output", "B", "input")
                  .strata({{"A"}, {"B"}})
                  .build();

    const auto scheduler = build(std::move(ir));

    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);

    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);

    scheduler->next(x::telem::MILLISECOND * 3, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);
}

/// @brief it should reset one-shot edges when a stage is re-entered
TEST_F(SchedulerTest, testOneShotResetsOnStageEntry) {
    // Use continuous edge for re-triggering to verify one-shots reset on stage re-entry
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();
    nodeA.mark_on_next("output");
    nodeA.param_truthy["output"] = true;

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("A")
                  .node("B")
                  // Global: continuous edge so it triggers every time
                  .edge("trigger", "activate", "entry_seq_stage", "input")
                  // Stage: A→B one-shot
                  .oneshot("A", "output", "B", "input")
                  .strata({{"trigger"}, {"entry_seq_stage"}})
                  .sequence("seq", {{"stage", {{"A"}, {"B"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));

    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);
    ASSERT_EQ(nodeA.reset_called, 1);

    // Stage re-activates via continuous edge, clearing fired_one_shots
    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 2);
    ASSERT_EQ(nodeA.reset_called, 2);
}

/// @brief it should propagate continuous edges regardless of truthiness
TEST_F(SchedulerTest, testContinuousEdgeUnaffectedByTruthiness) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    nodeA.mark_on_next("output");
    nodeA.param_truthy["output"] = false; // Falsy, but continuous edge

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .edge("A", "output", "B", "input")
                  .strata({{"A"}, {"B"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeB.next_called, 1);
}

/// @brief it should not execute staged nodes when no stage is active
TEST_F(SchedulerTest, testNoExecutionWhenNoStageActive) {
    mock("A");
    const auto &nodeB = mock("B");

    // No entry node activates stage
    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .strata({{"A"}})
                  .sequence("seq", {{"stage", {{"B"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeB.next_called, 0);
}

/// @brief it should execute staged nodes when their stage is active
TEST_F(SchedulerTest, testStagedNodesExecuteWhenActive) {
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    const auto &nodeA = mock("A");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("A")
                  .oneshot("trigger", "activate", "entry_seq_stage", "input")
                  .strata({{"trigger"}, {"entry_seq_stage"}})
                  .sequence("seq", {{"stage", {{"A"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1);
}

/// @brief it should always execute global strata regardless of stage activation
TEST_F(SchedulerTest, testGlobalStrataAlwaysExecutes) {
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    const auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("A")
                  .node("B")
                  .oneshot("trigger", "activate", "entry_seq_stage", "input")
                  .strata(
                      {{"trigger", "A"}, {"entry_seq_stage"}}
                  ) // A is global at stratum 0
                  .sequence("seq", {{"stage", {{"B"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1); // Global
    ASSERT_EQ(nodeB.next_called, 1); // Stage
}

/// @brief it should activate a stage when its entry node is triggered
TEST_F(SchedulerTest, testEntryNodeActivatesStage) {
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    const auto &nodeA = mock("A");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("A")
                  .oneshot("trigger", "activate", "entry_seq_stage", "input")
                  .strata({{"trigger"}, {"entry_seq_stage"}})
                  .sequence("seq", {{"stage", {{"A"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    ASSERT_EQ(nodeA.next_called, 0);
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);
}

/// @brief it should deactivate the previous stage when transitioning to a new stage
TEST_F(SchedulerTest, testStageTransitionDeactivatesPrevious) {
    auto &trigger = mock("trigger");
    auto &entry_a = mock("entry_seq_stage_a");
    auto &entry_b = mock("entry_seq_stage_b");
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_a.activate_on_next();
    entry_b.activate_on_next();
    nodeA.mark_on_next("to_b");
    nodeA.param_truthy["to_b"] = true;

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage_a")
                  .node("entry_seq_stage_b")
                  .node("A")
                  .node("B")
                  .oneshot("trigger", "activate", "entry_seq_stage_a", "input")
                  .oneshot("A", "to_b", "entry_seq_stage_b", "input")
                  .strata({{"trigger"}, {"entry_seq_stage_a", "entry_seq_stage_b"}})
                  .sequence(
                      "seq",
                      {{"stage_a", {{"A"}, {"entry_seq_stage_b"}}},
                       {"stage_b", {{"B"}}}}
                  )
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 1);

    // Stage_b remains active, stage_a deactivated
    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 2);
}

/// @brief it should reset nodes when entering a new stage
TEST_F(SchedulerTest, testStageTransitionResetsNodes) {
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    const auto &nodeA = mock("A");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("A")
                  .oneshot("trigger", "activate", "entry_seq_stage", "input")
                  .strata({{"trigger"}, {"entry_seq_stage"}})
                  .sequence("seq", {{"stage", {{"A"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));

    ASSERT_EQ(nodeA.reset_called, 0);
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.reset_called, 1);
}

/// @brief it should maintain independence between different sequences
TEST_F(SchedulerTest, testCrossSequenceIndependence) {
    auto &trigger1 = mock("trigger1");
    auto &trigger2 = mock("trigger2");
    auto &entry1 = mock("entry_seq1_stage");
    auto &entry2 = mock("entry_seq2_stage");
    const auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    trigger1.mark_on_next("activate");
    trigger1.param_truthy["activate"] = true;
    trigger2.mark_on_next("activate");
    trigger2.param_truthy["activate"] = true;
    entry1.activate_on_next();
    entry2.activate_on_next();

    auto ir = ir::testutil::Builder()
                  .node("trigger1")
                  .node("trigger2")
                  .node("entry_seq1_stage")
                  .node("entry_seq2_stage")
                  .node("A")
                  .node("B")
                  .oneshot("trigger1", "activate", "entry_seq1_stage", "input")
                  .oneshot("trigger2", "activate", "entry_seq2_stage", "input")
                  .strata(
                      {{"trigger1", "trigger2"},
                       {"entry_seq1_stage", "entry_seq2_stage"}}
                  )
                  .sequence("seq1", {{"stage", {{"A"}}}})
                  .sequence("seq2", {{"stage", {{"B"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    // Both sequences active independently
    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 1);
}

/// @brief it should support transitioning through multiple stages in a sequence
TEST_F(SchedulerTest, testMultipleStagesInSequence) {
    // Test transitioning through A→B→C stages via internal edges
    auto &trigger = mock("trigger");
    auto &entry_a = mock("entry_seq_stage_a");
    auto &entry_b = mock("entry_seq_stage_b");
    auto &entry_c = mock("entry_seq_stage_c");
    auto &nodeA = mock("A");
    auto &nodeB = mock("B");
    const auto &nodeC = mock("C");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_a.activate_on_next();
    entry_b.activate_on_next();
    entry_c.activate_on_next();
    nodeA.mark_on_next("to_b");
    nodeA.param_truthy["to_b"] = true;
    nodeB.mark_on_next("to_c");
    nodeB.param_truthy["to_c"] = true;

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage_a")
                  .node("entry_seq_stage_b")
                  .node("entry_seq_stage_c")
                  .node("A")
                  .node("B")
                  .node("C")
                  .oneshot("trigger", "activate", "entry_seq_stage_a", "input")
                  .oneshot("A", "to_b", "entry_seq_stage_b", "input")
                  .oneshot("B", "to_c", "entry_seq_stage_c", "input")
                  .strata(
                      {{"trigger"},
                       {"entry_seq_stage_a", "entry_seq_stage_b", "entry_seq_stage_c"}}
                  )
                  .sequence(
                      "seq",
                      {{"stage_a", {{"A"}, {"entry_seq_stage_b"}}},
                       {"stage_b", {{"B"}, {"entry_seq_stage_c"}}},
                       {"stage_c", {{"C"}}}}
                  )
                  .build();

    const auto scheduler = build(std::move(ir));

    // Single next() cascades through all stages: A→B→C
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 1);
    ASSERT_EQ(nodeC.next_called, 1);
}

/// @brief it should converge after a single stage transition
TEST_F(SchedulerTest, testSingleTransitionConverges) {
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    const auto &nodeA = mock("A");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("A")
                  .oneshot("trigger", "activate", "entry_seq_stage", "input")
                  .strata({{"trigger"}, {"entry_seq_stage"}})
                  .sequence("seq", {{"stage", {{"A"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);
}

/// @brief it should complete cascading stage transitions in a single next() call
TEST_F(SchedulerTest, testCascadingTransitionsComplete) {
    // A→B→C stage transitions complete in single next() via convergence loop
    auto &trigger = mock("trigger");
    auto &entry_a = mock("entry_seq_stage_a");
    auto &entry_b = mock("entry_seq_stage_b");
    auto &entry_c = mock("entry_seq_stage_c");
    auto &nodeA = mock("A");
    auto &nodeB = mock("B");
    const auto &nodeC = mock("C");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_a.activate_on_next();
    entry_b.activate_on_next();
    entry_c.activate_on_next();
    nodeA.mark_on_next("output");
    nodeB.mark_on_next("output");

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage_a")
                  .node("entry_seq_stage_b")
                  .node("entry_seq_stage_c")
                  .node("A")
                  .node("B")
                  .node("C")
                  .oneshot("trigger", "activate", "entry_seq_stage_a", "input")
                  .edge("A", "output", "entry_seq_stage_b", "input")
                  .edge("B", "output", "entry_seq_stage_c", "input")
                  .strata({{"trigger"}, {"entry_seq_stage_a"}})
                  .sequence(
                      "seq",
                      {{"stage_a", {{"A"}, {"entry_seq_stage_b"}}},
                       {"stage_b", {{"B"}, {"entry_seq_stage_c"}}},
                       {"stage_c", {{"C"}}}}
                  )
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 1);
    ASSERT_EQ(nodeC.next_called, 1);
}

/// @brief it should stop convergence iterations when the system becomes stable
TEST_F(SchedulerTest, testConvergenceStopsWhenStable) {
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    const auto &nodeA = mock("A");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("A")
                  .oneshot("trigger", "activate", "entry_seq_stage", "input")
                  .strata({{"trigger"}, {"entry_seq_stage"}})
                  .sequence("seq", {{"stage", {{"A"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 2);
}

/// @brief it should prevent infinite loops via maximum iteration limit
TEST_F(SchedulerTest, testMaxIterationsPreventInfiniteLoop) {
    // Pathological case: node keeps re-triggering same stage entry
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    auto &nodeA = mock("A");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();
    // A triggers entry which re-activates the stage (infinite loop attempt)
    nodeA.mark_on_next("reenter");
    nodeA.param_truthy["reenter"] = true;

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("A")
                  .oneshot("trigger", "activate", "entry_seq_stage", "input")
                  // A triggers entry inside the stage (would cause infinite re-entry)
                  .oneshot("A", "reenter", "entry_seq_stage", "input")
                  .strata({{"trigger"}, {"entry_seq_stage"}})
                  .sequence("seq", {{"stage", {{"A"}, {"entry_seq_stage"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_TRUE(true);
}

/// @brief it should detect stage transitions during convergence
TEST_F(SchedulerTest, testConvergenceDetectsTransition) {
    auto &trigger = mock("trigger");
    auto &entry_a = mock("entry_seq_stage_a");
    auto &entry_b = mock("entry_seq_stage_b");
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_a.activate_on_next();
    entry_b.activate_on_next();
    nodeA.mark_on_next("output");

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage_a")
                  .node("entry_seq_stage_b")
                  .node("A")
                  .node("B")
                  .oneshot("trigger", "activate", "entry_seq_stage_a", "input")
                  .edge("A", "output", "entry_seq_stage_b", "input")
                  .strata({{"trigger"}, {"entry_seq_stage_a"}})
                  .sequence(
                      "seq",
                      {{"stage_a", {{"A"}, {"entry_seq_stage_b"}}},
                       {"stage_b", {{"B"}}}}
                  )
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 1);
}

/// @brief it should receive errors reported by nodes
TEST_F(SchedulerTest, testErrorHandlerReceivesErrors) {
    auto &nodeA = mock("A");

    nodeA.error_on_next(x::errors::Error("test", "test error"));

    auto ir = ir::testutil::Builder().node("A").strata({{"A"}}).build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);
}

/// @brief it should continue execution after a node reports an error
TEST_F(SchedulerTest, testExecutionContinuesAfterError) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    nodeA.error_on_next(x::errors::Error("test", "error from A"));
    nodeA.mark_on_next("output");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .edge("A", "output", "B", "input")
                  .strata({{"A"}, {"B"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);
}

/// @brief it should return normally even when a node throws an error
TEST_F(SchedulerTest, testNextReturnsNormally) {
    auto &nodeA = mock("A");

    nodeA.next_error = x::errors::Error("test", "node error");

    auto ir = ir::testutil::Builder().node("A").strata({{"A"}}).build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_TRUE(true);
}

/// @brief it should handle a deep chain of strata with proper propagation
TEST_F(SchedulerTest, testDeepStrataChain) {
    // 10 strata deep
    for (int i = 0; i < 10; i++) {
        auto &node = mock("N" + std::to_string(i));
        if (i < 9) node.mark_on_next("output");
    }

    auto builder = ir::testutil::Builder();
    for (int i = 0; i < 10; i++)
        builder.node("N" + std::to_string(i));

    for (int i = 0; i < 9; i++)
        builder.edge(
            "N" + std::to_string(i),
            "output",
            "N" + std::to_string(i + 1),
            "input"
        );

    std::vector<std::vector<std::string>> strata;
    for (int i = 0; i < 10; i++)
        strata.push_back({"N" + std::to_string(i)});

    auto ir = builder.strata(strata).build();
    auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    // All 10 nodes should execute
    for (int i = 0; i < 10; i++)
        ASSERT_EQ(mocks_["N" + std::to_string(i)]->next_called, 1);
}

/// @brief it should handle mixed continuous and one-shot edges correctly
TEST_F(SchedulerTest, testMixedContinuousAndOneShot) {
    auto &nodeA = mock("A");
    auto &nodeB = mock("B");
    const auto &nodeC = mock("C");

    nodeA.mark_on_next("output");
    nodeB.mark_on_next("output");
    nodeB.param_truthy["output"] = true;

    // A -> B (continuous), B => C (one-shot)
    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .node("C")
                  .edge("A", "output", "B", "input")
                  .oneshot("B", "output", "C", "input")
                  .strata({{"A"}, {"B"}, {"C"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 1);
    ASSERT_EQ(nodeC.next_called, 1);
}

/// @brief it should execute both global and staged nodes in the same execution
TEST_F(SchedulerTest, testGlobalAndStagedMixed) {
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    auto &globalNode = mock("G");
    const auto &stagedNode = mock("S");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();
    globalNode.mark_on_next("output");

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("G")
                  .node("S")
                  .oneshot("trigger", "activate", "entry_seq_stage", "input")
                  .edge("G", "output", "S", "input")
                  .strata({{"trigger", "G"}, {"entry_seq_stage"}})
                  .sequence("seq", {{"stage", {{"S"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(globalNode.next_called, 1);
    ASSERT_EQ(stagedNode.next_called, 1);
}

/// @brief it should support multiple sequences sharing a global node
TEST_F(SchedulerTest, testMultiSequenceWithSharedGlobal) {
    auto &trigger1 = mock("trigger1");
    auto &trigger2 = mock("trigger2");
    auto &entry1 = mock("entry_seq1_stage");
    auto &entry2 = mock("entry_seq2_stage");
    auto &globalNode = mock("G");
    const auto &staged1 = mock("S1");
    const auto &staged2 = mock("S2");

    trigger1.mark_on_next("activate");
    trigger1.param_truthy["activate"] = true;
    trigger2.mark_on_next("activate");
    trigger2.param_truthy["activate"] = true;
    entry1.activate_on_next();
    entry2.activate_on_next();
    globalNode.mark_on_next("output");

    auto ir = ir::testutil::Builder()
                  .node("trigger1")
                  .node("trigger2")
                  .node("entry_seq1_stage")
                  .node("entry_seq2_stage")
                  .node("G")
                  .node("S1")
                  .node("S2")
                  .oneshot("trigger1", "activate", "entry_seq1_stage", "input")
                  .oneshot("trigger2", "activate", "entry_seq2_stage", "input")
                  .edge("G", "output", "S1", "input")
                  .edge("G", "output", "S2", "input")
                  .strata(
                      {{"trigger1", "trigger2", "G"},
                       {"entry_seq1_stage", "entry_seq2_stage"}}
                  )
                  .sequence("seq1", {{"stage", {{"S1"}}}})
                  .sequence("seq2", {{"stage", {{"S2"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(globalNode.next_called, 1);
    ASSERT_EQ(staged1.next_called, 1);
    ASSERT_EQ(staged2.next_called, 1);
}

/// @brief it should handle zero elapsed time correctly
TEST_F(SchedulerTest, testZeroElapsedTime) {
    const auto &nodeA = mock("A");

    auto ir = ir::testutil::Builder().node("A").strata({{"A"}}).build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND * 0, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeA.elapsed_values[0], x::telem::MILLISECOND * 0);
}

/// @brief it should handle self-loop edges without infinite recursion
TEST_F(SchedulerTest, testSelfLoopHandled) {
    auto &nodeA = mock("A");

    // Self-loop: A -> A
    nodeA.mark_on_next("output");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .edge("A", "output", "A", "input")
                  .strata({{"A"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);
}

/// @brief it should handle empty sequences without crashing
TEST_F(SchedulerTest, testEmptySequence) {
    mock("A");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .strata({{"A"}})
                  .sequence("empty_seq", {}) // Empty sequence
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(mocks_["A"]->next_called, 1);
}

/// @brief it should reset all execution state including nodes
TEST_F(SchedulerTest, testResetClearsState) {
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_stage");
    auto &nodeA = mock("A");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage")
                  .node("A")
                  .oneshot("trigger", "activate", "entry_seq_stage", "input")
                  .strata({{"trigger"}, {"entry_seq_stage"}})
                  .sequence("seq", {{"stage", {{"A"}}}})
                  .build();

    const auto scheduler = build(std::move(ir));

    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeA.reset_called, 1);

    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 2);
    ASSERT_EQ(nodeA.reset_called, 1);

    scheduler->reset();

    ASSERT_EQ(nodeA.reset_called, 2);

    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 3);
    ASSERT_EQ(nodeA.reset_called, 3);
}

/// @brief it should reset fired one-shots after reset
TEST_F(SchedulerTest, testResetClearsFiredOneShots) {
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    nodeA.mark_on_next("output");
    nodeA.param_truthy["output"] = true;

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .oneshot("A", "output", "B", "input")
                  .strata({{"A"}, {"B"}})
                  .build();

    const auto scheduler = build(std::move(ir));

    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);

    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 1);

    scheduler->reset();

    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeB.next_called, 2);
}

/// @brief Helper to create IR with interval node that has proper params
ir::IR build_interval_ir(const std::string &key, const int64_t period_ns) {
    ir::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type(arc::types::Kind::U8);

    ir::Param cfg_param;
    cfg_param.name = "period";
    cfg_param.type = arc::types::Type(arc::types::Kind::I64);
    cfg_param.value = period_ns;

    ir::Node ir_node;
    ir_node.key = key;
    ir_node.type = "interval";
    ir_node.outputs.params.push_back(output_param);
    ir_node.config.params.push_back(cfg_param);

    ir::Function fn;
    fn.key = "test";

    ir::IR ir;
    ir.nodes.push_back(ir_node);
    ir.functions.push_back(fn);
    return ir;
}

/// @brief Merge multiple IRs together (for building complex test scenarios)
ir::IR merge_irs(const std::vector<ir::IR> &irs) {
    ir::IR merged;
    for (const auto &ir: irs) {
        for (const auto &node: ir.nodes)
            merged.nodes.push_back(node);
        for (const auto &fn: ir.functions)
            merged.functions.push_back(fn);
    }
    return merged;
}

/// @brief Test with real Interval node and one-shot edge
TEST(RealNodeSchedulerTest, IntervalOneShotEdgeFires) {
    // Build IR with interval node
    auto interval_ir = build_interval_ir("interval_0", x::telem::SECOND.nanoseconds());

    // Add a mock target node to the IR
    ir::Param target_input;
    target_input.name = "input";
    target_input.type = arc::types::Type(arc::types::Kind::U8);

    ir::Node target_node;
    target_node.key = "target_0";
    target_node.type = "target";
    target_node.inputs.params.push_back(target_input);
    interval_ir.nodes.push_back(target_node);

    // Add one-shot edge: interval => target
    interval_ir.edges.emplace_back(
        ir::Handle{"interval_0", "output"},
        ir::Handle{"target_0", "input"},
        ir::EdgeKind::OneShot
    );

    // Set strata
    interval_ir.strata = ir::Strata({{"interval_0"}, {"target_0"}});

    // Create state for interval node
    state::State state(
        state::Config{.ir = interval_ir, .channels = {}},
        errors::noop_handler
    );

    // Create real Interval node using Factory
    time::Factory factory;
    auto [interval_node, err] = factory.create(
        node::Config(
            interval_ir,
            interval_ir.nodes[0],
            ASSERT_NIL_P(state.node("interval_0"))
        )
    );
    ASSERT_NIL(err);
    ASSERT_NE(interval_node, nullptr);

    // Create mock target node
    auto target = std::make_unique<MockNode>();
    auto *target_ptr = target.get();

    // Build nodes map
    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes;
    nodes["interval_0"] = std::move(interval_node);
    nodes["target_0"] = std::move(target);

    auto scheduler = std::make_unique<Scheduler>(
        std::move(interval_ir),
        nodes,
        x::telem::TimeSpan(0)
    );

    // First tick at t=1s - Interval should fire (since it starts at -period)
    scheduler->next(x::telem::SECOND, node::RunReason::TimerTick);

    // Target should have been called exactly once (one-shot fired)
    EXPECT_EQ(target_ptr->next_called, 1);

    // Second tick at t=2s - Interval fires again but one-shot already fired
    scheduler->next(x::telem::SECOND * 2, node::RunReason::TimerTick);

    // Target should still be 1 (one-shot only fires once)
    EXPECT_EQ(target_ptr->next_called, 1);
}

/// @brief Test that Interval is_output_truthy is used by scheduler
TEST(RealNodeSchedulerTest, IntervalTruthyCheckBeforeFiring) {
    // Build IR with interval node
    auto interval_ir = build_interval_ir("interval_0", x::telem::SECOND.nanoseconds());

    // Add a mock target node to the IR
    ir::Param target_input;
    target_input.name = "input";
    target_input.type = arc::types::Type(arc::types::Kind::U8);

    ir::Node target_node;
    target_node.key = "target_0";
    target_node.type = "target";
    target_node.inputs.params.push_back(target_input);
    interval_ir.nodes.push_back(target_node);

    // Add one-shot edge
    interval_ir.edges.emplace_back(
        ir::Handle{"interval_0", "output"},
        ir::Handle{"target_0", "input"},
        ir::EdgeKind::OneShot
    );

    interval_ir.strata = ir::Strata({{"interval_0"}, {"target_0"}});

    state::State state(
        state::Config{.ir = interval_ir, .channels = {}},
        errors::noop_handler
    );

    time::Factory factory;
    auto [interval_node, err] = factory.create(
        node::Config(
            interval_ir,
            interval_ir.nodes[0],
            ASSERT_NIL_P(state.node("interval_0"))
        )
    );
    ASSERT_NIL(err);

    auto target = std::make_unique<MockNode>();
    auto *target_ptr = target.get();

    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes;
    nodes["interval_0"] = std::move(interval_node);
    nodes["target_0"] = std::move(target);

    auto scheduler = std::make_unique<Scheduler>(
        std::move(interval_ir),
        nodes,
        x::telem::TimeSpan(0)
    );

    // First tick at t=0 - interval hasn't fired yet
    scheduler->next(x::telem::TimeSpan(0), node::RunReason::TimerTick);
    // Tick at t=500ms (before first period completes after initial fire)
    scheduler->next(x::telem::MILLISECOND * 500, node::RunReason::TimerTick);
    // Target should have been called once from t=0 fire
    EXPECT_EQ(target_ptr->next_called, 1);
    // Tick at t=1s - Interval fires again but one-shot already fired
    scheduler->next(x::telem::SECOND, node::RunReason::TimerTick);
    EXPECT_EQ(target_ptr->next_called, 1);
}

}
