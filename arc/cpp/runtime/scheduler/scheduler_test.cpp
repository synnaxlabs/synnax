// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/scheduler/scheduler.h"

#include "gtest/gtest.h"
#include "x/cpp/xtest/xtest.h"

// Mock node for testing
class MockNode : public arc::Node {
    std::string id_;
    int *exec_count_;  // Track execution count

public:
    MockNode(std::string id, int *exec_count)
        : id_(std::move(id)), exec_count_(exec_count) {}

    xerrors::Error execute(arc::NodeContext &ctx) override {
        (void)ctx;  // Unused in mock
        (*exec_count_)++;
        return xerrors::NIL;
    }

    std::string id() const override { return id_; }
};

// Mock node that can fail
class FailingNode : public arc::Node {
    std::string id_;

public:
    explicit FailingNode(std::string id) : id_(std::move(id)) {}

    xerrors::Error execute(arc::NodeContext &ctx) override {
        (void)ctx;  // Unused in mock
        return xerrors::Error("arc.test.node_failed");
    }

    std::string id() const override { return id_; }
};

TEST(SchedulerTest, RegisterNode) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    int exec_count = 0;
    auto node = std::make_unique<MockNode>("node1", &exec_count);

    auto err = scheduler.register_node("node1", std::move(node), 0);
    ASSERT_NIL(err);

    EXPECT_TRUE(scheduler.has_node("node1"));
    EXPECT_EQ(scheduler.get_stratum("node1"), 0);
    EXPECT_EQ(scheduler.num_nodes(), 1);
    EXPECT_EQ(scheduler.num_strata(), 1);
}

TEST(SchedulerTest, DuplicateNodeRegistration) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    int exec_count1 = 0, exec_count2 = 0;

    scheduler.register_node("node1", std::make_unique<MockNode>("node1", &exec_count1),
                           0);

    // Try to register same ID again
    auto err = scheduler.register_node("node1",
                                      std::make_unique<MockNode>("node1", &exec_count2),
                                      0);
    EXPECT_TRUE(err.matches(xerrors::Error("arc.scheduler.duplicate_node")));
}

TEST(SchedulerTest, Stratum0AlwaysExecutes) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    int exec_count = 0;
    scheduler.register_node("source", std::make_unique<MockNode>("source", &exec_count),
                           0);

    // Execute multiple cycles
    scheduler.next();
    EXPECT_EQ(exec_count, 1);

    scheduler.next();
    EXPECT_EQ(exec_count, 2);

    scheduler.next();
    EXPECT_EQ(exec_count, 3);
}

TEST(SchedulerTest, HigherStrataOnlyExecuteIfChanged) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    int stratum0_count = 0, stratum1_count = 0;

    scheduler.register_node("source",
                           std::make_unique<MockNode>("source", &stratum0_count), 0);
    scheduler.register_node("transform",
                           std::make_unique<MockNode>("transform", &stratum1_count), 1);

    // First cycle - stratum 0 executes, stratum 1 does NOT (not marked changed)
    scheduler.next();
    EXPECT_EQ(stratum0_count, 1);
    EXPECT_EQ(stratum1_count, 0);

    // Mark transform as changed
    scheduler.mark_changed("transform");

    // Second cycle - both execute
    scheduler.next();
    EXPECT_EQ(stratum0_count, 2);
    EXPECT_EQ(stratum1_count, 1);

    // Third cycle - stratum 1 NOT executed (changed cleared after cycle)
    scheduler.next();
    EXPECT_EQ(stratum0_count, 3);
    EXPECT_EQ(stratum1_count, 1);  // Still 1
}

TEST(SchedulerTest, MultipleStrata) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    int s0_count = 0, s1_count = 0, s2_count = 0;

    scheduler.register_node("s0_node", std::make_unique<MockNode>("s0_node", &s0_count),
                           0);
    scheduler.register_node("s1_node", std::make_unique<MockNode>("s1_node", &s1_count),
                           1);
    scheduler.register_node("s2_node", std::make_unique<MockNode>("s2_node", &s2_count),
                           2);

    EXPECT_EQ(scheduler.num_strata(), 3);

    // Mark higher strata as changed
    scheduler.mark_changed("s1_node");
    scheduler.mark_changed("s2_node");

    scheduler.next();

    EXPECT_EQ(s0_count, 1);
    EXPECT_EQ(s1_count, 1);
    EXPECT_EQ(s2_count, 1);

    // Next cycle - only stratum 0
    scheduler.next();
    EXPECT_EQ(s0_count, 2);
    EXPECT_EQ(s1_count, 1);  // Not changed
    EXPECT_EQ(s2_count, 1);  // Not changed
}

TEST(SchedulerTest, MarkDownstreamChanged) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    int s0_count = 0, s1_count = 0, s2_count = 0;

    scheduler.register_node("source", std::make_unique<MockNode>("source", &s0_count), 0);
    scheduler.register_node("middle", std::make_unique<MockNode>("middle", &s1_count), 1);
    scheduler.register_node("sink", std::make_unique<MockNode>("sink", &s2_count), 2);

    // Mark source's downstream as changed (should trigger middle and sink)
    scheduler.mark_downstream_changed("source");

    scheduler.next();

    EXPECT_EQ(s0_count, 1);  // Stratum 0 always executes
    EXPECT_EQ(s1_count, 1);  // Marked via downstream
    EXPECT_EQ(s2_count, 1);  // Marked via downstream
}

TEST(SchedulerTest, NodeExecutionFailure) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    scheduler.register_node("failing", std::make_unique<FailingNode>("failing"), 0);

    auto err = scheduler.next();
    EXPECT_TRUE(err.matches(xerrors::Error("arc.test.node_failed")));
}

TEST(SchedulerTest, MultipleNodesInSameStratum) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    int count1 = 0, count2 = 0, count3 = 0;

    // Register multiple nodes in stratum 0
    scheduler.register_node("node1", std::make_unique<MockNode>("node1", &count1), 0);
    scheduler.register_node("node2", std::make_unique<MockNode>("node2", &count2), 0);
    scheduler.register_node("node3", std::make_unique<MockNode>("node3", &count3), 0);

    scheduler.next();

    // All should execute
    EXPECT_EQ(count1, 1);
    EXPECT_EQ(count2, 1);
    EXPECT_EQ(count3, 1);
}

TEST(SchedulerTest, ChangedSetClearedAfterCycle) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    int count = 0;
    scheduler.register_node("node", std::make_unique<MockNode>("node", &count), 1);

    // Mark changed
    scheduler.mark_changed("node");

    // First cycle - executes
    scheduler.next();
    EXPECT_EQ(count, 1);

    // Second cycle - does NOT execute (changed was cleared)
    scheduler.next();
    EXPECT_EQ(count, 1);  // Still 1
}

TEST(SchedulerTest, GetStratumForNonexistentNode) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    EXPECT_EQ(scheduler.get_stratum("nonexistent"), 0);
    EXPECT_FALSE(scheduler.has_node("nonexistent"));
}

TEST(SchedulerTest, ProcessesInputQueueBeforeExecution) {
    queue::SPSC<arc::ChannelUpdate> input_queue(16);
    queue::SPSC<arc::ChannelOutput> output_queue(16);
    arc::State state(&input_queue, &output_queue);
    arc::Scheduler scheduler(&state);

    // Register channel
    state.register_channel(1, telem::FLOAT32_T);

    // Add data to input queue
    auto data = std::make_shared<telem::Series>(std::vector<float>{42.0f});
    auto time = std::make_shared<telem::Series>(
        telem::Series::linspace(telem::TimeStamp(0), telem::TimeStamp(1000000000), 1)
    );
    input_queue.push({1, data, time});

    // Before next(), data not yet processed
    auto [value1, err1] = state.read_channel(1);
    EXPECT_TRUE(err1.matches(xerrors::Error("arc.state.no_data")));

    // Register a node so scheduler has something to do
    int count = 0;
    scheduler.register_node("node", std::make_unique<MockNode>("node", &count), 0);

    // Execute scheduler (should process input queue)
    scheduler.next();

    // After next(), data should be available
    auto [value2, err2] = state.read_channel(1);
    ASSERT_NIL(err2);
    EXPECT_EQ(std::get<float>(value2), 42.0f);
}
