// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xjson/xjson.h"
#include "driver/ni/read_task.h"

#include "client/cpp/testutil/testutil.h"


template<typename T>
class MockHardwareInterface final : public ni::HardwareInterface<T> {
    xerrors::Error start() const override { return xerrors::NIL; }
    xerrors::Error stop() const override { return xerrors::NIL; }

    xerrors::Error read(
        size_t samples_per_channel,
        std::vector<T> &data,
        size_t data_size
    ) override {
        return xerrors::NIL;
    }
};

TEST(ReadTaskTest, testAnalogReadConfigParse) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());

    auto [index_channel, idx_err] = sy->channels.create(
        "time_channel",
        telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_FALSE(idx_err) << idx_err;
    auto [data_channel , data_err] = sy->channels.create(
        "data_channel",
        telem::FLOAT64_T,
        index_channel.key,
        false
    );
    ASSERT_FALSE(data_err) << data_err;

    auto [rack, rack_err] = sy->hardware.create_rack("cat");
    ASSERT_FALSE(rack_err) << rack_err;

    synnax::Device dev(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    auto dev_err = sy->hardware.create_device(dev);
    ASSERT_FALSE(dev_err) << dev_err;

    json j{
        {"data_saving", false},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {
            "channels", json::array({
                {
                    {"type", "ai_accel"},
                    {"key", "ks1VnWdrSVA"},
                    {"port", 0},
                    {"enabled", true},
                    {"name", ""},
                    {"channel", data_channel.key},
                    {"terminal_config", "Cfg_Default"},
                    {"min_val", 0},
                    {"max_val", 1},
                    {"sensitivity", 0},
                    {"current_excit_source", "Internal"},
                    {"current_excit_val", 0},
                    {"custom_scale", {{"type", "none"}}},
                    {"units", "g"},
                    {"sensitivity_units", "mVoltsPerG"},
                    {"device", dev.key}
                }
            })
        }
    };
    auto parser = xjson::Parser(j);
    auto cfg = ni::ReadTaskConfig(sy, parser, "ni_analog_read");
    ASSERT_FALSE(parser.error()) << parser.error().message();
    EXPECT_EQ(cfg.channels.size(), 1);
    EXPECT_EQ(cfg.sample_rate, telem::Rate(25));
    EXPECT_EQ(cfg.stream_rate, telem::Rate(25));
    EXPECT_EQ(cfg.data_saving, false);
    EXPECT_EQ(cfg.indexes.size(), 1);
    EXPECT_EQ(cfg.indexes.count(index_channel.key), 1);
    EXPECT_EQ(cfg.buffer_size, telem::FLOAT64_T.density());
}

TEST(ReadTaskTest, testAnalogRead) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());

    auto [index_channel, idx_err] = sy->channels.create(
        "time_channel",
        telem::TIMESTAMP_T,
        0,
        true
    );
    ASSERT_FALSE(idx_err) << idx_err;
    auto [data_channel , data_err] = sy->channels.create(
        "data_channel",
        telem::FLOAT64_T,
        index_channel.key,
        false
    );
    ASSERT_FALSE(data_err) << data_err;

    auto [rack, rack_err] = sy->hardware.create_rack("cat");
    ASSERT_FALSE(rack_err) << rack_err;

    synnax::Device dev(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    auto dev_err = sy->hardware.create_device(dev);
    ASSERT_FALSE(dev_err) << dev_err;

    json j{
        {"data_saving", false},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {
            "channels", json::array({
                {
                    {"type", "ai_accel"},
                    {"key", "ks1VnWdrSVA"},
                    {"port", 0},
                    {"enabled", true},
                    {"name", ""},
                    {"channel", data_channel.key},
                    {"terminal_config", "Cfg_Default"},
                    {"min_val", 0},
                    {"max_val", 1},
                    {"sensitivity", 0},
                    {"current_excit_source", "Internal"},
                    {"current_excit_val", 0},
                    {"custom_scale", {{"type", "none"}}},
                    {"units", "g"},
                    {"sensitivity_units", "mVoltsPerG"},
                    {"device", dev.key}
                }
            })
        }
    };


    auto task = synnax::Task(
        rack.key,
        "my_task",
        "ni_analog_read",
        ""
    );

    auto p = xjson::Parser(j);
    auto cfg = ni::ReadTaskConfig(sy, p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto ctx = std::make_shared<task::MockContext>(client);

    auto rt = std::make_unique<ni::ReadTask<double> >(
        task,
        ctx,
        std::move(cfg),
        breaker::default_config(task.name),
        std::make_unique<MockHardwareInterface<double> >()
    );
    rt->start("");
    std::this_thread::sleep_for(std::chrono::seconds(5));
    rt->stop();
}
