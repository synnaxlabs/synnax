// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/scheduler/testutil/testutil.h"

namespace arc::runtime::scheduler {
using namespace testutil;

/// @brief builds main, a triggered sequence of the stages first and second. The node
/// of each stage runs every cycle, and its truthy output leaves the stage. second
/// leaves through an exit.
class ScopeTest : public SchedulerTest {
protected:
    std::vector<std::vector<Path>> reports;
    MockNode *first = nullptr;
    MockNode *second = nullptr;
    std::unique_ptr<Scheduler> sched;

    void SetUp() override {
        this->mock("trigger", {true});
        this->first = &this->mock("first_node");
        this->second = &this->mock("second_node");
        for (auto *m: {this->first, this->second})
            m->on_next = [](node::Context &ctx) { ctx.mark_self_changed(); };
        ir::Transition to_second;
        to_second.on = ir::Handle{"first_node", "output"};
        to_second.target = step_key_target("second");
        ir::Transition to_exit;
        to_exit.on = ir::Handle{"second_node", "output"};
        to_exit.target = exit_target();
        auto main = sequential_scope(
            "main",
            {ir::scope_member(
                 parallel_scope("first", {stratum_of({ir::node_member("first_node")})})
             ),
             ir::scope_member(parallel_scope(
                 "second",
                 {stratum_of({ir::node_member("second_node")})}
             ))},
            {to_second, to_exit}
        );
        main.activations = {
            {.on = ir::Handle{"trigger", "output"}, .kind = ir::EdgeKind::Conditional}
        };
        this->sched = std::make_unique<Scheduler>(
            program_of(
                {ir_node("trigger", {"output"}),
                 ir_node("first_node", {"output"}),
                 ir_node("second_node", {"output"})},
                {},
                root_scope(
                    {ir::node_member("trigger"), ir::scope_member(std::move(main))}
                )
            ),
            this->nodes,
            noop_handler,
            [this](const std::vector<Path> &paths) { this->reports.push_back(paths); }
        );
    }

    void tick() {
        this->sched->next(
            {.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick}
        );
    }
};

/// @brief it should report each start, stage change, and end of a sequence.
TEST_F(ScopeTest, ReportsEachStartStageChangeAndEnd) {
    this->tick();
    this->first->set_truthy(0);
    this->tick();
    this->second->set_truthy(0);
    this->tick();
    EXPECT_EQ(
        this->reports,
        (std::vector<std::vector<Path>>{
            {{"main", "first"}},
            {{"main", "second"}},
            {},
        })
    );
    EXPECT_TRUE(this->sched->active_paths().empty());
}

/// @brief it should not report a cycle that changes no path.
TEST_F(ScopeTest, DoesNotReportACycleThatChangesNoPath) {
    for (int i = 0; i < 5; i++)
        this->tick();
    EXPECT_EQ(this->reports.size(), 1);
    EXPECT_EQ(this->sched->active_paths(), (std::vector<Path>{{"main", "first"}}));
}

/// @brief it should report the paths again on the first cycle after a reset.
TEST_F(ScopeTest, ReportsAgainAfterAReset) {
    this->tick();
    (*this->sched).reset();
    this->tick();
    EXPECT_EQ(this->reports.size(), 2);
    EXPECT_EQ(this->reports.back(), (std::vector<Path>{{"main", "first"}}));
}

/// @brief it should not report a program with no top-level scope.
TEST_F(SchedulerTest, DoesNotReportAProgramWithNoTopLevelScope) {
    mock("A");
    bool called = false;
    const auto s = std::make_unique<Scheduler>(
        program_of({ir_node("A")}, {}, root_scope({ir::node_member("A")})),
        this->nodes,
        noop_handler,
        [&called](const std::vector<Path> &) { called = true; }
    );
    s->next({.elapsed = x::telem::MILLISECOND, .reason = node::RunReason::TimerTick});
    EXPECT_FALSE(called);
}
}
