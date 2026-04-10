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
#include "arc/cpp/stl/time/time.h"

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
public:
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

/// @brief it should allow a self-changed node in a higher stratum to keep executing
TEST_F(SchedulerTest, testSelfChangedNodeKeepsExecuting) {
    auto &trigger = mock("trigger");
    auto &entry = mock("entry_seq_first");
    auto &comparison = mock("comparison");
    auto &wait = mock("wait");
    auto &entry_next = mock("entry_seq_next");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry.activate_on_next();
    entry_next.activate_on_next();

    comparison.on_next = [](node::Context &ctx) { ctx.mark_changed("output"); };
    comparison.param_truthy["output"] = true;

    bool wait_started = false;
    x::telem::TimeSpan wait_start_elapsed;
    wait.on_next = [&](node::Context &ctx) {
        if (ctx.reason != node::RunReason::TimerTick) return;
        if (!wait_started) {
            wait_started = true;
            wait_start_elapsed = ctx.elapsed;
            ctx.mark_self_changed();
            return;
        }
        if (ctx.elapsed - wait_start_elapsed < x::telem::SECOND) {
            ctx.mark_self_changed();
            return;
        }
        ctx.mark_changed("output");
    };
    wait.param_truthy["output"] = true;

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_first")
                  .node("comparison")
                  .node("wait")
                  .node("entry_seq_next")
                  .oneshot("trigger", "activate", "entry_seq_first", "input")
                  .oneshot("comparison", "output", "wait", "input")
                  .edge("wait", "output", "entry_seq_next", "input")
                  .strata({{"trigger"}, {"entry_seq_first"}})
                  .sequence(
                      "seq",
                      {{"first", {{"comparison"}, {"wait"}, {"entry_seq_next"}}},
                       {"next", {}}}
                  )
                  .build();

    const auto scheduler = build(std::move(ir));

    // Tick 0: trigger fires, stage activates, comparison fires one-shot to wait,
    // wait starts timing and calls mark_self_changed
    scheduler->next(x::telem::TimeSpan(0), node::RunReason::TimerTick);
    ASSERT_EQ(wait.next_called, 1);
    ASSERT_EQ(entry_next.next_called, 0);

    // Tick at 500ms: wait should execute (self-changed), but not fire yet
    scheduler->next(x::telem::MILLISECOND * 500, node::RunReason::TimerTick);
    ASSERT_EQ(wait.next_called, 2);
    ASSERT_EQ(entry_next.next_called, 0);

    // Tick at 1s: wait fires, propagates to entry_seq_next
    scheduler->next(x::telem::SECOND, node::RunReason::TimerTick);
    ASSERT_EQ(wait.next_called, 3);
    ASSERT_EQ(entry_next.next_called, 1);
}

/// @brief it should not execute a self-changed node after it stops calling
/// mark_self_changed. Node A is in stratum 1 (behind a one-shot) so it only executes
/// when in changed or self_changed. Once it stops calling mark_self_changed, it should
/// stop executing.
TEST_F(SchedulerTest, testSelfChangedDropsWhenNotRenewed) {
    auto &trigger = mock("trigger");
    auto &nodeA = mock("A");

    trigger.mark_on_next("output");
    trigger.param_truthy["output"] = true;

    int call_count = 0;
    nodeA.on_next = [&](node::Context &ctx) {
        call_count++;
        if (call_count <= 2) ctx.mark_self_changed();
    };

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("A")
                  .oneshot("trigger", "output", "A", "input")
                  .strata({{"trigger"}, {"A"}})
                  .build();

    const auto scheduler = build(std::move(ir));

    // Tick 0: trigger fires one-shot to A, A executes and self-changes
    scheduler->next(x::telem::TimeSpan(0), node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);

    // Tick 1: A executes via self-changed (one-shot already fired)
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 2);

    // Tick 2: A executes via self-changed (callCount=2, still calls mark_self_changed)
    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 3);

    // Tick 3: A should NOT execute (stopped calling mark_self_changed, one-shot
    // already fired, not in changed set)
    scheduler->next(x::telem::MILLISECOND * 3, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 3);
}

/// @brief it should clear self-changed when a node is reset via stage transition
TEST_F(SchedulerTest, testSelfChangedClearedOnStageTransition) {
    auto &trigger = mock("trigger");
    auto &entry_a = mock("entry_seq_stage_a");
    auto &entry_b = mock("entry_seq_stage_b");
    auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_a.activate_on_next();
    entry_b.activate_on_next();

    int a_call_count = 0;
    nodeA.on_next = [&](node::Context &ctx) {
        a_call_count++;
        if (a_call_count == 1) {
            ctx.mark_self_changed();
            ctx.mark_changed("to_b");
        }
    };
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

    // Tick 0: trigger → stage_a activates, A runs, self-changes + transitions to
    // stage_b
    scheduler->next(x::telem::TimeSpan(0), node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 1);

    // Tick 1: stage_b is active, A's self-changed should have been cleared by reset
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 2);
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

/// @brief it should clear self-changed entries after reset. Node A is behind a
/// one-shot edge and stays alive via mark_self_changed. After reset, A should not
/// execute because its self-changed entry was cleared and the one-shot source
/// (trigger) no longer fires the edge (trigger stops marking its output after the
/// first tick).
TEST_F(SchedulerTest, testResetClearsSelfChanged) {
    auto &trigger = mock("trigger");
    auto &nodeA = mock("A");

    int trigger_call_count = 0;
    trigger.on_next = [&](node::Context &ctx) {
        trigger_call_count++;
        if (trigger_call_count == 1) ctx.mark_changed("output");
    };
    trigger.param_truthy["output"] = true;

    nodeA.on_next = [&](node::Context &ctx) { ctx.mark_self_changed(); };

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("A")
                  .oneshot("trigger", "output", "A", "input")
                  .strata({{"trigger"}, {"A"}})
                  .build();

    const auto scheduler = build(std::move(ir));

    // Tick 0: trigger fires one-shot to A, A executes and self-changes
    scheduler->next(x::telem::TimeSpan(0), node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 1);

    // Tick 1: A executes via self-changed (one-shot already consumed)
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 2);

    scheduler->reset();

    // After reset, the one-shot is available again but trigger no longer fires it
    // (trigger_call_count > 1), and A's self-changed entry was cleared.
    // A should NOT execute.
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 2);

    // Confirm A stays dormant on subsequent ticks.
    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    ASSERT_EQ(nodeA.next_called, 2);
}

/// @brief Helper to create IR with interval node that has proper params
ir::IR build_interval_ir(const std::string &key, const int64_t period_ns) {
    types::Param output_param;
    output_param.name = "output";
    output_param.type = arc::types::Type{.kind = arc::types::Kind::U8};

    types::Param cfg_param;
    cfg_param.name = "period";
    cfg_param.type = arc::types::Type{.kind = arc::types::Kind::I64};
    cfg_param.value = period_ns;

    ir::Node ir_node;
    ir_node.key = key;
    ir_node.type = "interval";
    ir_node.outputs.push_back(output_param);
    ir_node.config.push_back(cfg_param);

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
    types::Param target_input;
    target_input.name = "input";
    target_input.type = arc::types::Type{.kind = arc::types::Kind::U8};

    ir::Node target_node;
    target_node.key = "target_0";
    target_node.type = "target";
    target_node.inputs.push_back(target_input);
    interval_ir.nodes.push_back(target_node);

    // Add one-shot edge: interval => target
    interval_ir.edges.emplace_back(
        ir::Handle{"interval_0", "output"},
        ir::Handle{"target_0", "input"},
        ir::EdgeKind::OneShot
    );

    // Set strata
    interval_ir.root.strata = ir::Strata({{"interval_0"}, {"target_0"}});

    // Create state for interval node
    state::State state(
        state::Config{.ir = interval_ir, .channels = {}},
        errors::noop_handler
    );

    // Create real Interval node using Factory
    stl::time::Module time_factory;
    auto [interval_node, err] = time_factory.create(
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
    types::Param target_input;
    target_input.name = "input";
    target_input.type = arc::types::Type{.kind = arc::types::Kind::U8};

    ir::Node target_node;
    target_node.key = "target_0";
    target_node.type = "target";
    target_node.inputs.push_back(target_input);
    interval_ir.nodes.push_back(target_node);

    // Add one-shot edge
    interval_ir.edges.emplace_back(
        ir::Handle{"interval_0", "output"},
        ir::Handle{"target_0", "input"},
        ir::EdgeKind::OneShot
    );

    interval_ir.root.strata = ir::Strata({{"interval_0"}, {"target_0"}});

    state::State state(
        state::Config{.ir = interval_ir, .channels = {}},
        errors::noop_handler
    );

    stl::time::Module time_factory;
    auto [interval_node, err] = time_factory.create(
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

/// @brief Helper to create IR with wait node that has proper params
ir::IR build_wait_ir(const std::string &key, const int64_t duration_ns) {
    types::Param output_param;
    output_param.name = "output";
    output_param.type.kind = types::Kind::U8;

    types::Param cfg_param;
    cfg_param.name = "duration";
    cfg_param.type.kind = types::Kind::I64;
    cfg_param.value = duration_ns;

    ir::Node ir_node;
    ir_node.key = key;
    ir_node.type = "wait";
    ir_node.outputs.push_back(output_param);
    ir_node.config.push_back(cfg_param);

    ir::Function fn;
    fn.key = "test";

    ir::IR ir;
    ir.nodes.push_back(ir_node);
    ir.functions.push_back(fn);
    return ir;
}

/// @brief Test with real Wait node behind a one-shot edge. Verifies the Wait
/// survives via mark_self_changed and eventually fires.
TEST(RealNodeSchedulerTest, WaitOneShotEdgeFiresAfterDuration) {
    auto wait_ir = build_wait_ir("wait_0", x::telem::SECOND.nanoseconds());

    types::Param target_input;
    target_input.name = "input";
    target_input.type.kind = types::Kind::U8;

    // Trigger node in stratum 0
    ir::Node trigger_node;
    trigger_node.key = "trigger_0";
    trigger_node.type = "trigger";
    wait_ir.nodes.push_back(trigger_node);

    // Target node in stratum 2
    ir::Node target_node;
    target_node.key = "target_0";
    target_node.type = "target";
    target_node.inputs.push_back(target_input);
    wait_ir.nodes.push_back(target_node);

    // Trigger => wait (one-shot), wait -> target (continuous)
    wait_ir.edges.emplace_back(
        ir::Handle{"trigger_0", "output"},
        ir::Handle{"wait_0", "input"},
        ir::EdgeKind::OneShot
    );
    wait_ir.edges.emplace_back(
        ir::Handle{"wait_0", "output"},
        ir::Handle{"target_0", "input"},
        ir::EdgeKind::Continuous
    );

    wait_ir.root.strata = ir::Strata({{"trigger_0"}, {"wait_0"}, {"target_0"}});

    state::State state(
        state::Config{.ir = wait_ir, .channels = {}},
        errors::noop_handler
    );

    stl::time::Module factory;
    auto [wait_node, err] = factory.create(
        node::Config(wait_ir, wait_ir.nodes[0], ASSERT_NIL_P(state.node("wait_0")))
    );
    ASSERT_NIL(err);

    // Trigger mock: always marks output as changed and truthy
    auto trigger = std::make_unique<MockNode>();
    trigger->on_next = [](node::Context &ctx) { ctx.mark_changed("output"); };
    trigger->param_truthy["output"] = true;

    auto target = std::make_unique<MockNode>();
    auto *target_ptr = target.get();

    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes;
    nodes["trigger_0"] = std::move(trigger);
    nodes["wait_0"] = std::move(wait_node);
    nodes["target_0"] = std::move(target);

    auto scheduler = std::make_unique<Scheduler>(
        std::move(wait_ir),
        nodes,
        x::telem::TimeSpan(0)
    );

    // Tick 0: trigger fires one-shot to wait, wait starts timing
    scheduler->next(x::telem::TimeSpan(0), node::RunReason::TimerTick);
    EXPECT_EQ(target_ptr->next_called, 0);

    // Tick 500ms: wait survives via self-changed, still timing
    scheduler->next(x::telem::MILLISECOND * 500, node::RunReason::TimerTick);
    EXPECT_EQ(target_ptr->next_called, 0);

    // Tick 1s: wait fires, propagates to target
    scheduler->next(x::telem::SECOND, node::RunReason::TimerTick);
    EXPECT_EQ(target_ptr->next_called, 1);

    // Tick 2s: wait already fired, target should not be called again
    scheduler->next(x::telem::SECOND * 2, node::RunReason::TimerTick);
    EXPECT_EQ(target_ptr->next_called, 1);
}

TEST_F(SchedulerTest, testNextDeadlineReturnsMaxWhenNoTimeNodes) {
    mock("A").mark_on_next("output");
    mock("B");

    auto ir = ir::testutil::Builder()
                  .node("A")
                  .node("B")
                  .edge("A", "output", "B", "input")
                  .strata({{"A"}, {"B"}})
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(scheduler->next_deadline(), x::telem::TimeSpan::max());
}

TEST_F(SchedulerTest, testNextDeadlineReturnsMinimumAcrossNodes) {
    auto &nodeA = mock("A");
    auto &nodeB = mock("B");
    nodeA.on_next = [](node::Context &ctx) {
        if (ctx.set_deadline) ctx.set_deadline(x::telem::SECOND * 3);
    };
    nodeB.on_next = [](node::Context &ctx) {
        if (ctx.set_deadline) ctx.set_deadline(x::telem::SECOND * 1);
    };

    auto ir = ir::testutil::Builder().node("A").node("B").strata({{"A", "B"}}).build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(scheduler->next_deadline(), x::telem::SECOND);
}

TEST_F(SchedulerTest, testNextDeadlineResetsEachCycle) {
    auto &nodeA = mock("A");
    int call = 0;
    nodeA.on_next = [&call](node::Context &ctx) {
        call++;
        if (call == 1 && ctx.set_deadline) ctx.set_deadline(x::telem::SECOND);
    };

    auto ir = ir::testutil::Builder().node("A").strata({{"A"}}).build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);
    EXPECT_EQ(scheduler->next_deadline(), x::telem::SECOND);

    scheduler->next(x::telem::MILLISECOND * 2, node::RunReason::TimerTick);
    EXPECT_EQ(scheduler->next_deadline(), x::telem::TimeSpan::max());
}

TEST_F(SchedulerTest, testNextDeadlineFromStageNode) {
    auto &nodeA = mock("A");
    nodeA.on_next = [](node::Context &ctx) {
        if (ctx.set_deadline) ctx.set_deadline(x::telem::SECOND * 2);
    };

    auto &trigger = mock("trigger");
    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    auto &entry = mock("entry_seq_stage");
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
    EXPECT_EQ(scheduler->next_deadline(), x::telem::SECOND * 2);
}

/// @brief it should stop evaluating statements after the first transition fires
TEST_F(SchedulerTest, testFirstStatementWinsWhenMultipleTransitionsAreTrue) {
    // Setup: trigger activates stage_on, which has two transition nodes (A and B)
    // in the same stratum. Both are truthy and wire to different stage entries.
    // A (first in stratum order) should win; B's transition should never fire.
    auto &trigger = mock("trigger");
    auto &entry_on = mock("entry_seq_stage_on");
    auto &nodeA = mock("A");
    auto &nodeB = mock("B");
    auto &entry_off = mock("entry_seq_stage_off");
    auto &entry_pause = mock("entry_seq_stage_pause");
    const auto &nodeOff = mock("Off");
    const auto &nodePause = mock("Pause");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_on.activate_on_next();
    entry_off.activate_on_next();
    entry_pause.activate_on_next();

    // Both transitions fire and are truthy
    nodeA.mark_on_next("check");
    nodeA.param_truthy["check"] = true;
    nodeB.mark_on_next("check");
    nodeB.param_truthy["check"] = true;

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage_on")
                  .node("A")
                  .node("B")
                  .node("entry_seq_stage_off")
                  .node("entry_seq_stage_pause")
                  .node("Off")
                  .node("Pause")
                  .oneshot("trigger", "activate", "entry_seq_stage_on", "input")
                  .oneshot("A", "check", "entry_seq_stage_off", "input")
                  .oneshot("B", "check", "entry_seq_stage_pause", "input")
                  .strata({{"trigger"}, {"entry_seq_stage_on"}})
                  .sequence(
                      "seq",
                      {{"stage_on",
                        {{"A", "B"}, {"entry_seq_stage_off", "entry_seq_stage_pause"}}},
                       {"stage_off", {{"Off"}}},
                       {"stage_pause", {{"Pause"}}}}
                  )
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    // A's transition to stage_off should win
    ASSERT_EQ(nodeOff.next_called, 1);
    // B's transition to stage_pause should never fire
    ASSERT_EQ(nodePause.next_called, 0);
}

/// @brief it should skip later write statements after a transition fires
TEST_F(SchedulerTest, testTransitionSkipsLaterWriteStatementInSameStage) {
    auto &trigger = mock("trigger");
    auto &entry_on = mock("entry_seq_stage_on");
    auto &transition = mock("to_abort");
    auto &write_cmd = mock("write_ox_tpc_cmd");
    auto &entry_abort = mock("entry_seq_stage_abort");
    const auto &abort_node = mock("abort_node");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_on.activate_on_next();
    entry_abort.activate_on_next();

    transition.mark_on_next("check");
    transition.param_truthy["check"] = true;

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_stage_on")
                  .node("to_abort")
                  .node("write_ox_tpc_cmd")
                  .node("entry_seq_stage_abort")
                  .node("abort_node")
                  .oneshot("trigger", "activate", "entry_seq_stage_on", "input")
                  .oneshot("to_abort", "check", "entry_seq_stage_abort", "input")
                  .strata({{"trigger"}, {"entry_seq_stage_on"}})
                  .sequence(
                      "seq",
                      {{"stage_on",
                        {{"to_abort"}, {"entry_seq_stage_abort", "write_ox_tpc_cmd"}}},
                       {"stage_abort", {{"abort_node"}}}}
                  )
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    // Transition should fire and move into abort stage.
    ASSERT_EQ(abort_node.next_called, 1);
    // Statement after transition in the same stage pass should be skipped.
    ASSERT_EQ(write_cmd.next_called, 0);
}

/// @brief it should respect source order when entry nodes are at the same stratum
TEST_F(SchedulerTest, testSourceOrderPriorityWhenEntriesAtSameStratum) {
    auto &trigger = mock("trigger");
    auto &entry_active = mock("entry_seq_active");
    auto &condA = mock("condA");
    auto &condB = mock("condB");
    auto &entryA = mock("entry_seq_stage_a");
    auto &entryB = mock("entry_seq_stage_b");
    const auto &nodeA = mock("A");
    const auto &nodeB = mock("B");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_active.activate_on_next();

    condA.mark_on_next("check");
    condA.param_truthy["check"] = true;
    condB.mark_on_next("check");
    condB.param_truthy["check"] = true;
    entryA.activate_on_next();
    entryB.activate_on_next();

    auto ir = ir::testutil::Builder()
                  .node("trigger")
                  .node("entry_seq_active")
                  .node("condA")
                  .node("condB")
                  .node("entry_seq_stage_a")
                  .node("entry_seq_stage_b")
                  .node("A")
                  .node("B")
                  .oneshot("trigger", "activate", "entry_seq_active", "input")
                  .oneshot("condA", "check", "entry_seq_stage_a", "input")
                  .oneshot("condB", "check", "entry_seq_stage_b", "input")
                  .strata({{"trigger"}, {"entry_seq_active"}})
                  .sequence(
                      "seq",
                      {{"active",
                        {{"condA", "condB"},
                         {"entry_seq_stage_a", "entry_seq_stage_b"}}},
                       {"stage_a", {{"A"}}},
                       {"stage_b", {{"B"}}}}
                  )
                  .build();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MILLISECOND, node::RunReason::TimerTick);

    ASSERT_EQ(nodeA.next_called, 1);
    ASSERT_EQ(nodeB.next_called, 0);
}

ir::IR build_flow_seq(
    SchedulerTest &t,
    const std::string &seq_key,
    const std::vector<std::string> &step_keys
) {
    ir::IR ir;
    ir::Sequence seq;
    seq.key = seq_key;
    ir::Stratum data_stratum;
    ir::Stratum entry_stratum;

    for (size_t i = 0; i < step_keys.size(); i++) {
        const auto &sk = step_keys[i];
        const auto node_key = "node_" + sk;
        const auto entry_key = "entry_" + seq_key + "_" + sk;
        t.mock(node_key);
        t.mock(entry_key);
        ir.nodes.push_back(ir::Node{.key = node_key});
        ir.nodes.push_back(ir::Node{.key = entry_key});
        ir::Step step;
        step.key = sk;
        step.flow = x::mem::indirect<ir::Flow>(ir::Flow{});
        step.flow->nodes.push_back(node_key);
        seq.steps.push_back(std::move(step));
        data_stratum.push_back(node_key);
        if (i > 0) entry_stratum.push_back(entry_key);
        if (i + 1 < step_keys.size()) {
            const auto next_entry = "entry_" + seq_key + "_" + step_keys[i + 1];
            ir.edges.push_back(ir::Edge{
                .source = {node_key, "output"},
                .target = {next_entry, "activate"},
                .kind = ir::EdgeKind::OneShot,
            });
        }
    }
    const auto trigger_key = "trigger_" + seq_key;
    const auto first_entry = "entry_" + seq_key + "_" + step_keys[0];
    t.mock(trigger_key);
    ir.nodes.push_back(ir::Node{.key = trigger_key});
    ir.edges.push_back(ir::Edge{
        .source = {trigger_key, "activate"},
        .target = {first_entry, "input"},
        .kind = ir::EdgeKind::OneShot,
    });
    ir.root.strata.push_back({trigger_key});
    ir.root.strata.push_back({first_entry});
    seq.strata.push_back(data_stratum);
    if (!entry_stratum.empty()) seq.strata.push_back(entry_stratum);
    ir.root.sequences.push_back(std::move(seq));
    return ir;
}

/// @brief it should execute a single flow step
TEST_F(SchedulerTest, testFlowStepSingleExecution) {
    auto ir = build_flow_seq(*this, "seq", {"s0"});
    mocks_["trigger_seq"]->mark_on_next("activate");
    mocks_["trigger_seq"]->param_truthy["activate"] = true;
    mocks_["entry_seq_s0"]->activate_on_next();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(mocks_["node_s0"]->next_called, 1);
    ASSERT_EQ(mocks_["node_s0"]->reset_called, 1);
}

/// @brief it should cascade two consecutive truthy writes on the same tick
TEST_F(SchedulerTest, testFlowStepCascadeTwoTruthy) {
    auto ir = build_flow_seq(*this, "seq", {"s0", "s1"});
    mocks_["trigger_seq"]->mark_on_next("activate");
    mocks_["trigger_seq"]->param_truthy["activate"] = true;
    mocks_["entry_seq_s0"]->activate_on_next();
    mocks_["node_s0"]->mark_on_next("output");
    mocks_["node_s0"]->param_truthy["output"] = true;
    mocks_["entry_seq_s1"]->activate_on_next();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(mocks_["node_s0"]->next_called, 1);
    ASSERT_EQ(mocks_["node_s1"]->next_called, 1);
}

/// @brief it should cascade three consecutive truthy writes on the same tick
TEST_F(SchedulerTest, testFlowStepCascadeThreeTruthy) {
    auto ir = build_flow_seq(*this, "seq", {"s0", "s1", "s2"});
    mocks_["trigger_seq"]->mark_on_next("activate");
    mocks_["trigger_seq"]->param_truthy["activate"] = true;
    mocks_["entry_seq_s0"]->activate_on_next();
    mocks_["node_s0"]->mark_on_next("output");
    mocks_["node_s0"]->param_truthy["output"] = true;
    mocks_["entry_seq_s1"]->activate_on_next();
    mocks_["node_s1"]->mark_on_next("output");
    mocks_["node_s1"]->param_truthy["output"] = true;
    mocks_["entry_seq_s2"]->activate_on_next();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(mocks_["node_s0"]->next_called, 1);
    ASSERT_EQ(mocks_["node_s1"]->next_called, 1);
    ASSERT_EQ(mocks_["node_s2"]->next_called, 1);
}

/// @brief it should cascade five consecutive truthy writes on the same tick
TEST_F(SchedulerTest, testFlowStepCascadeFiveTruthy) {
    auto ir = build_flow_seq(
        *this, "seq", {"s0", "s1", "s2", "s3", "s4"}
    );
    mocks_["trigger_seq"]->mark_on_next("activate");
    mocks_["trigger_seq"]->param_truthy["activate"] = true;
    for (const auto &sk : {"s0", "s1", "s2", "s3", "s4"}) {
        mocks_["entry_seq_" + std::string(sk)]->activate_on_next();
        mocks_["node_" + std::string(sk)]->mark_on_next("output");
        mocks_["node_" + std::string(sk)]->param_truthy["output"] = true;
    }

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    for (const auto &sk : {"s0", "s1", "s2", "s3", "s4"})
        ASSERT_EQ(mocks_["node_" + std::string(sk)]->next_called, 1);
}

/// @brief it should block at a falsy gate and advance when it becomes truthy
TEST_F(SchedulerTest, testFlowStepBlocksAtFalsyGate) {
    auto ir = build_flow_seq(*this, "seq", {"s0", "s1"});
    mocks_["trigger_seq"]->mark_on_next("activate");
    mocks_["trigger_seq"]->param_truthy["activate"] = true;
    mocks_["entry_seq_s0"]->activate_on_next();
    mocks_["node_s0"]->mark_on_next("output");
    mocks_["node_s0"]->param_truthy["output"] = false;
    mocks_["entry_seq_s1"]->activate_on_next();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(mocks_["node_s0"]->next_called, 1);
    ASSERT_EQ(mocks_["node_s1"]->next_called, 0);

    mocks_["node_s0"]->param_truthy["output"] = true;
    scheduler->mark_node_changed("node_s0");
    scheduler->next(2 * x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(mocks_["node_s1"]->next_called, 1);
}

/// @brief it should not execute nodes from inactive flow steps
TEST_F(SchedulerTest, testFlowStepInactiveNodesSkipped) {
    auto ir = build_flow_seq(*this, "seq", {"s0", "s1", "s2"});
    mocks_["trigger_seq"]->mark_on_next("activate");
    mocks_["trigger_seq"]->param_truthy["activate"] = true;
    mocks_["entry_seq_s0"]->activate_on_next();
    mocks_["node_s0"]->mark_on_next("output");
    mocks_["node_s0"]->param_truthy["output"] = false;
    mocks_["entry_seq_s1"]->activate_on_next();
    mocks_["entry_seq_s2"]->activate_on_next();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(mocks_["node_s0"]->next_called, 1);
    ASSERT_EQ(mocks_["node_s1"]->next_called, 0);
    ASSERT_EQ(mocks_["node_s2"]->next_called, 0);
}

/// @brief it should reset flow nodes when entering the step
TEST_F(SchedulerTest, testFlowStepResetOnEntry) {
    auto ir = build_flow_seq(*this, "seq", {"s0"});
    mocks_["trigger_seq"]->mark_on_next("activate");
    mocks_["trigger_seq"]->param_truthy["activate"] = true;
    mocks_["entry_seq_s0"]->activate_on_next();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(mocks_["node_s0"]->reset_called, 1);
}

/// @brief it should advance from a stage step to a flow step
TEST_F(SchedulerTest, testStageToFlowTransition) {
    auto &trigger = mock("trigger");
    auto &entry_stage = mock("entry_main_stage_a");
    auto &entry_flow = mock("entry_main_flow_b");
    auto &stage_node = mock("stage_node");
    auto &flow_node = mock("flow_node");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_stage.activate_on_next();
    stage_node.mark_on_next("output");
    stage_node.param_truthy["output"] = true;
    entry_flow.activate_on_next();

    ir::IR ir;
    ir.nodes.push_back({.key = "trigger"});
    ir.nodes.push_back({.key = "entry_main_stage_a"});
    ir.nodes.push_back({.key = "entry_main_flow_b"});
    ir.nodes.push_back({.key = "stage_node"});
    ir.nodes.push_back({.key = "flow_node"});
    ir.edges.push_back({
        .source = {"trigger", "activate"},
        .target = {"entry_main_stage_a", "input"},
        .kind = ir::EdgeKind::OneShot,
    });
    ir.edges.push_back({
        .source = {"stage_node", "output"},
        .target = {"entry_main_flow_b", "activate"},
        .kind = ir::EdgeKind::OneShot,
    });
    ir.root.strata.push_back({"trigger"});
    ir.root.strata.push_back({"entry_main_stage_a"});

    ir::Sequence seq;
    seq.key = "main";
    ir::Step stage_step;
    stage_step.key = "stage_a";
    stage_step.stage = x::mem::indirect<ir::Stage>(ir::Stage{});
    stage_step.stage->key = "stage_a";
    stage_step.stage->nodes = {"stage_node"};
    stage_step.stage->strata.push_back({"stage_node"});
    stage_step.stage->strata.push_back({"entry_main_flow_b"});
    seq.steps.push_back(std::move(stage_step));
    ir::Step flow_step;
    flow_step.key = "flow_b";
    flow_step.flow = x::mem::indirect<ir::Flow>(ir::Flow{});
    flow_step.flow->nodes = {"flow_node"};
    seq.steps.push_back(std::move(flow_step));
    seq.strata.push_back({"flow_node"});
    ir.root.sequences.push_back(std::move(seq));

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(stage_node.next_called, 1);
    ASSERT_EQ(flow_node.next_called, 1);
}

/// @brief it should advance from a flow step to a stage step
TEST_F(SchedulerTest, testFlowToStageTransition) {
    auto &trigger = mock("trigger");
    auto &entry_flow = mock("entry_main_flow_a");
    auto &entry_stage = mock("entry_main_stage_b");
    auto &flow_node = mock("flow_node");
    auto &stage_node = mock("stage_node");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_flow.activate_on_next();
    flow_node.mark_on_next("output");
    flow_node.param_truthy["output"] = true;
    entry_stage.activate_on_next();

    ir::IR ir;
    ir.nodes.push_back({.key = "trigger"});
    ir.nodes.push_back({.key = "entry_main_flow_a"});
    ir.nodes.push_back({.key = "entry_main_stage_b"});
    ir.nodes.push_back({.key = "flow_node"});
    ir.nodes.push_back({.key = "stage_node"});
    ir.edges.push_back({
        .source = {"trigger", "activate"},
        .target = {"entry_main_flow_a", "input"},
        .kind = ir::EdgeKind::OneShot,
    });
    ir.edges.push_back({
        .source = {"flow_node", "output"},
        .target = {"entry_main_stage_b", "activate"},
        .kind = ir::EdgeKind::OneShot,
    });
    ir.root.strata.push_back({"trigger"});
    ir.root.strata.push_back({"entry_main_flow_a"});

    ir::Sequence seq;
    seq.key = "main";
    ir::Step f_step;
    f_step.key = "flow_a";
    f_step.flow = x::mem::indirect<ir::Flow>(ir::Flow{});
    f_step.flow->nodes = {"flow_node"};
    seq.steps.push_back(std::move(f_step));
    ir::Step s_step;
    s_step.key = "stage_b";
    s_step.stage = x::mem::indirect<ir::Stage>(ir::Stage{});
    s_step.stage->key = "stage_b";
    s_step.stage->nodes = {"stage_node"};
    s_step.stage->strata.push_back({"stage_node"});
    seq.steps.push_back(std::move(s_step));
    seq.strata.push_back({"flow_node"});
    seq.strata.push_back({"entry_main_stage_b"});
    ir.root.sequences.push_back(std::move(seq));

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(flow_node.next_called, 1);
    ASSERT_EQ(stage_node.next_called, 1);
}

/// @brief it should support mixed stage, flow, stage pattern
TEST_F(SchedulerTest, testMixedStageFlowStagePattern) {
    auto &trigger = mock("trigger");
    auto &entry_press = mock("entry_main_press");
    auto &entry_write = mock("entry_main_write");
    auto &entry_vent = mock("entry_main_vent");
    auto &press_node = mock("press_node");
    auto &write_node = mock("write_node");
    auto &vent_node = mock("vent_node");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_press.activate_on_next();
    press_node.mark_on_next("output");
    press_node.param_truthy["output"] = true;
    entry_write.activate_on_next();
    write_node.mark_on_next("output");
    write_node.param_truthy["output"] = true;
    entry_vent.activate_on_next();

    ir::IR ir;
    for (const auto &k : {"trigger", "entry_main_press", "entry_main_write",
                           "entry_main_vent", "press_node", "write_node",
                           "vent_node"})
        ir.nodes.push_back({.key = k});
    ir.edges.push_back({
        .source = {"trigger", "activate"},
        .target = {"entry_main_press", "input"},
        .kind = ir::EdgeKind::OneShot,
    });
    ir.edges.push_back({
        .source = {"press_node", "output"},
        .target = {"entry_main_write", "activate"},
        .kind = ir::EdgeKind::OneShot,
    });
    ir.edges.push_back({
        .source = {"write_node", "output"},
        .target = {"entry_main_vent", "activate"},
        .kind = ir::EdgeKind::OneShot,
    });
    ir.root.strata.push_back({"trigger"});
    ir.root.strata.push_back({"entry_main_press"});

    ir::Sequence seq;
    seq.key = "main";
    ir::Step press;
    press.key = "press";
    press.stage = x::mem::indirect<ir::Stage>(ir::Stage{});
    press.stage->key = "press";
    press.stage->nodes = {"press_node"};
    press.stage->strata.push_back({"press_node"});
    press.stage->strata.push_back({"entry_main_write"});
    seq.steps.push_back(std::move(press));
    ir::Step write;
    write.key = "write";
    write.flow = x::mem::indirect<ir::Flow>(ir::Flow{});
    write.flow->nodes = {"write_node"};
    seq.steps.push_back(std::move(write));
    ir::Step vent;
    vent.key = "vent";
    vent.stage = x::mem::indirect<ir::Stage>(ir::Stage{});
    vent.stage->key = "vent";
    vent.stage->nodes = {"vent_node"};
    vent.stage->strata.push_back({"vent_node"});
    seq.steps.push_back(std::move(vent));
    seq.strata.push_back({"write_node"});
    seq.strata.push_back({"entry_main_vent"});
    ir.root.sequences.push_back(std::move(seq));

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(press_node.next_called, 1);
    ASSERT_EQ(write_node.next_called, 1);
    ASSERT_EQ(vent_node.next_called, 1);
}

/// @brief it should report deadline from a wait node in a flow step
TEST_F(SchedulerTest, testFlowStepDeadline) {
    auto ir = build_flow_seq(*this, "seq", {"s0"});
    mocks_["trigger_seq"]->mark_on_next("activate");
    mocks_["trigger_seq"]->param_truthy["activate"] = true;
    mocks_["entry_seq_s0"]->activate_on_next();
    mocks_["node_s0"]->on_next = [](node::Context &ctx) {
        ctx.set_deadline(5 * x::telem::SECOND);
        ctx.mark_self_changed();
    };

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(scheduler->next_deadline(), 5 * x::telem::SECOND);
}

/// @brief it should handle a nested sequence inside a stage step
TEST_F(SchedulerTest, testNestedSequenceInStage) {
    auto &trigger = mock("trigger");
    auto &entry_outer = mock("entry_outer_stage_a");
    auto &stage_node = mock("stage_node");
    auto &inner_node = mock("inner_node");

    trigger.mark_on_next("activate");
    trigger.param_truthy["activate"] = true;
    entry_outer.activate_on_next();

    ir::IR ir;
    ir.nodes.push_back({.key = "trigger"});
    ir.nodes.push_back({.key = "entry_outer_stage_a"});
    ir.nodes.push_back({.key = "stage_node"});
    ir.nodes.push_back({.key = "inner_node"});
    ir.edges.push_back({
        .source = {"trigger", "activate"},
        .target = {"entry_outer_stage_a", "input"},
        .kind = ir::EdgeKind::OneShot,
    });
    ir.root.strata.push_back({"trigger"});
    ir.root.strata.push_back({"entry_outer_stage_a"});

    ir::Sequence inner_seq;
    inner_seq.key = "inner";
    ir::Step inner_flow;
    inner_flow.key = "s0";
    inner_flow.flow = x::mem::indirect<ir::Flow>(ir::Flow{});
    inner_flow.flow->nodes = {"inner_node"};
    inner_seq.steps.push_back(std::move(inner_flow));
    inner_seq.strata.push_back({"inner_node"});

    ir::Stage stage;
    stage.key = "stage_a";
    stage.nodes = {"stage_node"};
    stage.strata.push_back({"stage_node"});
    stage.strata.push_back({"boundary_inner"});
    stage.sequences.push_back(std::move(inner_seq));

    ir::Sequence outer_seq;
    outer_seq.key = "outer";
    ir::Step stage_step;
    stage_step.key = "stage_a";
    stage_step.stage = x::mem::indirect<ir::Stage>(std::move(stage));
    outer_seq.steps.push_back(std::move(stage_step));
    ir.root.sequences.push_back(std::move(outer_seq));

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(stage_node.next_called, 1);
    ASSERT_EQ(inner_node.next_called, 1);
}

/// @brief it should cascade through three consecutive flow steps in one tick
TEST_F(SchedulerTest, testThreeStepFlowCascade) {
    auto ir = build_flow_seq(*this, "seq", {"s0", "s1", "s2"});

    mocks_["trigger_seq"]->mark_on_next("activate");
    mocks_["trigger_seq"]->param_truthy["activate"] = true;
    mocks_["entry_seq_s0"]->activate_on_next();

    mocks_["node_s0"]->mark_on_next("output");
    mocks_["node_s0"]->param_truthy["output"] = true;
    mocks_["entry_seq_s1"]->activate_on_next();

    mocks_["node_s1"]->mark_on_next("output");
    mocks_["node_s1"]->param_truthy["output"] = true;
    mocks_["entry_seq_s2"]->activate_on_next();

    const auto scheduler = build(std::move(ir));
    scheduler->next(x::telem::MICROSECOND, node::RunReason::TimerTick);

    ASSERT_EQ(mocks_["node_s0"]->next_called, 1);
    ASSERT_EQ(mocks_["node_s1"]->next_called, 1);
    ASSERT_EQ(mocks_["node_s2"]->next_called, 1);
}

}
