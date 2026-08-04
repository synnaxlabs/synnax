// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/arc/authority.h"
#include "driver/common/status.h"
#include "driver/control/state.h"
#include "driver/task/task.h"

namespace driver::arc::authority { namespace {

const x::control::Subject SELF{.key = "1", .name = "arc"};
const x::control::Subject OPERATOR{.key = "op-1", .name = "operator"};
const x::control::Subject OTHER_ARC{.key = "2", .name = "arc-high"};

synnax::channel::Channel make_channel(synnax::channel::Key key, std::string name) {
    synnax::channel::Channel ch;
    ch.key = key;
    ch.name = std::move(name);
    return ch;
}

void acquire(
    driver::control::States &states,
    const x::control::Subject &subject,
    synnax::channel::Key key,
    x::control::Authority authority
) {
    states.apply(
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = subject,
                    .resource = key,
                    .authority = authority,
                },
            }},
        }
    );
}

TEST(AuthorityTest, NoConflictWhenUncontrolled) {
    driver::control::States states;
    const std::vector writes{make_channel(1, "press_vlv_cmd")};
    ASSERT_TRUE(evaluate_conflicts(states, writes, SELF).empty());
}

TEST(AuthorityTest, NoConflictWhenSelfHolds) {
    driver::control::States states;
    acquire(states, SELF, 1, 200);
    const std::vector writes{make_channel(1, "press_vlv_cmd")};
    ASSERT_TRUE(evaluate_conflicts(states, writes, SELF).empty());
}

TEST(AuthorityTest, ConflictWhenOtherHolds) {
    driver::control::States states;
    acquire(states, OPERATOR, 1, 255);
    const std::vector writes{make_channel(1, "press_vlv_cmd")};
    const auto conflicts = evaluate_conflicts(states, writes, SELF);
    ASSERT_EQ(conflicts.size(), 1);
    ASSERT_EQ(conflicts[0].holder.key, OPERATOR.key);
    const auto [message, description] = build_warning(conflicts);
    ASSERT_EQ(message, "Authority held on 1 channel by another writer");
    ASSERT_EQ(description, "operator: press_vlv_cmd");
}

TEST(AuthorityTest, ConflictIgnoresAuthorityLevel) {
    driver::control::States states;
    acquire(states, OPERATOR, 1, 1);
    const std::vector writes{make_channel(1, "press_vlv_cmd")};
    ASSERT_EQ(evaluate_conflicts(states, writes, SELF).size(), 1);
}

TEST(AuthorityTest, MultipleChannelsSingleHolder) {
    driver::control::States states;
    acquire(states, OPERATOR, 1, 255);
    acquire(states, OPERATOR, 2, 255);
    const std::vector writes{
        make_channel(1, "press_vlv_cmd"),
        make_channel(2, "vent_vlv_cmd"),
    };
    const auto conflicts = evaluate_conflicts(states, writes, SELF);
    ASSERT_EQ(conflicts.size(), 2);
    const auto [message, description] = build_warning(conflicts);
    ASSERT_EQ(message, "Authority held on 2 channels by another writer");
    ASSERT_EQ(description, "operator: press_vlv_cmd, vent_vlv_cmd");
}

TEST(AuthorityTest, MultipleHoldersUseOtherWriters) {
    driver::control::States states;
    acquire(states, OPERATOR, 1, 255);
    acquire(states, OTHER_ARC, 2, 255);
    const std::vector writes{
        make_channel(1, "press_vlv_cmd"),
        make_channel(2, "vent_vlv_cmd"),
    };
    const auto conflicts = evaluate_conflicts(states, writes, SELF);
    ASSERT_EQ(conflicts.size(), 2);
    ASSERT_EQ(
        build_warning(conflicts).first,
        "Authority held on 2 channels by other writers"
    );
}

TEST(AuthorityTest, ConflictsSortedByChannelKey) {
    driver::control::States states;
    acquire(states, OPERATOR, 2, 255);
    acquire(states, OPERATOR, 1, 255);
    const std::vector writes{
        make_channel(2, "vent_vlv_cmd"),
        make_channel(1, "press_vlv_cmd"),
    };
    const auto conflicts = evaluate_conflicts(states, writes, SELF);
    ASSERT_EQ(conflicts.size(), 2);
    ASSERT_EQ(conflicts[0].channel.key, 1);
    ASSERT_EQ(conflicts[1].channel.key, 2);
}

TEST(AuthorityTest, OneChannelOneWriter) {
    driver::control::States states;
    acquire(states, OPERATOR, 1, 255);
    const std::vector writes{make_channel(1, "press_vlv_cmd")};
    const auto conflicts = evaluate_conflicts(states, writes, SELF);
    ASSERT_EQ(conflicts.size(), 1);
    const auto [message, description] = build_warning(conflicts);
    ASSERT_EQ(message, "Authority held on 1 channel by another writer");
    ASSERT_EQ(description, "operator: press_vlv_cmd");
}

TEST(AuthorityTest, ManyChannelsOneWriter) {
    driver::control::States states;
    acquire(states, OPERATOR, 1, 255);
    acquire(states, OPERATOR, 2, 255);
    acquire(states, OPERATOR, 3, 255);
    const std::vector writes{
        make_channel(1, "press_vlv_cmd"),
        make_channel(2, "vent_vlv_cmd"),
        make_channel(3, "fill_vlv_cmd"),
    };
    const auto conflicts = evaluate_conflicts(states, writes, SELF);
    ASSERT_EQ(conflicts.size(), 3);
    const auto [message, description] = build_warning(conflicts);
    ASSERT_EQ(message, "Authority held on 3 channels by another writer");
    ASSERT_EQ(description, "operator: press_vlv_cmd, vent_vlv_cmd, fill_vlv_cmd");
}

TEST(AuthorityTest, ManyChannelsManyWritersGroupedByHolder) {
    driver::control::States states;
    acquire(states, OPERATOR, 1, 255);
    acquire(states, OTHER_ARC, 2, 255);
    acquire(states, OPERATOR, 3, 255);
    const std::vector writes{
        make_channel(1, "press_vlv_cmd"),
        make_channel(2, "vent_vlv_cmd"),
        make_channel(3, "fill_vlv_cmd"),
    };
    const auto conflicts = evaluate_conflicts(states, writes, SELF);
    ASSERT_EQ(conflicts.size(), 3);
    const auto [message, description] = build_warning(conflicts);
    ASSERT_EQ(message, "Authority held on 3 channels by other writers");
    ASSERT_EQ(
        description,
        "operator: press_vlv_cmd, fill_vlv_cmd\narc-high: vent_vlv_cmd"
    );
}

TEST(AuthorityTest, MixedSelfAndOtherCountsOnlyOther) {
    driver::control::States states;
    acquire(states, SELF, 1, 200);
    acquire(states, OPERATOR, 2, 255);
    const std::vector writes{
        make_channel(1, "press_vlv_cmd"),
        make_channel(2, "vent_vlv_cmd"),
    };
    const auto conflicts = evaluate_conflicts(states, writes, SELF);
    ASSERT_EQ(conflicts.size(), 1);
    ASSERT_EQ(conflicts[0].channel.key, 2);
    ASSERT_EQ(
        build_warning(conflicts).first,
        "Authority held on 1 channel by another writer"
    );
}

TEST(AuthorityTest, NoConflictWhenAllSelfHeld) {
    driver::control::States states;
    acquire(states, SELF, 1, 200);
    acquire(states, SELF, 2, 200);
    const std::vector writes{
        make_channel(1, "press_vlv_cmd"),
        make_channel(2, "vent_vlv_cmd")
    };
    ASSERT_TRUE(evaluate_conflicts(states, writes, SELF).empty());
}

TEST(AuthorityTest, NoConflictForUncontrolledChannel) {
    driver::control::States states;
    acquire(states, OPERATOR, 99, 255);
    const std::vector writes{make_channel(1, "press_vlv_cmd")};
    ASSERT_TRUE(evaluate_conflicts(states, writes, SELF).empty());
}

TEST(AuthorityTest, NoConflictWithNoWriteChannels) {
    driver::control::States states;
    acquire(states, OPERATOR, 1, 255);
    const std::vector<synnax::channel::Channel> writes;
    ASSERT_TRUE(evaluate_conflicts(states, writes, SELF).empty());
}

TEST(AuthorityTest, SelfMatchedByKeyNotName) {
    driver::control::States states;
    const x::control::Subject self_renamed{.key = "1", .name = "arc-renamed"};
    acquire(states, self_renamed, 1, 200);
    const std::vector writes{make_channel(1, "press_vlv_cmd")};
    ASSERT_TRUE(evaluate_conflicts(states, writes, SELF).empty());
}

void release(driver::control::States &states, synnax::channel::Key key) {
    states.apply(
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from =
                    x::control::State<synnax::channel::Key>{
                        .subject = OPERATOR,
                        .resource = key,
                        .authority = 255,
                    },
                .to = std::nullopt,
            }},
        }
    );
}

common::StatusHandler make_status(std::shared_ptr<driver::task::MockContext> &ctx) {
    synnax::task::Task meta;
    meta.key = 1;
    meta.name = "arc";
    return common::StatusHandler(ctx, meta);
}

TEST(WarnerTest, InertWithoutStates) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    auto state = make_status(ctx);
    const Warner warner(nullptr, {make_channel(1, "press_vlv_cmd")}, SELF);
    warner.report(state);
    ASSERT_TRUE(ctx->statuses.empty());
}

TEST(WarnerTest, WarnsThenClears) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    auto state = make_status(ctx);
    auto states = std::make_shared<driver::control::States>();
    const Warner warner(states, {make_channel(1, "press_vlv_cmd")}, SELF);

    warner.report(state);
    ASSERT_TRUE(ctx->statuses.empty());

    acquire(*states, OPERATOR, 1, 255);
    warner.report(state);
    ASSERT_FALSE(ctx->statuses.empty());
    ASSERT_EQ(ctx->statuses.back().variant, synnax::status::VARIANT_WARNING);
    ASSERT_EQ(
        ctx->statuses.back().message,
        "Authority held on 1 channel by another writer"
    );

    release(*states, 1);
    warner.report(state);
    ASSERT_EQ(ctx->statuses.back().variant, synnax::status::VARIANT_SUCCESS);
}

}}
