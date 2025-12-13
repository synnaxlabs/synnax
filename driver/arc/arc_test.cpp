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

#include "driver/arc/arc.h"
#include "driver/pipeline/mock/pipeline.h"
#include "driver/task/task.h"
#include "x/cpp/xtest/xtest.h"

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
    auto output_ch = synnax::Channel(output_name, telem::FLOAT32_T, output_idx.key, false);
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

    auto rack = ASSERT_NIL_P(client->racks.create(make_unique_channel_name("arc_test_rack")));

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
        task_meta, ctx, runtime, task_cfg,
        mock_writer, mock_streamer
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
