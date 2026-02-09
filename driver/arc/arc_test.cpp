// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "driver/arc/arc.h"
#include "driver/arc/task.h"
#include "driver/pipeline/mock/pipeline.h"
#include "driver/task/task.h"

namespace driver::arc {
TEST(TaskConfigParsing, DefaultLoopConfig) {
    nlohmann::json cfg{{"arc_key", "test-arc-key"}};
    auto parser = x::json::Parser(nlohmann::to_string(cfg));
    arc::TaskConfig task_cfg(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(task_cfg.loop.mode, ::arc::runtime::loop::ExecutionMode::AUTO);
    EXPECT_EQ(task_cfg.loop.rt_priority, ::arc::runtime::loop::DEFAULT_RT_PRIORITY);
    EXPECT_EQ(task_cfg.loop.cpu_affinity, ::arc::runtime::loop::CPU_AFFINITY_AUTO);
    EXPECT_FALSE(task_cfg.loop.lock_memory);
}

TEST(TaskConfigParsing, ExplicitExecutionMode) {
    nlohmann::json cfg{{"arc_key", "test-arc-key"}, {"execution_mode", "BUSY_WAIT"}};
    auto parser = x::json::Parser(nlohmann::to_string(cfg));
    arc::TaskConfig task_cfg(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(task_cfg.loop.mode, ::arc::runtime::loop::ExecutionMode::BUSY_WAIT);
}

TEST(TaskConfigParsing, AllExecutionModes) {
    std::vector<std::pair<std::string, ::arc::runtime::loop::ExecutionMode>> modes = {
        {"AUTO", ::arc::runtime::loop::ExecutionMode::AUTO},
        {"BUSY_WAIT", ::arc::runtime::loop::ExecutionMode::BUSY_WAIT},
        {"HIGH_RATE", ::arc::runtime::loop::ExecutionMode::HIGH_RATE},
        {"RT_EVENT", ::arc::runtime::loop::ExecutionMode::RT_EVENT},
        {"HYBRID", ::arc::runtime::loop::ExecutionMode::HYBRID},
        {"EVENT_DRIVEN", ::arc::runtime::loop::ExecutionMode::EVENT_DRIVEN},
    };
    for (const auto &[mode_str, expected_mode]: modes) {
        nlohmann::json cfg{{"arc_key", "test-arc-key"}, {"execution_mode", mode_str}};
        auto parser = x::json::Parser(nlohmann::to_string(cfg));
        arc::TaskConfig task_cfg(parser);
        ASSERT_TRUE(parser.ok()) << "Failed to parse mode: " << mode_str;
        EXPECT_EQ(task_cfg.loop.mode, expected_mode)
            << "Mode mismatch for: " << mode_str;
    }
}

TEST(TaskConfigParsing, InvalidExecutionMode) {
    nlohmann::json cfg{{"arc_key", "test-arc-key"}, {"execution_mode", "INVALID_MODE"}};
    auto parser = x::json::Parser(nlohmann::to_string(cfg));
    arc::TaskConfig task_cfg(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(TaskConfigParsing, RtPriority) {
    nlohmann::json cfg{{"arc_key", "test-arc-key"}, {"rt_priority", 99}};
    auto parser = x::json::Parser(nlohmann::to_string(cfg));
    arc::TaskConfig task_cfg(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(task_cfg.loop.rt_priority, 99);
}

TEST(TaskConfigParsing, CpuAffinity) {
    nlohmann::json cfg{{"arc_key", "test-arc-key"}, {"cpu_affinity", 3}};
    auto parser = x::json::Parser(nlohmann::to_string(cfg));
    arc::TaskConfig task_cfg(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(task_cfg.loop.cpu_affinity, 3);
}

TEST(TaskConfigParsing, CpuAffinityNone) {
    nlohmann::json cfg{
        {"arc_key", "test-arc-key"},
        {"cpu_affinity", ::arc::runtime::loop::CPU_AFFINITY_NONE}
    };
    auto parser = x::json::Parser(nlohmann::to_string(cfg));
    arc::TaskConfig task_cfg(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(task_cfg.loop.cpu_affinity, ::arc::runtime::loop::CPU_AFFINITY_NONE);
}

TEST(TaskConfigParsing, LockMemory) {
    nlohmann::json cfg{{"arc_key", "test-arc-key"}, {"lock_memory", true}};
    auto parser = x::json::Parser(nlohmann::to_string(cfg));
    arc::TaskConfig task_cfg(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_TRUE(task_cfg.loop.lock_memory);
}

TEST(TaskConfigParsing, FullLoopConfig) {
    nlohmann::json cfg{
        {"arc_key", "test-arc-key"},
        {"execution_mode", "RT_EVENT"},
        {"rt_priority", 80},
        {"cpu_affinity", 7},
        {"lock_memory", true}
    };
    auto parser = x::json::Parser(nlohmann::to_string(cfg));
    arc::TaskConfig task_cfg(parser);
    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(task_cfg.loop.mode, ::arc::runtime::loop::ExecutionMode::RT_EVENT);
    EXPECT_EQ(task_cfg.loop.rt_priority, 80);
    EXPECT_EQ(task_cfg.loop.cpu_affinity, 7);
    EXPECT_TRUE(task_cfg.loop.lock_memory);
}

TEST(ArcTests, testCalcDoubling) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto input_idx_name = make_unique_channel_name("ox_pt_1_idx");
    auto input_name = make_unique_channel_name("ox_pt_1");
    auto output_idx_name = make_unique_channel_name("ox_pt_doubled_idx");
    auto output_name = make_unique_channel_name("ox_pt_doubled");

    auto input_idx = synnax::channel::Channel(
        input_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(input_idx));
    auto output_idx = synnax::channel::Channel(
        output_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(output_idx));

    auto input_ch = synnax::channel::Channel(
        input_name,
        x::telem::FLOAT32_T,
        input_idx.key,
        false
    );
    auto output_ch = synnax::channel::Channel(
        output_name,
        x::telem::FLOAT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));
    ASSERT_NIL(client->channels.create(output_ch));

    synnax::arc::Arc arc_prog(make_unique_channel_name("calc_test"));
    arc_prog.text = ::arc::text::Text(
        "func calc(val f32) f32 {\n"
        "    return val * 2\n"
        "}\n" +
        input_name + " -> calc{} -> " + output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_calc_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame input_fr(2);
    auto now = x::telem::TimeStamp::now();
    auto input_idx_series = x::telem::Series(now);
    input_idx_series.alignment = x::telem::Alignment(1, 0);
    auto input_val_series = x::telem::Series(5.0f);
    input_val_series.alignment = x::telem::Alignment(1, 0);
    input_fr.emplace(input_idx.key, std::move(input_idx_series));
    input_fr.emplace(input_ch.key, std::move(input_val_series));
    input_frames->push_back(std::move(input_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("test_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    ASSERT_EVENTUALLY_GE(mock_writer->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

    auto &output_fr = mock_writer->writes->at(0);
    ASSERT_TRUE(output_fr.contains(output_ch.key));
    auto output_val = output_fr.at<float>(output_ch.key, 0);
    EXPECT_FLOAT_EQ(output_val, 10.0f);

    task->stop("test_stop", true);
}

TEST(ArcTests, testBasicSequence) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    // Create trigger channel (start_cmd)
    auto start_cmd_idx_name = make_unique_channel_name("start_cmd_idx");
    auto start_cmd_name = make_unique_channel_name("start_cmd");
    auto start_cmd_idx = synnax::channel::Channel(
        start_cmd_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(start_cmd_idx));
    auto start_cmd_ch = synnax::channel::Channel(
        start_cmd_name,
        x::telem::UINT8_T,
        start_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(start_cmd_ch));

    // Create output channel (valve_cmd)
    auto valve_cmd_idx_name = make_unique_channel_name("valve_cmd_idx");
    auto valve_cmd_name = make_unique_channel_name("valve_cmd");
    auto valve_cmd_idx = synnax::channel::Channel(
        valve_cmd_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(valve_cmd_idx));
    auto valve_cmd_ch = synnax::channel::Channel(
        valve_cmd_name,
        x::telem::INT64_T,
        valve_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(valve_cmd_ch));

    // Create Arc program with the sequence
    synnax::arc::Arc arc_prog(make_unique_channel_name("sequence_test"));
    arc_prog.text = ::arc::text::Text(
        "sequence main {\n"
        "    stage run {\n"
        "        1 -> " +
        valve_cmd_name +
        "\n"
        "    }\n"
        "}\n"
        "\n" +
        start_cmd_name + " => main\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    // Create rack and task
    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_sequence_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_sequence_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    // Setup mock writer to capture outputs
    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    // Setup mock streamer to send trigger frame
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame trigger_fr(2);
    auto now = x::telem::TimeStamp::now();
    auto trigger_idx_series = x::telem::Series(now);
    trigger_idx_series.alignment = x::telem::Alignment(1, 0);
    auto trigger_val_series = x::telem::Series(static_cast<uint8_t>(1));
    trigger_val_series.alignment = x::telem::Alignment(1, 0);
    trigger_fr.emplace(start_cmd_idx.key, std::move(trigger_idx_series));
    trigger_fr.emplace(start_cmd_ch.key, std::move(trigger_val_series));
    input_frames->push_back(std::move(trigger_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {start_cmd_idx.key, start_cmd_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("test_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    // Wait for the sequence to execute and write to valve_cmd
    ASSERT_EVENTUALLY_GE(mock_writer->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

    // Verify valve_cmd received the value 1
    bool found_valve_cmd = false;
    for (const auto &output_fr: *mock_writer->writes) {
        if (output_fr.contains(valve_cmd_ch.key)) {
            auto output_val = output_fr.at<int64_t>(valve_cmd_ch.key, 0);
            EXPECT_EQ(output_val, 1);
            found_valve_cmd = true;
            break;
        }
    }
    EXPECT_TRUE(found_valve_cmd) << "valve_cmd channel was not written to";

    task->stop("test_stop", true);
}

TEST(ArcTests, testOneShotTruthiness) {
    // This test verifies that one-shot edges (=>) only fire when the value is truthy.
    // A value of 0 should NOT trigger the sequence, while a value of 1 should.
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    // Create trigger channel (start_cmd)
    auto start_cmd_idx_name = make_unique_channel_name("truthiness_start_cmd_idx");
    auto start_cmd_name = make_unique_channel_name("truthiness_start_cmd");
    auto start_cmd_idx = synnax::channel::Channel(
        start_cmd_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(start_cmd_idx));
    auto start_cmd_ch = synnax::channel::Channel(
        start_cmd_name,
        x::telem::UINT8_T,
        start_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(start_cmd_ch));

    // Create output channel (valve_cmd)
    auto valve_cmd_idx_name = make_unique_channel_name("truthiness_valve_cmd_idx");
    auto valve_cmd_name = make_unique_channel_name("truthiness_valve_cmd");
    auto valve_cmd_idx = synnax::channel::Channel(
        valve_cmd_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(valve_cmd_idx));
    auto valve_cmd_ch = synnax::channel::Channel(
        valve_cmd_name,
        x::telem::INT64_T,
        valve_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(valve_cmd_ch));

    // Create Arc program with a sequence triggered by one-shot edge
    synnax::arc::Arc arc_prog(make_unique_channel_name("truthiness_test"));
    arc_prog.text = ::arc::text::Text(
        "sequence main {\n"
        "    stage run {\n"
        "        42 -> " +
        valve_cmd_name +
        "\n"
        "    }\n"
        "}\n"
        "\n" +
        start_cmd_name + " => main\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    // Create rack and task
    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_truthiness_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_truthiness_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    // Setup mock writer to capture outputs
    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    // Setup mock streamer to send TWO frames:
    // 1. First frame with value 0 (should NOT trigger)
    // 2. Second frame with value 1 (should trigger)
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();

    // Frame 1: falsy trigger (0) - should NOT trigger the sequence
    x::telem::Frame falsy_trigger_fr(2);
    auto now = x::telem::TimeStamp::now();
    auto falsy_idx_series = x::telem::Series(now);
    falsy_idx_series.alignment = x::telem::Alignment(1, 0);
    auto falsy_val_series = x::telem::Series(static_cast<uint8_t>(0));
    falsy_val_series.alignment = x::telem::Alignment(1, 0);
    falsy_trigger_fr.emplace(start_cmd_idx.key, std::move(falsy_idx_series));
    falsy_trigger_fr.emplace(start_cmd_ch.key, std::move(falsy_val_series));
    input_frames->push_back(std::move(falsy_trigger_fr));

    // Frame 2: truthy trigger (1) - should trigger the sequence
    x::telem::Frame truthy_trigger_fr(2);
    auto later = x::telem::TimeStamp::now() + x::telem::SECOND;
    auto truthy_idx_series = x::telem::Series(later);
    truthy_idx_series.alignment = x::telem::Alignment(1, 1);
    auto truthy_val_series = x::telem::Series(static_cast<uint8_t>(1));
    truthy_val_series.alignment = x::telem::Alignment(1, 1);
    truthy_trigger_fr.emplace(start_cmd_idx.key, std::move(truthy_idx_series));
    truthy_trigger_fr.emplace(start_cmd_ch.key, std::move(truthy_val_series));
    input_frames->push_back(std::move(truthy_trigger_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {start_cmd_idx.key, start_cmd_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("test_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    // Wait for the sequence to execute and write to valve_cmd
    ASSERT_EVENTUALLY_GE(mock_writer->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

    // Verify valve_cmd received the value 42 (from the sequence stage)
    // This confirms the sequence was triggered only by the truthy value (1),
    // not by the falsy value (0)
    bool found_valve_cmd = false;
    for (const auto &output_fr: *mock_writer->writes) {
        if (output_fr.contains(valve_cmd_ch.key)) {
            auto output_val = output_fr.at<int64_t>(valve_cmd_ch.key, 0);
            EXPECT_EQ(output_val, 42);
            found_valve_cmd = true;
            break;
        }
    }
    EXPECT_TRUE(found_valve_cmd)
        << "valve_cmd channel was not written to - "
           "sequence should have been triggered by truthy value (1)";

    task->stop("test_stop", true);
}

TEST(ArcTests, testTwoStageSequenceWithTransition) {
    // This test verifies two-stage sequence execution with per-stage stratification.
    // It tests the two-tier execution model where:
    // 1. Each stage has independent strata (stage-local sources at stratum 0)
    // 2. Stage transitions work correctly via the convergence loop
    //
    // Sequence flow:
    // - start_cmd triggers sequence entry to "pressurize" stage
    // - "pressurize" stage: outputs 1 to valve, monitors pressure for transition
    // - When pressure > 50, transitions to "idle" stage
    // - "idle" stage: outputs 0 to valve
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    // Create trigger channel (start_cmd)
    auto start_cmd_idx_name = make_unique_channel_name("two_stage_start_cmd_idx");
    auto start_cmd_name = make_unique_channel_name("two_stage_start_cmd");
    auto start_cmd_idx = synnax::channel::Channel(
        start_cmd_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(start_cmd_idx));
    auto start_cmd_ch = synnax::channel::Channel(
        start_cmd_name,
        x::telem::UINT8_T,
        start_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(start_cmd_ch));

    // Create pressure sensor channel
    auto pressure_idx_name = make_unique_channel_name("two_stage_pressure_idx");
    auto pressure_name = make_unique_channel_name("two_stage_pressure");
    auto pressure_idx = synnax::channel::Channel(
        pressure_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(pressure_idx));
    auto pressure_ch = synnax::channel::Channel(
        pressure_name,
        x::telem::FLOAT32_T,
        pressure_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(pressure_ch));

    // Create output channel (valve_cmd)
    auto valve_cmd_idx_name = make_unique_channel_name("two_stage_valve_cmd_idx");
    auto valve_cmd_name = make_unique_channel_name("two_stage_valve_cmd");
    auto valve_cmd_idx = synnax::channel::Channel(
        valve_cmd_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(valve_cmd_idx));
    auto valve_cmd_ch = synnax::channel::Channel(
        valve_cmd_name,
        x::telem::INT64_T,
        valve_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(valve_cmd_ch));

    // Create Arc program with a two-stage sequence
    // Stage "pressurize": outputs 1, transitions to "idle" when pressure > 50
    // Stage "idle": outputs 0 (terminal stage)
    synnax::arc::Arc arc_prog(make_unique_channel_name("two_stage_test"));
    arc_prog.text = ::arc::text::Text(
        "sequence main {\n"
        "    stage pressurize {\n"
        "        1 -> " +
        valve_cmd_name +
        ",\n"
        "        " +
        pressure_name + " -> " + pressure_name +
        " > 50 => next\n"
        "    }\n"
        "    stage idle {\n"
        "        0 -> " +
        valve_cmd_name +
        "\n"
        "    }\n"
        "}\n"
        "\n" +
        start_cmd_name + " => main\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    // Create rack and task
    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_two_stage_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_two_stage_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    // Setup mock writer to capture outputs
    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    // Setup mock streamer to send frames:
    // 1. Trigger frame to start the sequence
    // 2. Pressure frame with value < 50 (should stay in pressurize stage)
    // 3. Pressure frame with value > 50 (should transition to idle stage)
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();

    // Frame 1: Trigger the sequence
    x::telem::Frame trigger_fr(4);
    auto now = x::telem::TimeStamp::now();
    auto sequence_trigger_idx = x::telem::Series(now);
    sequence_trigger_idx.alignment = x::telem::Alignment(1, 0);
    trigger_fr.emplace(start_cmd_idx.key, std::move(sequence_trigger_idx));
    auto sequence_trigger_data = x::telem::Series(static_cast<std::uint8_t>(1));
    sequence_trigger_data.alignment = x::telem::Alignment(1, 0);
    trigger_fr.emplace(start_cmd_ch.key, std::move(sequence_trigger_data));
    auto pressure_idx_series = x::telem::Series(now);
    pressure_idx_series.alignment = x::telem::Alignment(1, 0);
    trigger_fr.emplace(pressure_idx.key, std::move(pressure_idx_series));
    auto pressure_val_series = x::telem::Series(10.0f);
    pressure_val_series.alignment = x::telem::Alignment(1, 0);
    trigger_fr.emplace(pressure_ch.key, std::move(pressure_val_series));
    input_frames->push_back(std::move(trigger_fr));

    // Frame 2: Pressure still low - should stay in pressurize, output 1
    x::telem::Frame low_pressure_fr(2);
    auto t2 = now + x::telem::MILLISECOND * 100;
    auto low_pressure_idx_series = x::telem::Series(t2);
    low_pressure_idx_series.alignment = x::telem::Alignment(1, 1);
    auto low_pressure_val_series = x::telem::Series(30.0f);
    low_pressure_val_series.alignment = x::telem::Alignment(1, 1);
    low_pressure_fr.emplace(pressure_idx.key, std::move(low_pressure_idx_series));
    low_pressure_fr.emplace(pressure_ch.key, std::move(low_pressure_val_series));
    input_frames->push_back(std::move(low_pressure_fr));

    // Frame 3: Pressure exceeds threshold - should transition to idle, output 0
    x::telem::Frame high_pressure_fr(2);
    auto t3 = now + x::telem::MILLISECOND * 200;
    auto high_pressure_idx_series = x::telem::Series(t3);
    high_pressure_idx_series.alignment = x::telem::Alignment(1, 2);
    auto high_pressure_val_series = x::telem::Series(60.0f);
    high_pressure_val_series.alignment = x::telem::Alignment(1, 2);
    high_pressure_fr.emplace(pressure_idx.key, std::move(high_pressure_idx_series));
    high_pressure_fr.emplace(pressure_ch.key, std::move(high_pressure_val_series));
    input_frames->push_back(std::move(high_pressure_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {start_cmd_idx.key, start_cmd_ch.key, pressure_idx.key, pressure_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("test_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    // Wait for multiple writes (at least 2: one from pressurize stage, one from idle
    // stage)
    ASSERT_EVENTUALLY_GE(mock_writer->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 2);

    // Verify we got both valve states:
    // - At least one write with value 1 (from pressurize stage)
    // - At least one write with value 0 (from idle stage after transition)
    bool found_pressurize_output = false;
    bool found_idle_output = false;

    for (const auto &output_fr: *mock_writer->writes) {
        if (output_fr.contains(valve_cmd_ch.key)) {
            auto output_val = output_fr.at<int64_t>(valve_cmd_ch.key, 0);
            if (output_val == 1) {
                found_pressurize_output = true;
            } else if (output_val == 0) {
                found_idle_output = true;
            }
        }
    }

    EXPECT_TRUE(found_pressurize_output)
        << "valve_cmd should have received value 1 from pressurize stage";
    EXPECT_TRUE(found_idle_output)
        << "valve_cmd should have received value 0 from idle stage after transition";

    task->stop("test_stop", true);
}

synnax::task::Status *find_status_by_variant(
    std::vector<synnax::task::Status> &statuses,
    const std::string &variant
) {
    for (auto &s: statuses)
        if (s.variant == variant) return &s;
    return nullptr;
}

void expect_status(
    const synnax::task::Status &status,
    const std::string &expected_variant,
    bool expected_running,
    const std::string &message_contains = ""
) {
    EXPECT_EQ(status.variant, expected_variant);
    EXPECT_EQ(status.details.running, expected_running);
    if (!message_contains.empty()) {
        EXPECT_TRUE(status.message.find(message_contains) != std::string::npos)
            << "Expected message to contain '" << message_contains
            << "' but got: " << status.message;
    }
}

TEST(ArcErrorHandling, WasmTrapTriggersFatalError) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto input_idx_name = make_unique_channel_name("trap_input_idx");
    auto input_name = make_unique_channel_name("trap_input");
    auto output_idx_name = make_unique_channel_name("trap_output_idx");
    auto output_name = make_unique_channel_name("trap_output");

    auto input_idx = synnax::channel::Channel(
        input_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(input_idx));
    auto output_idx = synnax::channel::Channel(
        output_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(output_idx));

    auto input_ch = synnax::channel::Channel(
        input_name,
        x::telem::INT32_T,
        input_idx.key,
        false
    );
    auto output_ch = synnax::channel::Channel(
        output_name,
        x::telem::INT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));
    ASSERT_NIL(client->channels.create(output_ch));

    synnax::arc::Arc arc_prog(make_unique_channel_name("trap_test"));
    arc_prog.text = ::arc::text::Text(
        "func divide_by_zero(val i32) i32 { return val / 0 }\n" + input_name +
        " -> divide_by_zero{} -> " + output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_trap_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_trap_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame trigger_fr(2);
    auto now = x::telem::TimeStamp::now();
    auto trigger_idx_series = x::telem::Series(now);
    trigger_idx_series.alignment = x::telem::Alignment(1, 0);
    auto trigger_val_series = x::telem::Series(static_cast<int32_t>(42));
    trigger_val_series.alignment = x::telem::Alignment(1, 0);
    trigger_fr.emplace(input_idx.key, std::move(trigger_idx_series));
    trigger_fr.emplace(input_ch.key, std::move(trigger_val_series));
    input_frames->push_back(std::move(trigger_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("test_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    std::this_thread::sleep_for(std::chrono::milliseconds(300));

    auto *error_status = find_status_by_variant(
        ctx->statuses,
        x::status::VARIANT_ERROR
    );
    ASSERT_NE(error_status, nullptr) << "Fatal WASM trap should produce error status";
    expect_status(*error_status, x::status::VARIANT_ERROR, false);

    task->stop("test_stop", true);
}

TEST(ArcErrorHandling, RestartAfterWasmTrap) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto input_idx_name = make_unique_channel_name("restart_trap_input_idx");
    auto input_name = make_unique_channel_name("restart_trap_input");
    auto output_idx_name = make_unique_channel_name("restart_trap_output_idx");
    auto output_name = make_unique_channel_name("restart_trap_output");

    auto input_idx = synnax::channel::Channel(
        input_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(input_idx));
    auto output_idx = synnax::channel::Channel(
        output_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(output_idx));

    auto input_ch = synnax::channel::Channel(
        input_name,
        x::telem::INT32_T,
        input_idx.key,
        false
    );
    auto output_ch = synnax::channel::Channel(
        output_name,
        x::telem::INT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));
    ASSERT_NIL(client->channels.create(output_ch));

    synnax::arc::Arc arc_prog(make_unique_channel_name("restart_trap_test"));
    arc_prog.text = ::arc::text::Text(
        "func maybe_trap(val i32) i32 {\n"
        "    if val == 0 { return 1 / val }\n"
        "    return val * 2\n"
        "}\n" +
        input_name + " -> maybe_trap{} -> " + output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_restart_trap_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_restart_trap_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();

    x::telem::Frame trap_trigger(2);
    auto now = x::telem::TimeStamp::now();
    auto trap_idx = x::telem::Series(now);
    trap_idx.alignment = x::telem::Alignment(1, 0);
    auto trap_val = x::telem::Series(static_cast<int32_t>(0));
    trap_val.alignment = x::telem::Alignment(1, 0);
    trap_trigger.emplace(input_idx.key, std::move(trap_idx));
    trap_trigger.emplace(input_ch.key, std::move(trap_val));
    input_frames->push_back(std::move(trap_trigger));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("test_start_1");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    std::this_thread::sleep_for(std::chrono::milliseconds(300));

    auto *error_status = find_status_by_variant(
        ctx->statuses,
        x::status::VARIANT_ERROR
    );
    EXPECT_NE(error_status, nullptr) << "Should have error status after WASM trap";

    task->stop("test_stop_1", true);

    mock_writer->writes->clear();
    mock_writer->writer_opens = 0;
    ctx->statuses.clear();
    input_frames->clear();

    x::telem::Frame normal_trigger(2);
    auto now2 = x::telem::TimeStamp::now();
    auto normal_idx = x::telem::Series(now2);
    normal_idx.alignment = x::telem::Alignment(2, 0);
    auto normal_val = x::telem::Series(static_cast<int32_t>(5));
    normal_val.alignment = x::telem::Alignment(2, 0);
    normal_trigger.emplace(input_idx.key, std::move(normal_idx));
    normal_trigger.emplace(input_ch.key, std::move(normal_val));
    input_frames->push_back(std::move(normal_trigger));

    task->start("test_start_2");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

    bool found_output = false;
    for (const auto &fr: *mock_writer->writes) {
        if (fr.contains(output_ch.key)) {
            auto val = fr.at<int32_t>(output_ch.key, 0);
            EXPECT_EQ(val, 10);
            found_output = true;
            break;
        }
    }
    EXPECT_TRUE(found_output) << "Task should produce output after restart from trap";

    task->stop("test_stop_2", true);
}

TEST(ArcErrorHandling, MultipleErrorRecoveryCycles) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto input_idx_name = make_unique_channel_name("multi_cycle_input_idx");
    auto input_name = make_unique_channel_name("multi_cycle_input");
    auto output_idx_name = make_unique_channel_name("multi_cycle_output_idx");
    auto output_name = make_unique_channel_name("multi_cycle_output");

    auto input_idx = synnax::channel::Channel(
        input_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(input_idx));
    auto output_idx = synnax::channel::Channel(
        output_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(output_idx));

    auto input_ch = synnax::channel::Channel(
        input_name,
        x::telem::FLOAT32_T,
        input_idx.key,
        false
    );
    auto output_ch = synnax::channel::Channel(
        output_name,
        x::telem::FLOAT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));
    ASSERT_NIL(client->channels.create(output_ch));

    synnax::arc::Arc arc_prog(make_unique_channel_name("multi_cycle_test"));
    arc_prog.text = ::arc::text::Text(
        "func double(val f32) f32 { return val * 2 }\n" + input_name +
        " -> double{} -> " + output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_multi_cycle_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_multi_cycle_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    for (int cycle = 0; cycle < 3; cycle++) {
        mock_writer->writes->clear();
        mock_writer->writer_opens = 0;
        ctx->statuses.clear();
        input_frames->clear();

        x::telem::Frame input_fr(2);
        auto now = x::telem::TimeStamp::now();
        auto idx_series = x::telem::Series(now);
        idx_series.alignment = x::telem::Alignment(cycle + 1, 0);
        auto val_series = x::telem::Series(static_cast<float>(cycle + 1));
        val_series.alignment = x::telem::Alignment(cycle + 1, 0);
        input_fr.emplace(input_idx.key, std::move(idx_series));
        input_fr.emplace(input_ch.key, std::move(val_series));
        input_frames->push_back(std::move(input_fr));

        task->start("test_start_" + std::to_string(cycle));
        ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
        ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

        bool found_output = false;
        for (const auto &fr: *mock_writer->writes) {
            if (fr.contains(output_ch.key)) {
                auto val = fr.at<float>(output_ch.key, 0);
                EXPECT_FLOAT_EQ(val, static_cast<float>((cycle + 1) * 2));
                found_output = true;
                break;
            }
        }
        EXPECT_TRUE(found_output) << "Cycle " << cycle << " should produce output";

        task->stop("test_stop_" + std::to_string(cycle), true);
    }
}

TEST(ArcStatusVerification, StartStatusHasCorrectVariantAndRunning) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto input_idx_name = make_unique_channel_name("status_verify_input_idx");
    auto input_name = make_unique_channel_name("status_verify_input");
    auto output_idx_name = make_unique_channel_name("status_verify_output_idx");
    auto output_name = make_unique_channel_name("status_verify_output");

    auto input_idx = synnax::channel::Channel(
        input_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(input_idx));
    auto output_idx = synnax::channel::Channel(
        output_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(output_idx));

    auto input_ch = synnax::channel::Channel(
        input_name,
        x::telem::FLOAT32_T,
        input_idx.key,
        false
    );
    auto output_ch = synnax::channel::Channel(
        output_name,
        x::telem::FLOAT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));
    ASSERT_NIL(client->channels.create(output_ch));

    synnax::arc::Arc arc_prog(make_unique_channel_name("status_verify_test"));
    arc_prog.text = ::arc::text::Text(
        "func pass(val f32) f32 { return val }\n" + input_name + " -> pass{} -> " +
        output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_status_verify_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_status_verify_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();

    x::telem::Frame input_fr(2);
    auto now = x::telem::TimeStamp::now();
    auto idx_series = x::telem::Series(now);
    idx_series.alignment = x::telem::Alignment(1, 0);
    auto val_series = x::telem::Series(1.0f);
    val_series.alignment = x::telem::Alignment(1, 0);
    input_fr.emplace(input_idx.key, std::move(idx_series));
    input_fr.emplace(input_ch.key, std::move(val_series));
    input_frames->push_back(std::move(input_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("verify_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    auto *start_status = find_status_by_variant(
        ctx->statuses,
        x::status::VARIANT_SUCCESS
    );
    ASSERT_NE(start_status, nullptr) << "Should have a success status after start";
    expect_status(*start_status, x::status::VARIANT_SUCCESS, true, "started");

    task->stop("verify_stop", true);

    auto *stop_status = find_status_by_variant(
        ctx->statuses,
        x::status::VARIANT_SUCCESS
    );
    ASSERT_NE(stop_status, nullptr);

    bool found_stopped = false;
    for (const auto &s: ctx->statuses) {
        if (s.variant == x::status::VARIANT_SUCCESS && !s.details.running) {
            found_stopped = true;
            break;
        }
    }
    EXPECT_TRUE(found_stopped)
        << "Should have a success status with running=false after stop";
}

TEST(ArcEdgeCases, RapidStartStop) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto input_idx_name = make_unique_channel_name("rapid_input_idx");
    auto input_name = make_unique_channel_name("rapid_input");
    auto output_idx_name = make_unique_channel_name("rapid_output_idx");
    auto output_name = make_unique_channel_name("rapid_output");

    auto input_idx = synnax::channel::Channel(
        input_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(input_idx));
    auto output_idx = synnax::channel::Channel(
        output_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(output_idx));

    auto input_ch = synnax::channel::Channel(
        input_name,
        x::telem::FLOAT32_T,
        input_idx.key,
        false
    );
    auto output_ch = synnax::channel::Channel(
        output_name,
        x::telem::FLOAT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));
    ASSERT_NIL(client->channels.create(output_ch));

    synnax::arc::Arc arc_prog(make_unique_channel_name("rapid_test"));
    arc_prog.text = ::arc::text::Text(
        "func pass(val f32) f32 { return val }\n" + input_name + " -> pass{} -> " +
        output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_rapid_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_rapid_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    for (int i = 0; i < 5; i++) {
        task->start("rapid_start_" + std::to_string(i));
        task->stop("rapid_stop_" + std::to_string(i), true);
    }

    task->start("final_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    auto *final_status = find_status_by_variant(
        ctx->statuses,
        x::status::VARIANT_SUCCESS
    );
    ASSERT_NE(final_status, nullptr);
    EXPECT_TRUE(final_status->details.running);

    task->stop("final_stop", true);
}

TEST(ArcEdgeCases, StopWithoutStart) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto input_idx_name = make_unique_channel_name("nostart_input_idx");
    auto input_name = make_unique_channel_name("nostart_input");
    auto output_idx_name = make_unique_channel_name("nostart_output_idx");
    auto output_name = make_unique_channel_name("nostart_output");

    auto input_idx = synnax::channel::Channel(
        input_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(input_idx));
    auto output_idx = synnax::channel::Channel(
        output_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(output_idx));

    auto input_ch = synnax::channel::Channel(
        input_name,
        x::telem::FLOAT32_T,
        input_idx.key,
        false
    );
    auto output_ch = synnax::channel::Channel(
        output_name,
        x::telem::FLOAT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));
    ASSERT_NIL(client->channels.create(output_ch));

    synnax::arc::Arc arc_prog(make_unique_channel_name("nostart_test"));
    arc_prog.text = ::arc::text::Text(
        "func pass(val f32) f32 { return val }\n" + input_name + " -> pass{} -> " +
        output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_nostart_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_nostart_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->stop("stop_without_start", true);

    task->start("start_after_cold_stop");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    auto *status = find_status_by_variant(ctx->statuses, x::status::VARIANT_SUCCESS);
    ASSERT_NE(status, nullptr);

    task->stop("final_stop", true);
}

TEST(ArcTests, testChannelConfigParam) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    // Trigger channel (u8)
    auto trigger_idx_name = make_unique_channel_name("cfg_ch_trigger_idx");
    auto trigger_name = make_unique_channel_name("cfg_ch_trigger");
    auto trigger_idx = synnax::channel::Channel(
        trigger_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(trigger_idx));
    auto trigger_ch = synnax::channel::Channel(
        trigger_name,
        x::telem::UINT8_T,
        trigger_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(trigger_ch));

    // Data channel that the config param refers to (f32)
    auto data_idx_name = make_unique_channel_name("cfg_ch_data_idx");
    auto data_name = make_unique_channel_name("cfg_ch_data");
    auto data_idx = synnax::channel::Channel(
        data_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(data_idx));
    auto data_ch = synnax::channel::Channel(
        data_name,
        x::telem::FLOAT32_T,
        data_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(data_ch));

    // Output channel (f32)
    auto output_idx_name = make_unique_channel_name("cfg_ch_output_idx");
    auto output_name = make_unique_channel_name("cfg_ch_output");
    auto output_idx = synnax::channel::Channel(
        output_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(output_idx));
    auto output_ch = synnax::channel::Channel(
        output_name,
        x::telem::FLOAT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(output_ch));

    // Arc program: function with channel-typed config param
    synnax::arc::Arc arc_prog(make_unique_channel_name("cfg_ch_test"));
    arc_prog.text = ::arc::text::Text(
        "func read_data{ch chan f32}(trigger u8) f32 {\n"
        "    return ch + f32(0.0)\n"
        "}\n" +
        trigger_name + " -> read_data{ch=" + data_name + "} -> " + output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_cfg_ch_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_cfg_ch_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    // Build input frames: trigger + data channel values in same frame
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame input_fr(6);
    auto now = x::telem::TimeStamp::now();

    auto trigger_idx_series = x::telem::Series(now);
    trigger_idx_series.alignment = x::telem::Alignment(1, 0);
    auto trigger_val_series = x::telem::Series(static_cast<uint8_t>(1));
    trigger_val_series.alignment = x::telem::Alignment(1, 0);
    input_fr.emplace(trigger_idx.key, std::move(trigger_idx_series));
    input_fr.emplace(trigger_ch.key, std::move(trigger_val_series));

    auto data_idx_series = x::telem::Series(now);
    data_idx_series.alignment = x::telem::Alignment(1, 0);
    auto data_val_series = x::telem::Series(42.5f);
    data_val_series.alignment = x::telem::Alignment(1, 0);
    input_fr.emplace(data_idx.key, std::move(data_idx_series));
    input_fr.emplace(data_ch.key, std::move(data_val_series));

    input_frames->push_back(std::move(input_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {trigger_idx.key, trigger_ch.key, data_idx.key, data_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("test_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    ASSERT_EVENTUALLY_GE(mock_writer->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

    bool found_output = false;
    for (const auto &output_fr: *mock_writer->writes) {
        if (output_fr.contains(output_ch.key)) {
            auto output_val = output_fr.at<float>(output_ch.key, 0);
            EXPECT_FLOAT_EQ(output_val, 42.5f)
                << "Channel config param should read the data channel value (42.5)";
            found_output = true;
            break;
        }
    }
    EXPECT_TRUE(
        found_output
    ) << "Output channel should have been written with the config param channel value";

    task->stop("test_stop", true);
}

TEST(ArcTests, testChannelConfigParamReadWrite) {
    // Reproducer for the real-world count_rising pattern:
    // func with two chan config params - one read, one write.
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    // Input trigger channel (u8)
    auto input_idx_name = make_unique_channel_name("crw_input_idx");
    auto input_name = make_unique_channel_name("crw_input");
    auto input_idx = synnax::channel::Channel(
        input_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(input_idx));
    auto input_ch = synnax::channel::Channel(
        input_name,
        x::telem::UINT8_T,
        input_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));

    // Counter max channel - READ only config param (f32)
    auto max_idx_name = make_unique_channel_name("crw_max_idx");
    auto max_name = make_unique_channel_name("crw_max");
    auto max_idx = synnax::channel::Channel(
        max_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(max_idx));
    auto max_ch = synnax::channel::Channel(
        max_name,
        x::telem::FLOAT32_T,
        max_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(max_ch));

    // Counter output channel - WRITE config param (f32)
    auto counter_idx_name = make_unique_channel_name("crw_counter_idx");
    auto counter_name = make_unique_channel_name("crw_counter");
    auto counter_idx = synnax::channel::Channel(
        counter_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(counter_idx));
    auto counter_ch = synnax::channel::Channel(
        counter_name,
        x::telem::FLOAT32_T,
        counter_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(counter_ch));

    // Arc program mimicking count_rising:
    // counter_ch is a WRITE config param (channel_write_f32)
    // max_ch is a READ config param (channel_read_f32)
    synnax::arc::Arc arc_prog(make_unique_channel_name("crw_test"));
    arc_prog.text = ::arc::text::Text(
        "func count_rising_test{counter_ch chan f32, max_ch chan f32}(input u8) {\n"
        "    prev u8 $= input\n"
        "    counter f32 $= 0.0\n"
        "    read_val := max_ch + f32(0.0)\n"
        "    if counter < read_val {\n"
        "        counter = read_val\n"
        "    }\n"
        "    if input and not prev {\n"
        "        counter = counter + 1.0\n"
        "    }\n"
        "    counter_ch = counter\n"
        "    prev = input\n"
        "}\n" +
        input_name + " -> count_rising_test{counter_ch=" + counter_name +
        ", max_ch=" + max_name + "}\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_crw_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_crw_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    // Frame: trigger=1 (rising edge from 0), max_ch=100.0
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame input_fr(6);
    auto now = x::telem::TimeStamp::now();

    auto in_idx_s = x::telem::Series(now);
    in_idx_s.alignment = x::telem::Alignment(1, 0);
    auto in_val_s = x::telem::Series(static_cast<uint8_t>(1));
    in_val_s.alignment = x::telem::Alignment(1, 0);
    input_fr.emplace(input_idx.key, std::move(in_idx_s));
    input_fr.emplace(input_ch.key, std::move(in_val_s));

    auto max_idx_s = x::telem::Series(now);
    max_idx_s.alignment = x::telem::Alignment(1, 0);
    auto max_val_s = x::telem::Series(100.0f);
    max_val_s.alignment = x::telem::Alignment(1, 0);
    input_fr.emplace(max_idx.key, std::move(max_idx_s));
    input_fr.emplace(max_ch.key, std::move(max_val_s));

    input_frames->push_back(std::move(input_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key, max_idx.key, max_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("test_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    ASSERT_EVENTUALLY_GE(mock_writer->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

    // counter should be 100: prev $= input initializes prev=1, so no rising
    // edge. But counter < read_val (0 < 100) → counter = 100.
    // This proves the config param channel read returned 100 (not 0).
    bool found_output = false;
    for (const auto &output_fr: *mock_writer->writes) {
        if (output_fr.contains(counter_ch.key)) {
            auto output_val = output_fr.at<float>(counter_ch.key, 0);
            EXPECT_FLOAT_EQ(output_val, 100.0f)
                << "counter should be 100 (read from max_ch config param channel)";
            found_output = true;
            break;
        }
    }
    EXPECT_TRUE(
        found_output
    ) << "counter_ch should have been written with the counter value";

    task->stop("test_stop", true);
}

TEST(ArcEdgeCases, DoubleStart) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto input_idx_name = make_unique_channel_name("double_start_input_idx");
    auto input_name = make_unique_channel_name("double_start_input");
    auto output_idx_name = make_unique_channel_name("double_start_output_idx");
    auto output_name = make_unique_channel_name("double_start_output");

    auto input_idx = synnax::channel::Channel(
        input_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(input_idx));
    auto output_idx = synnax::channel::Channel(
        output_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(output_idx));

    auto input_ch = synnax::channel::Channel(
        input_name,
        x::telem::FLOAT32_T,
        input_idx.key,
        false
    );
    auto output_ch = synnax::channel::Channel(
        output_name,
        x::telem::FLOAT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));
    ASSERT_NIL(client->channels.create(output_ch));

    synnax::arc::Arc arc_prog(make_unique_channel_name("double_start_test"));
    arc_prog.text = ::arc::text::Text(
        "func pass(val f32) f32 { return val }\n" + input_name + " -> pass{} -> " +
        output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_double_start_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_double_start_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();
    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();

    x::telem::Frame fr(2);
    auto now = x::telem::TimeStamp::now();
    auto idx = x::telem::Series(now);
    idx.alignment = x::telem::Alignment(1, 0);
    auto val = x::telem::Series(5.0f);
    val.alignment = x::telem::Alignment(1, 0);
    fr.emplace(input_idx.key, std::move(idx));
    fr.emplace(input_ch.key, std::move(val));
    input_frames->push_back(std::move(fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("first_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    task->start("second_start");

    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

    task->stop("final_stop", true);
}

TEST(ArcTests, testRestartResetsState) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto input_idx_name = make_unique_channel_name("restart_input_idx");
    auto input_name = make_unique_channel_name("restart_input");
    auto output_idx_name = make_unique_channel_name("restart_output_idx");
    auto output_name = make_unique_channel_name("restart_output");

    auto input_idx = synnax::channel::Channel(
        input_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(input_idx));
    auto output_idx = synnax::channel::Channel(
        output_idx_name,
        x::telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_NIL(client->channels.create(output_idx));

    auto input_ch = synnax::channel::Channel(
        input_name,
        x::telem::INT64_T,
        input_idx.key,
        false
    );
    auto output_ch = synnax::channel::Channel(
        output_name,
        x::telem::INT64_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));
    ASSERT_NIL(client->channels.create(output_ch));

    synnax::arc::Arc arc_prog(make_unique_channel_name("restart_test"));
    arc_prog.text = ::arc::text::Text(
        "func counter(trigger i64) i64 {\n"
        "    count $= 0\n"
        "    count = count + trigger\n"
        "    return count\n"
        "}\n" +
        input_name + " -> counter{} -> " + output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_restart_test_rack"))
    );

    synnax::task::Task task_meta(rack.key, "arc_restart_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = x::json::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    auto input_frames = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame input_fr(2);
    auto now = x::telem::TimeStamp::now();
    auto input_idx_series = x::telem::Series(now);
    input_idx_series.alignment = x::telem::Alignment(1, 0);
    auto input_val_series = x::telem::Series(static_cast<int64_t>(1));
    input_val_series.alignment = x::telem::Alignment(1, 0);
    input_fr.emplace(input_idx.key, std::move(input_idx_series));
    input_fr.emplace(input_ch.key, std::move(input_val_series));
    input_frames->push_back(std::move(input_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = ASSERT_NIL_P(
        arc::Task::create(task_meta, ctx, task_cfg, mock_writer, mock_streamer)
    );

    task->start("test_start_1");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

    auto &output_fr_1 = mock_writer->writes->at(0);
    ASSERT_TRUE(output_fr_1.contains(output_ch.key));
    auto output_val_1 = output_fr_1.at<int64_t>(output_ch.key, 0);
    EXPECT_EQ(output_val_1, 1);

    task->stop("test_stop_1", true);

    mock_writer->writes->clear();
    mock_writer->writer_opens = 0;
    ctx->statuses.clear();

    input_frames->clear();
    x::telem::Frame input_fr_2(2);
    auto now_2 = x::telem::TimeStamp::now();
    auto input_idx_series_2 = x::telem::Series(now_2);
    input_idx_series_2.alignment = x::telem::Alignment(2, 0);
    auto input_val_series_2 = x::telem::Series(static_cast<int64_t>(1));
    input_val_series_2.alignment = x::telem::Alignment(2, 0);
    input_fr_2.emplace(input_idx.key, std::move(input_idx_series_2));
    input_fr_2.emplace(input_ch.key, std::move(input_val_series_2));
    input_frames->push_back(std::move(input_fr_2));

    task->start("test_start_2");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

    auto &output_fr_2 = mock_writer->writes->at(0);
    ASSERT_TRUE(output_fr_2.contains(output_ch.key));
    auto output_val_2 = output_fr_2.at<int64_t>(output_ch.key, 0);
    EXPECT_EQ(output_val_2, 1) << "State should be reset on restart, count should be 1";

    task->stop("test_stop_2", true);
}
}
