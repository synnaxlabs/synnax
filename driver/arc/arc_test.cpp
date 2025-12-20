// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/xtest/xtest.h"

#include "driver/arc/arc.h"
#include "driver/arc/task.h"
#include "driver/pipeline/mock/pipeline.h"
#include "driver/task/task.h"

TEST(ArcTests, testCalcDoubling) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto input_idx_name = make_unique_channel_name("ox_pt_1_idx");
    auto input_name = make_unique_channel_name("ox_pt_1");
    auto output_idx_name = make_unique_channel_name("ox_pt_doubled_idx");
    auto output_name = make_unique_channel_name("ox_pt_doubled");

    auto input_idx = synnax::Channel(input_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client->channels.create(input_idx));
    auto output_idx = synnax::Channel(output_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client->channels.create(output_idx));

    auto input_ch = synnax::Channel(input_name, telem::FLOAT32_T, input_idx.key, false);
    auto output_ch = synnax::Channel(
        output_name,
        telem::FLOAT32_T,
        output_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(input_ch));
    ASSERT_NIL(client->channels.create(output_ch));

    synnax::Arc arc_prog(make_unique_channel_name("calc_test"));
    arc_prog.text = arc::text::Text(
        "func calc(val f32) f32 {\n"
        "    return val * 2\n"
        "}\n" +
        input_name + " -> calc{} -> " + output_name + "\n"
    );
    ASSERT_NIL(client->arcs.create(arc_prog));

    auto rack = ASSERT_NIL_P(
        client->racks.create(make_unique_channel_name("arc_test_rack"))
    );

    synnax::Task task_meta(rack.key, "arc_calc_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = xjson::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto runtime = ASSERT_NIL_P(arc::load_runtime(task_cfg, client));

    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    auto input_frames = std::make_shared<std::vector<telem::Frame>>();
    telem::Frame input_fr(2);
    auto now = telem::TimeStamp::now();
    input_fr.emplace(input_idx.key, telem::Series(std::vector<telem::TimeStamp>{now}));
    input_fr.emplace(input_ch.key, telem::Series(std::vector<float>{5.0f}));
    input_frames->push_back(std::move(input_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {input_idx.key, input_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = std::make_unique<arc::Task>(
        task_meta,
        ctx,
        runtime,
        task_cfg,
        mock_writer,
        mock_streamer
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
    auto start_cmd_idx = synnax::Channel(start_cmd_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client->channels.create(start_cmd_idx));
    auto start_cmd_ch = synnax::Channel(
        start_cmd_name,
        telem::UINT8_T,
        start_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(start_cmd_ch));

    // Create output channel (valve_cmd)
    auto valve_cmd_idx_name = make_unique_channel_name("valve_cmd_idx");
    auto valve_cmd_name = make_unique_channel_name("valve_cmd");
    auto valve_cmd_idx = synnax::Channel(valve_cmd_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client->channels.create(valve_cmd_idx));
    auto valve_cmd_ch = synnax::Channel(
        valve_cmd_name,
        telem::INT64_T,
        valve_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(valve_cmd_ch));

    // Create Arc program with the sequence
    synnax::Arc arc_prog(make_unique_channel_name("sequence_test"));
    arc_prog.text = arc::text::Text(
        "sequence main {\n"
        "    stage run {\n"
        "        1 -> " + valve_cmd_name + "\n"
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

    synnax::Task task_meta(rack.key, "arc_sequence_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = xjson::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto runtime = ASSERT_NIL_P(arc::load_runtime(task_cfg, client));

    // Setup mock writer to capture outputs
    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    // Setup mock streamer to send trigger frame
    auto input_frames = std::make_shared<std::vector<telem::Frame>>();
    telem::Frame trigger_fr(2);
    auto now = telem::TimeStamp::now();
    trigger_fr.emplace(start_cmd_idx.key, telem::Series(std::vector<telem::TimeStamp>{now}));
    trigger_fr.emplace(start_cmd_ch.key, telem::Series(std::vector<uint8_t>{1}));
    input_frames->push_back(std::move(trigger_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {start_cmd_idx.key, start_cmd_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = std::make_unique<arc::Task>(
        task_meta,
        ctx,
        runtime,
        task_cfg,
        mock_writer,
        mock_streamer
    );

    task->start("test_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    // Wait for the sequence to execute and write to valve_cmd
    ASSERT_EVENTUALLY_GE(mock_writer->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 1);

    // Verify valve_cmd received the value 1
    bool found_valve_cmd = false;
    for (const auto &output_fr : *mock_writer->writes) {
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
    auto start_cmd_idx = synnax::Channel(start_cmd_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client->channels.create(start_cmd_idx));
    auto start_cmd_ch = synnax::Channel(
        start_cmd_name,
        telem::UINT8_T,
        start_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(start_cmd_ch));

    // Create output channel (valve_cmd)
    auto valve_cmd_idx_name = make_unique_channel_name("truthiness_valve_cmd_idx");
    auto valve_cmd_name = make_unique_channel_name("truthiness_valve_cmd");
    auto valve_cmd_idx = synnax::Channel(valve_cmd_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client->channels.create(valve_cmd_idx));
    auto valve_cmd_ch = synnax::Channel(
        valve_cmd_name,
        telem::INT64_T,
        valve_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(valve_cmd_ch));

    // Create Arc program with a sequence triggered by one-shot edge
    synnax::Arc arc_prog(make_unique_channel_name("truthiness_test"));
    arc_prog.text = arc::text::Text(
        "sequence main {\n"
        "    stage run {\n"
        "        42 -> " + valve_cmd_name + "\n"
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

    synnax::Task task_meta(rack.key, "arc_truthiness_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = xjson::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto runtime = ASSERT_NIL_P(arc::load_runtime(task_cfg, client));

    // Setup mock writer to capture outputs
    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    // Setup mock streamer to send TWO frames:
    // 1. First frame with value 0 (should NOT trigger)
    // 2. Second frame with value 1 (should trigger)
    auto input_frames = std::make_shared<std::vector<telem::Frame>>();

    // Frame 1: falsy trigger (0) - should NOT trigger the sequence
    telem::Frame falsy_trigger_fr(2);
    auto now = telem::TimeStamp::now();
    auto falsy_idx_series = telem::Series(std::vector<telem::TimeStamp>{now});
    falsy_idx_series.alignment = telem::Alignment(1, 0);
    auto falsy_val_series = telem::Series(std::vector<uint8_t>{0});
    falsy_val_series.alignment = telem::Alignment(1, 0);
    falsy_trigger_fr.emplace(start_cmd_idx.key, std::move(falsy_idx_series));
    falsy_trigger_fr.emplace(start_cmd_ch.key, std::move(falsy_val_series));
    input_frames->push_back(std::move(falsy_trigger_fr));

    // Frame 2: truthy trigger (1) - should trigger the sequence
    telem::Frame truthy_trigger_fr(2);
    auto later = telem::TimeStamp::now() + telem::SECOND;
    auto truthy_idx_series = telem::Series(std::vector<telem::TimeStamp>{later});
    truthy_idx_series.alignment = telem::Alignment(1, 1);
    auto truthy_val_series = telem::Series(std::vector<uint8_t>{1});
    truthy_val_series.alignment = telem::Alignment(1, 1);
    truthy_trigger_fr.emplace(start_cmd_idx.key, std::move(truthy_idx_series));
    truthy_trigger_fr.emplace(start_cmd_ch.key, std::move(truthy_val_series));
    input_frames->push_back(std::move(truthy_trigger_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {start_cmd_idx.key, start_cmd_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = std::make_unique<arc::Task>(
        task_meta,
        ctx,
        runtime,
        task_cfg,
        mock_writer,
        mock_streamer
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
    for (const auto &output_fr : *mock_writer->writes) {
        if (output_fr.contains(valve_cmd_ch.key)) {
            auto output_val = output_fr.at<int64_t>(valve_cmd_ch.key, 0);
            EXPECT_EQ(output_val, 42);
            found_valve_cmd = true;
            break;
        }
    }
    EXPECT_TRUE(found_valve_cmd) << "valve_cmd channel was not written to - "
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
    auto start_cmd_idx = synnax::Channel(start_cmd_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client->channels.create(start_cmd_idx));
    auto start_cmd_ch = synnax::Channel(
        start_cmd_name,
        telem::UINT8_T,
        start_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(start_cmd_ch));

    // Create pressure sensor channel
    auto pressure_idx_name = make_unique_channel_name("two_stage_pressure_idx");
    auto pressure_name = make_unique_channel_name("two_stage_pressure");
    auto pressure_idx = synnax::Channel(pressure_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client->channels.create(pressure_idx));
    auto pressure_ch = synnax::Channel(
        pressure_name,
        telem::FLOAT32_T,
        pressure_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(pressure_ch));

    // Create output channel (valve_cmd)
    auto valve_cmd_idx_name = make_unique_channel_name("two_stage_valve_cmd_idx");
    auto valve_cmd_name = make_unique_channel_name("two_stage_valve_cmd");
    auto valve_cmd_idx = synnax::Channel(valve_cmd_idx_name, telem::TIMESTAMP_T, 0, true);
    ASSERT_NIL(client->channels.create(valve_cmd_idx));
    auto valve_cmd_ch = synnax::Channel(
        valve_cmd_name,
        telem::INT64_T,
        valve_cmd_idx.key,
        false
    );
    ASSERT_NIL(client->channels.create(valve_cmd_ch));

    // Create Arc program with a two-stage sequence
    // Stage "pressurize": outputs 1, transitions to "idle" when pressure > 50
    // Stage "idle": outputs 0 (terminal stage)
    synnax::Arc arc_prog(make_unique_channel_name("two_stage_test"));
    arc_prog.text = arc::text::Text(
        "sequence main {\n"
        "    stage pressurize {\n"
        "        1 -> " + valve_cmd_name + ",\n"
        "        " + pressure_name + " -> " + pressure_name + " > 50 => next\n"
        "    }\n"
        "    stage idle {\n"
        "        0 -> " + valve_cmd_name + "\n"
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

    synnax::Task task_meta(rack.key, "arc_two_stage_test", "arc_runtime", "");
    nlohmann::json cfg{{"arc_key", arc_prog.key}};
    task_meta.config = nlohmann::to_string(cfg);

    auto parser = xjson::Parser(task_meta.config);
    auto task_cfg = ASSERT_NIL_P(arc::TaskConfig::parse(client, parser));

    auto runtime = ASSERT_NIL_P(arc::load_runtime(task_cfg, client));

    // Setup mock writer to capture outputs
    auto mock_writer = std::make_shared<pipeline::mock::WriterFactory>();

    // Setup mock streamer to send frames:
    // 1. Trigger frame to start the sequence
    // 2. Pressure frame with value < 50 (should stay in pressurize stage)
    // 3. Pressure frame with value > 50 (should transition to idle stage)
    auto input_frames = std::make_shared<std::vector<telem::Frame>>();

    // Frame 1: Trigger the sequence
    telem::Frame trigger_fr(4);
    auto now = telem::TimeStamp::now();
    trigger_fr.emplace(start_cmd_idx.key, telem::Series(std::vector<telem::TimeStamp>{now}));
    trigger_fr.emplace(start_cmd_ch.key, telem::Series(std::vector<uint8_t>{1}));
    trigger_fr.emplace(pressure_idx.key, telem::Series(std::vector<telem::TimeStamp>{now}));
    trigger_fr.emplace(pressure_ch.key, telem::Series(std::vector<float>{10.0f}));
    input_frames->push_back(std::move(trigger_fr));

    // Frame 2: Pressure still low - should stay in pressurize, output 1
    telem::Frame low_pressure_fr(2);
    auto t2 = now + telem::MILLISECOND * 100;
    auto low_pressure_idx_series = telem::Series(std::vector<telem::TimeStamp>{t2});
    low_pressure_idx_series.alignment = telem::Alignment(1, 1);
    auto low_pressure_val_series = telem::Series(std::vector<float>{30.0f});
    low_pressure_val_series.alignment = telem::Alignment(1, 1);
    low_pressure_fr.emplace(pressure_idx.key, std::move(low_pressure_idx_series));
    low_pressure_fr.emplace(pressure_ch.key, std::move(low_pressure_val_series));
    input_frames->push_back(std::move(low_pressure_fr));

    // Frame 3: Pressure exceeds threshold - should transition to idle, output 0
    telem::Frame high_pressure_fr(2);
    auto t3 = now + telem::MILLISECOND * 200;
    auto high_pressure_idx_series = telem::Series(std::vector<telem::TimeStamp>{t3});
    high_pressure_idx_series.alignment = telem::Alignment(1, 2);
    auto high_pressure_val_series = telem::Series(std::vector<float>{60.0f});
    high_pressure_val_series.alignment = telem::Alignment(1, 2);
    high_pressure_fr.emplace(pressure_idx.key, std::move(high_pressure_idx_series));
    high_pressure_fr.emplace(pressure_ch.key, std::move(high_pressure_val_series));
    input_frames->push_back(std::move(high_pressure_fr));

    auto mock_streamer = pipeline::mock::simple_streamer_factory(
        {start_cmd_idx.key, start_cmd_ch.key, pressure_idx.key, pressure_ch.key},
        input_frames
    );

    auto ctx = std::make_shared<task::MockContext>(client);

    auto task = std::make_unique<arc::Task>(
        task_meta,
        ctx,
        runtime,
        task_cfg,
        mock_writer,
        mock_streamer
    );

    task->start("test_start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    // Wait for multiple writes (at least 2: one from pressurize stage, one from idle stage)
    ASSERT_EVENTUALLY_GE(mock_writer->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer->writes->size(), 2);

    // Verify we got both valve states:
    // - At least one write with value 1 (from pressurize stage)
    // - At least one write with value 0 (from idle stage after transition)
    bool found_pressurize_output = false;
    bool found_idle_output = false;

    for (const auto &output_fr : *mock_writer->writes) {
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
