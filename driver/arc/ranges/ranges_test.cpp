// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <filesystem>
#include <memory>
#include <mutex>
#include <string>
#include <utility>
#include <variant>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/pipe.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/stl/testutil/testutil.h"
#include "driver/arc/ranges/ranges.h"

namespace driver::arc::ranges {
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

::arc::types::Param str_input(const std::string &name) {
    ::arc::types::Param p;
    p.name = name;
    p.type = ::arc::types::Type{.kind = ::arc::types::Kind::String};
    return p;
}

::arc::stl::testutil::NodeSpec
spec_of(const std::string &type, const std::vector<std::string> &inputs) {
    ::arc::types::Param out;
    out.name = ::arc::ir::default_output_param;
    out.type = ::arc::types::Type{.kind = ::arc::types::Kind::String};
    ::arc::stl::testutil::NodeSpec spec;
    spec.type = type;
    spec.outputs = ::arc::types::Params{out};
    for (const auto &name: inputs)
        spec.inputs.push_back(str_input(name));
    return spec;
}

::arc::stl::testutil::NodeSpec create_spec() {
    return spec_of("create", {"name", "parent", "color"});
}

::arc::stl::testutil::NodeSpec end_spec() {
    return spec_of("end", {"key"});
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

std::string unique_name(const std::string &prefix) {
    return prefix + std::to_string(
                        static_cast<unsigned>(x::telem::TimeStamp::now().nanoseconds())
                    );
}

std::string output_of(const ::arc::stl::testutil::Fixture &f) {
    return std::get<std::string>((*f.make_node().output(0)).at(0));
}
}

class RangesTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::filesystem::path dir;
    std::shared_ptr<::driver::outbox::Outbox> outbox;

    void SetUp() override {
        this->client = std::make_shared<synnax::Synnax>(new_test_client());
        this->dir = std::filesystem::temp_directory_path() /
                    ("ranges_test_" + x::uuid::create().to_string());
        this->outbox = this->open_outbox(this->client);
    }

    void TearDown() override {
        this->outbox.reset();
        std::filesystem::remove_all(this->dir);
    }

    [[nodiscard]] std::shared_ptr<::driver::outbox::Outbox> open_outbox(
        const std::shared_ptr<synnax::Synnax> &c,
        const std::size_t cap = ::driver::outbox::DEFAULT_CAP
    ) const {
        ::driver::outbox::Config cfg{
            .dir = this->dir,
            .statuses = c->statuses,
            .ranges = c->ranges,
            .cap = cap,
        };
        cfg.retry.base_interval = 10 * x::telem::MILLISECOND;
        cfg.retry.max_interval = 10 * x::telem::MILLISECOND;
        return ASSERT_NIL_P(::driver::outbox::Outbox::open(cfg));
    }

    void open_unreachable(const std::size_t cap = ::driver::outbox::DEFAULT_CAP) {
        this->outbox.reset();
        auto cfg = test_client_config();
        cfg.port = 1;
        this->outbox = this->open_outbox(std::make_shared<synnax::Synnax>(cfg), cap);
    }

    void wait_sent() const { ASSERT_EVENTUALLY_EQ(this->outbox->size(), 0); }

    /// @brief runs the node that f describes once in a module that reports through
    /// report.
    void run(const ::arc::stl::testutil::Fixture &f, Reporter report = noopReporter()) {
        Module module(this->outbox, std::move(report));
        auto node = ASSERT_NIL_P(module.create(f.make_config()));
        auto ctx = make_context();
        ASSERT_NIL(node->next(ctx));
    }

    [[nodiscard]] synnax::ranger::Range retrieve(const std::string &key) const {
        return ASSERT_NIL_P(
            this->client->ranges.retrieve(ASSERT_NIL_P(x::uuid::UUID::parse(key)))
        );
    }
};

/// @brief it should handle only the create and end nodes.
TEST_F(RangesTest, HandlesCreateAndEnd) {
    Module module(this->outbox, noopReporter());
    EXPECT_TRUE(module.handles("create"));
    EXPECT_TRUE(module.handles("end"));
    EXPECT_FALSE(module.handles("start"));
    EXPECT_FALSE(module.handles("anything_else"));
    EXPECT_EQ(module.module_name(), "ranges");
}

/// @brief it should create a create node and an end node from their bare types.
TEST_F(RangesTest, CreatesNodesFromBareTypes) {
    Module module(this->outbox, noopReporter());
    const auto create = create_spec().config({"rng", "", ""});
    auto created = ASSERT_NIL_P(module.create(create.make_config()));
    EXPECT_NE(dynamic_cast<CreateRange *>(created.get()), nullptr);
    const auto end = end_spec().config({x::uuid::create().to_string()});
    auto ended = ASSERT_NIL_P(module.create(end.make_config()));
    EXPECT_NE(dynamic_cast<EndRange *>(ended.get()), nullptr);
}

/// @brief it should create a node from its qualified type through a multi factory.
TEST_F(RangesTest, CreatesNodeWithQualifiedTypeViaMultiFactory) {
    auto spec = create_spec();
    spec.type = "ranges.create";
    const auto f = spec.config({"rng", "", ""});
    ::arc::runtime::node::MultiFactory multi(
        {std::make_shared<Module>(this->outbox, noopReporter())}
    );
    auto created = ASSERT_NIL_P(multi.create(f.make_config()));
    EXPECT_NE(dynamic_cast<CreateRange *>(created.get()), nullptr);
}

/// @brief it should return NOT_FOUND for an unknown node type.
TEST_F(RangesTest, ReturnsNotFoundForUnknownType) {
    auto spec = create_spec();
    spec.type = "not_create_or_end";
    const auto f = spec.config({"rng", "", ""});
    Module module(this->outbox, noopReporter());
    const auto created = ASSERT_OCCURRED_AS_P(
        module.create(f.make_config()),
        x::errors::NOT_FOUND
    );
    EXPECT_EQ(created, nullptr);
}

/// @brief it should emit the key of an open range at once, and create the range.
TEST_F(RangesTest, CreatesOpenRange) {
    const auto name = unique_name("cpp_create_");
    const auto f = create_spec().config({name, "", ""});
    this->run(f);
    const auto key = output_of(f);
    ASSERT_FALSE(key.empty());
    this->wait_sent();
    const auto r = this->retrieve(key);
    EXPECT_EQ(r.name, name);
    EXPECT_EQ(r.time_range.end, x::telem::TIME_STAMP_MAX);
}

/// @brief it should create the range with the color it parses.
TEST_F(RangesTest, CreatesRangeWithColor) {
    const auto f = create_spec().config({unique_name("cpp_color_"), "", "#df6d38"});
    this->run(f);
    this->wait_sent();
    const auto r = this->retrieve(output_of(f));
    ASSERT_TRUE(r.color.has_value());
    EXPECT_EQ(r.color.value().r, 0xdf);
    EXPECT_EQ(r.color.value().g, 0x6d);
    EXPECT_EQ(r.color.value().b, 0x38);
}

/// @brief it should create the range under its parent.
TEST_F(RangesTest, CreatesRangeUnderParent) {
    synnax::ranger::Range parent;
    parent.name = unique_name("cpp_parent_");
    parent.time_range = x::telem::TimeRange{
        x::telem::TimeStamp::now(),
        x::telem::TIME_STAMP_MAX
    };
    ASSERT_NIL(this->client->ranges.create(parent));
    const auto f = create_spec().config(
        {unique_name("cpp_child_"), parent.key.to_string(), ""}
    );
    this->run(f);
    this->wait_sent();
    EXPECT_EQ(this->retrieve(output_of(f)).parent.value(), parent.key);
}

/// @brief it should warn, emit an empty key, and queue nothing for a color that is
/// not valid.
TEST_F(RangesTest, WarnsOnInvalidColor) {
    const auto reports = std::make_shared<Reports>();
    const auto f = create_spec().config(
        {unique_name("cpp_badcolor_"), "", "not-a-color"}
    );
    this->run(f, recordingReporter(reports));
    EXPECT_EQ(output_of(f), "");
    EXPECT_EQ(this->outbox->size(), 0);
    const auto calls = reports->snapshot();
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].message, "Failed to create range");
    EXPECT_TRUE(
        calls[0].description.starts_with("The color \"not-a-color\" is not valid: ")
    );
}

/// @brief it should warn, emit an empty key, and queue nothing for a blank name.
TEST_F(RangesTest, WarnsOnBlankName) {
    const auto reports = std::make_shared<Reports>();
    const auto f = create_spec().config({" ", "", ""});
    this->run(f, recordingReporter(reports));
    EXPECT_EQ(output_of(f), "");
    EXPECT_EQ(this->outbox->size(), 0);
    const auto calls = reports->snapshot();
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].message, "Failed to create range");
    EXPECT_EQ(calls[0].description, "The range name is empty.");
}

/// @brief it should warn and drop a create the Core rejects.
TEST_F(RangesTest, WarnsWhenCoreRejectsCreate) {
    const auto reports = std::make_shared<Reports>();
    const auto f = create_spec().config(
        {unique_name("cpp_orphan_"), x::uuid::create().to_string(), ""}
    );
    Module module(this->outbox, recordingReporter(reports));
    auto node = ASSERT_NIL_P(module.create(f.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    this->wait_sent();
    EXPECT_EQ(this->outbox->dropped(), 1);
    const auto calls = reports->snapshot();
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].message, "Failed to create range");
    EXPECT_FALSE(calls[0].description.empty());
}

/// @brief it should read a var-bound parent when the node runs.
TEST_F(RangesTest, ReadsAVarBoundParentAtFireTime) {
    const auto reports = std::make_shared<Reports>();
    // The configured "" would succeed parentless; only the live slot value
    // can produce this warning.
    const auto f = create_spec().config(
        {unique_name("cpp_var_parent_"),
         std::shared_ptr<::arc::stl::testutil::VarBinding>(
             ::arc::stl::testutil::var_of<std::string>("not-a-uuid")
         ),
         ""}
    );
    this->run(f, recordingReporter(reports));
    EXPECT_EQ(output_of(f), "");
    const auto calls = reports->snapshot();
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].message, "Failed to create range");
    EXPECT_EQ(
        calls[0].description,
        "The parent key \"not-a-uuid\" is not a valid range key."
    );
}

/// @brief it should count a create as sent when the Core already holds its key.
TEST_F(RangesTest, CountsAnExistingRangeAsCreated) {
    synnax::ranger::Range r;
    r.key = x::uuid::create();
    r.name = unique_name("cpp_exists_");
    r.time_range = x::telem::TimeRange{
        x::telem::TimeStamp::now(),
        x::telem::TIME_STAMP_MAX
    };
    ASSERT_NIL(this->client->ranges.create(r));
    ASSERT_NIL(this->outbox->create_range(r, nullptr));
    this->wait_sent();
    EXPECT_EQ(this->outbox->dropped(), 0);
}

/// @brief it should set the end of the range to the time the node runs.
TEST_F(RangesTest, EndsRangeAtTimeOfRun) {
    synnax::ranger::Range r;
    r.name = unique_name("cpp_end_");
    r.time_range = x::telem::TimeRange{
        x::telem::TimeStamp::now(),
        x::telem::TIME_STAMP_MAX
    };
    ASSERT_NIL(this->client->ranges.create(r));
    const auto f = end_spec().config({r.key.to_string()});
    const auto before = x::telem::TimeStamp::now();
    this->run(f);
    const auto after = x::telem::TimeStamp::now();
    EXPECT_EQ(output_of(f), r.key.to_string());
    this->wait_sent();
    const auto updated = this->retrieve(r.key.to_string());
    EXPECT_GE(updated.time_range.end, before);
    EXPECT_LE(updated.time_range.end, after);
}

/// @brief it should warn and drop the end of a range that does not exist.
TEST_F(RangesTest, WarnsWhenEndedRangeDoesNotExist) {
    const auto reports = std::make_shared<Reports>();
    const auto f = end_spec().config({x::uuid::create().to_string()});
    Module module(this->outbox, recordingReporter(reports));
    auto node = ASSERT_NIL_P(module.create(f.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(node->next(ctx));
    this->wait_sent();
    EXPECT_EQ(this->outbox->dropped(), 1);
    const auto calls = reports->snapshot();
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].message, "Failed to end range");
    EXPECT_FALSE(calls[0].description.empty());
}

/// @brief it should warn and queue nothing for a key that is not valid.
TEST_F(RangesTest, WarnsOnInvalidEndKey) {
    const auto reports = std::make_shared<Reports>();
    const auto f = end_spec().config({"not-a-uuid"});
    this->run(f, recordingReporter(reports));
    EXPECT_EQ(output_of(f), "");
    EXPECT_EQ(this->outbox->size(), 0);
    const auto calls = reports->snapshot();
    ASSERT_EQ(calls.size(), 1u);
    EXPECT_EQ(calls[0].message, "Failed to end range");
    EXPECT_EQ(calls[0].description, "The key \"not-a-uuid\" is not a valid range key.");
}

/// @brief it should refuse and count a create when the outbox is full, and still
/// take an end.
TEST_F(RangesTest, RefusesCreateButNotEndWhenFull) {
    this->open_unreachable(1);
    const auto reports = std::make_shared<Reports>();
    Module module(this->outbox, recordingReporter(reports));
    const auto first = create_spec().config({unique_name("cpp_full_"), "", ""});
    auto first_node = ASSERT_NIL_P(module.create(first.make_config()));
    const auto second = create_spec().config({unique_name("cpp_full_"), "", ""});
    auto second_node = ASSERT_NIL_P(module.create(second.make_config()));
    auto ctx = make_context();
    ASSERT_NIL(first_node->next(ctx));
    ASSERT_NIL(second_node->next(ctx));
    EXPECT_FALSE(output_of(first).empty());
    EXPECT_EQ(output_of(second), "");
    EXPECT_EQ(this->outbox->refused(), 1);
    const auto end = end_spec().config({output_of(first)});
    auto end_node = ASSERT_NIL_P(module.create(end.make_config()));
    ASSERT_NIL(end_node->next(ctx));
    EXPECT_EQ(this->outbox->size(), 2);
    ASSERT_EVENTUALLY_EQ(reports->snapshot().size(), 1u);
    EXPECT_EQ(reports->snapshot()[0].message, "Failed to create range");
}

/// @brief it should warn from the outbox thread when the full outbox refuses an end,
/// while the Core stays unreachable.
TEST_F(RangesTest, WarnsOnRefusedEndWhileCoreUnreachable) {
    this->open_unreachable(1);
    const auto reports = std::make_shared<Reports>();
    Module module(this->outbox, recordingReporter(reports));
    auto ctx = make_context();
    const auto create = create_spec().config({unique_name("cpp_full_end_"), "", ""});
    auto create_node = ASSERT_NIL_P(module.create(create.make_config()));
    ASSERT_NIL(create_node->next(ctx));
    const auto end = end_spec().config({output_of(create)});
    auto end_node = ASSERT_NIL_P(module.create(end.make_config()));
    ASSERT_NIL(end_node->next(ctx));
    ASSERT_NIL(end_node->next(ctx));
    EXPECT_EQ(output_of(end), "");
    EXPECT_EQ(this->outbox->refused(), 1);
    ASSERT_EVENTUALLY_EQ(reports->snapshot().size(), 1u);
    const auto calls = reports->snapshot();
    EXPECT_EQ(calls[0].message, "Failed to end range");
    EXPECT_FALSE(calls[0].description.empty());
}

/// @brief it should report the refused calls of each message once, with their count
/// and reason.
TEST(CallsTest, ReportsRefusedCallsOnce) {
    const auto reports = std::make_shared<Reports>();
    Calls calls(recordingReporter(reports));
    EXPECT_TRUE(calls.refuse(CREATE_FAILURE_MSG, "outbox is full."));
    EXPECT_FALSE(calls.refuse(CREATE_FAILURE_MSG, "outbox is full."));
    EXPECT_FALSE(calls.refuse(END_FAILURE_MSG, "outbox is full."));
    calls.report_refused();
    calls.report_refused();
    const auto got = reports->snapshot();
    ASSERT_EQ(got.size(), 2u);
    EXPECT_EQ(got[0].message, CREATE_FAILURE_MSG);
    EXPECT_EQ(got[0].description, "outbox is full. Dropped 2 calls.");
    EXPECT_EQ(got[1].message, END_FAILURE_MSG);
    EXPECT_EQ(got[1].description, "outbox is full. Dropped 1 calls.");
}

#ifndef _WIN32
/// @brief it should create and end a range while the Core is unreachable, and the
/// Core should hold the range with the times of the runs after it reconnects.
TEST_F(RangesTest, CreatesAndEndsRangeWhileCoreUnreachable) {
    CorePipe pipe(test_client_config().port);
    pipe.pause();
    auto cfg = test_client_config();
    cfg.host = "127.0.0.1";
    cfg.port = pipe.port;
    this->outbox.reset();
    this->outbox = this->open_outbox(std::make_shared<synnax::Synnax>(cfg));
    Module module(this->outbox, noopReporter());
    auto ctx = make_context();

    const auto name = unique_name("cpp_outage_");
    const auto create = create_spec().config({name, "", ""});
    auto create_node = ASSERT_NIL_P(module.create(create.make_config()));
    const auto created_from = x::telem::TimeStamp::now();
    ASSERT_NIL(create_node->next(ctx));
    const auto created_to = x::telem::TimeStamp::now();
    const auto key = output_of(create);
    ASSERT_FALSE(key.empty());

    const auto end = end_spec().config({key});
    auto end_node = ASSERT_NIL_P(module.create(end.make_config()));
    const auto ended_from = x::telem::TimeStamp::now();
    ASSERT_NIL(end_node->next(ctx));
    const auto ended_to = x::telem::TimeStamp::now();
    EXPECT_EQ(output_of(end), key);
    EXPECT_LT(ended_to - created_from, x::telem::SECOND);
    EXPECT_EQ(this->outbox->size(), 2);

    pipe.resume();
    this->wait_sent();
    EXPECT_EQ(this->outbox->dropped(), 0);
    const auto r = this->retrieve(key);
    EXPECT_EQ(r.name, name);
    EXPECT_GE(r.time_range.start, created_from);
    EXPECT_LE(r.time_range.start, created_to);
    EXPECT_GE(r.time_range.end, ended_from);
    EXPECT_LE(r.time_range.end, ended_to);
    this->outbox.reset();
}

/// @brief it should report the creates a full outbox refused once it sends a call.
TEST_F(RangesTest, ReportsRefusedCreatesOnceSent) {
    CorePipe pipe(test_client_config().port);
    pipe.pause();
    auto cfg = test_client_config();
    cfg.host = "127.0.0.1";
    cfg.port = pipe.port;
    this->outbox.reset();
    this->outbox = this->open_outbox(std::make_shared<synnax::Synnax>(cfg), 1);
    const auto reports = std::make_shared<Reports>();
    Module module(this->outbox, recordingReporter(reports));
    auto ctx = make_context();
    const auto first = create_spec().config({unique_name("cpp_drain_"), "", ""});
    auto first_node = ASSERT_NIL_P(module.create(first.make_config()));
    const auto second = create_spec().config({unique_name("cpp_drain_"), "", ""});
    auto second_node = ASSERT_NIL_P(module.create(second.make_config()));
    ASSERT_NIL(first_node->next(ctx));
    ASSERT_NIL(second_node->next(ctx));
    EXPECT_EQ(this->outbox->refused(), 1);

    pipe.resume();
    this->wait_sent();
    ASSERT_EVENTUALLY_EQ(reports->snapshot().size(), 1u);
    const auto calls = reports->snapshot();
    EXPECT_EQ(calls[0].message, "Failed to create range");
    EXPECT_TRUE(calls[0].description.ends_with(" Dropped 1 calls."));
    EXPECT_FALSE(this->retrieve(output_of(first)).name.empty());
    this->outbox.reset();
}
#endif
}
