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

#include "driver/modbus/mock/slave.h"
#include "driver/modbus/write_task.h"
#include "driver/pipeline/mock/pipeline.h"

class ModbusWriteTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> sy;
    synnax::Task task;
    std::unique_ptr<modbus::WriteTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::StreamerFactory> mock_streamer_factory;
    std::shared_ptr<modbus::device::Manager> devs;
    synnax::Channel coil_ch;
    synnax::Channel reg_ch;

    void setup_task_config() {
        this->sy = std::make_shared<synnax::Synnax>(new_test_client());
        this->devs = std::make_shared<modbus::device::Manager>();
        this->ctx = std::make_shared<task::MockContext>(sy);
        if (this->coil_ch.name.empty()) this->coil_ch.name = "coil";
        if (this->coil_ch.data_type == telem::UNKNOWN_T)
            this->coil_ch.data_type = telem::UINT8_T;
        this->coil_ch.is_virtual = true;
        ASSERT_NIL(sy->channels.create(this->coil_ch));
        if (this->reg_ch.name.empty()) this->reg_ch.name = "register";
        if (this->reg_ch.data_type == telem::UNKNOWN_T)
            this->reg_ch.data_type = telem::UINT16_T;
        this->reg_ch.is_virtual = true;
        ASSERT_NIL(sy->channels.create(this->reg_ch));
        auto rack = ASSERT_NIL_P(sy->hardware.create_rack("test_rack"));
        json properties{
            {"connection",
             {{"host", "127.0.0.1"},
              {"port", 1502},
              {"swap_bytes", false},
              {"swap_words", false}}}
        };

        synnax::Device dev(
            "modbus_test_dev",
            "modbus_test_dev",
            rack.key,
            "dev1",
            "modbus",
            "Modbus Device",
            nlohmann::to_string(properties)
        );
        ASSERT_NIL(sy->hardware.create_device(dev));

        task = synnax::Task(rack.key, "modbus_write_test", "modbus_write", "");
    }
};

TEST_F(ModbusWriteTest, testBasicWrite) {
    this->setup_task_config();
    modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer stop_slave([&slave] { slave.stop(); });

    json task_cfg{
        {"device", "modbus_test_dev"},
        {"channels",
         json::array(
             {{{"type", "coil_output"},
               {"address", 0},
               {"enabled", true},
               {"channel", coil_ch.key}},
              {{"type", "holding_register_output"},
               {"address", 1},
               {"enabled", true},
               {"channel", reg_ch.key},
               {"data_type", "uint16"}}}
         )}
    };

    auto p = xjson::Parser(task_cfg);
    cfg = std::make_unique<modbus::WriteTaskConfig>(sy, p);
    ASSERT_NIL(p.error());
    const auto reads = std::make_shared<std::vector<synnax::Frame>>();
    synnax::Frame fr(2);
    fr.emplace(coil_ch.key, telem::Series(static_cast<uint8_t>(1)));
    fr.emplace(reg_ch.key, telem::Series(static_cast<uint16_t>(12345)));
    reads->push_back(std::move(fr));

    mock_streamer_factory = pipeline::mock::simple_streamer_factory(
        {coil_ch.key, reg_ch.key},
        reads
    );

    auto dev = ASSERT_NIL_P(devs->acquire(cfg->conn));

    auto wt = std::make_unique<common::WriteTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<modbus::WriteTaskSink>(dev, std::move(*cfg)),
        nullptr,
        mock_streamer_factory
    );

    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_streamer_factory->streamer_opens, 1);
    ASSERT_EVENTUALLY_EQ(slave.get_coil(0), 1);
    ASSERT_EVENTUALLY_EQ(slave.get_holding_register(1), 12345);
    wt->stop("stop_cmd", true);
}

TEST_F(ModbusWriteTest, testMultipleDataTypes) {
    this->setup_task_config();
    modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer stop_slave([&slave] { slave.stop(); });

    auto int16_ch = ASSERT_NIL_P(sy->channels.create("int16", telem::INT16_T, true));
    auto uint32_ch = ASSERT_NIL_P(sy->channels.create("uint32", telem::UINT32_T, true));
    auto int32_ch = ASSERT_NIL_P(sy->channels.create("int32", telem::INT32_T, true));
    auto float32_ch = ASSERT_NIL_P(
        sy->channels.create("float32", telem::FLOAT32_T, true)
    );
    auto float64_ch = ASSERT_NIL_P(
        sy->channels.create("float64", telem::FLOAT64_T, true)
    );

    json task_cfg{
        {"device", "modbus_test_dev"},
        {"channels",
         json::array(
             {{{"type", "holding_register_output"},
               {"address", 0},
               {"enabled", true},
               {"channel", int16_ch.key},
               {"data_type", "int16"}},
              {{"type", "holding_register_output"},
               {"address", 1},
               {"enabled", true},
               {"channel", uint32_ch.key},
               {"data_type", "uint32"}},
              {{"type", "holding_register_output"},
               {"address", 3},
               {"enabled", true},
               {"channel", int32_ch.key},
               {"data_type", "int32"}},
              {{"type", "holding_register_output"},
               {"address", 5},
               {"enabled", true},
               {"channel", float32_ch.key},
               {"data_type", "float32"}},
              {{"type", "holding_register_output"},
               {"address", 7},
               {"enabled", true},
               {"channel", float64_ch.key},
               {"data_type", "float64"}}}
         )}
    };

    auto p = xjson::Parser(task_cfg);
    cfg = std::make_unique<modbus::WriteTaskConfig>(sy, p);
    ASSERT_NIL(p.error());

    const auto reads = std::make_shared<std::vector<synnax::Frame>>();
    synnax::Frame fr(5);
    fr.emplace(int16_ch.key, telem::Series(static_cast<int16_t>(-1234)));
    fr.emplace(uint32_ch.key, telem::Series(static_cast<uint32_t>(0xDEADBEEF)));
    fr.emplace(int32_ch.key, telem::Series(static_cast<int32_t>(-2147483648)));
    fr.emplace(float32_ch.key, telem::Series(static_cast<float>(3.14159f)));
    fr.emplace(float64_ch.key, telem::Series(static_cast<double>(2.71828)));
    reads->push_back(std::move(fr));

    mock_streamer_factory = pipeline::mock::simple_streamer_factory(
        {int16_ch.key, uint32_ch.key, int32_ch.key, float32_ch.key, float64_ch.key},
        reads
    );

    auto dev = ASSERT_NIL_P(devs->acquire(cfg->conn));

    auto wt = std::make_unique<common::WriteTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<modbus::WriteTaskSink>(dev, std::move(*cfg)),
        nullptr,
        mock_streamer_factory
    );

    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_streamer_factory->streamer_opens, 1);
    ASSERT_EVENTUALLY_EQ(slave.get_holding_register(0), static_cast<uint16_t>(-1234));
    uint32_t uint32_val = (static_cast<uint32_t>(slave.get_holding_register(2)) << 16) |
                          slave.get_holding_register(1);
    ASSERT_EVENTUALLY_EQ(uint32_val, 0xDEADBEEF);
    wt->stop("stop_cmd", true);
}

TEST_F(ModbusWriteTest, testInvalidWriteConfiguration) {
    this->setup_task_config();

    json task_cfg{
        {"device", "non_existent_device"},
        {"channels",
         json::array(
             {{{"type", "coil_output"},
               {"address", 0},
               {"enabled", true},
               {"channel", coil_ch.key}}}
         )}
    };

    auto p1 = xjson::Parser(task_cfg);
    cfg = std::make_unique<modbus::WriteTaskConfig>(sy, p1);
    ASSERT_OCCURRED_AS(p1.error(), xerrors::VALIDATION);

    task_cfg["device"] = "modbus_test_dev";
    task_cfg["channels"][0]["type"] = "invalid_type";
    auto p2 = xjson::Parser(task_cfg);
    cfg = std::make_unique<modbus::WriteTaskConfig>(sy, p2);
    ASSERT_OCCURRED_AS(p2.error(), xerrors::VALIDATION);

    task_cfg["channels"][0]["type"] = "coil_output";
    task_cfg["channels"][0].erase("channel");
    auto p3 = xjson::Parser(task_cfg);
    cfg = std::make_unique<modbus::WriteTaskConfig>(sy, p3);
    ASSERT_OCCURRED_AS(p3.error(), xerrors::VALIDATION);

    task_cfg["channels"][0]["channel"] = reg_ch.key;
    task_cfg["channels"][0]["type"] = "holding_register_output";
    task_cfg["channels"][0].erase("data_type");
    auto p4 = xjson::Parser(task_cfg);
    cfg = std::make_unique<modbus::WriteTaskConfig>(sy, p4);
    ASSERT_OCCURRED_AS(p4.error(), xerrors::VALIDATION);
}

TEST_F(ModbusWriteTest, testConcurrentWrites) {
    this->setup_task_config();
    modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer stop_slave([&slave] { slave.stop(); });

    auto coil1 = ASSERT_NIL_P(sy->channels.create("coil1", telem::UINT8_T, true));
    auto coil2 = ASSERT_NIL_P(sy->channels.create("coil2", telem::UINT8_T, true));
    auto reg1 = ASSERT_NIL_P(sy->channels.create("reg1", telem::UINT16_T, true));
    auto reg2 = ASSERT_NIL_P(sy->channels.create("reg2", telem::UINT16_T, true));

    json task_cfg{
        {"device", "modbus_test_dev"},
        {"channels",
         json::array(
             {{{"type", "coil_output"},
               {"address", 0},
               {"enabled", true},
               {"channel", coil1.key}},
              {{"type", "coil_output"},
               {"address", 1},
               {"enabled", true},
               {"channel", coil2.key}},
              {{"type", "holding_register_output"},
               {"address", 0},
               {"enabled", true},
               {"channel", reg1.key},
               {"data_type", "uint16"}},
              {{"type", "holding_register_output"},
               {"address", 1},
               {"enabled", true},
               {"channel", reg2.key},
               {"data_type", "uint16"}}}
         )}
    };

    auto p = xjson::Parser(task_cfg);
    cfg = std::make_unique<modbus::WriteTaskConfig>(sy, p);
    ASSERT_NIL(p.error());

    const auto reads = std::make_shared<std::vector<synnax::Frame>>();
    synnax::Frame fr(4);
    fr.emplace(coil1.key, telem::Series(static_cast<uint8_t>(1)));
    fr.emplace(coil2.key, telem::Series(static_cast<uint8_t>(0)));
    fr.emplace(reg1.key, telem::Series(static_cast<uint16_t>(1000)));
    fr.emplace(reg2.key, telem::Series(static_cast<uint16_t>(2000)));
    reads->push_back(std::move(fr));

    mock_streamer_factory = pipeline::mock::simple_streamer_factory(
        {coil1.key, coil2.key, reg1.key, reg2.key},
        reads
    );

    auto dev = ASSERT_NIL_P(devs->acquire(cfg->conn));

    auto wt = std::make_unique<common::WriteTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<modbus::WriteTaskSink>(dev, std::move(*cfg)),
        nullptr,
        mock_streamer_factory
    );

    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_streamer_factory->streamer_opens, 1);
    ASSERT_EVENTUALLY_EQ(slave.get_coil(0), 1);
    ASSERT_EVENTUALLY_EQ(slave.get_coil(1), 0);
    ASSERT_EVENTUALLY_EQ(slave.get_holding_register(0), 1000);
    ASSERT_EVENTUALLY_EQ(slave.get_holding_register(1), 2000);
    wt->stop("stop_cmd", true);
}

TEST_F(ModbusWriteTest, testWriteVerification) {
    this->setup_task_config();
    modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;
    slave_cfg.coils[0] = 0;
    slave_cfg.holding_registers[0] = 0;
    slave_cfg.holding_registers[1] = 0;

    auto slave = modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer stop_slave([&slave] { slave.stop(); });

    json task_cfg{
        {"device", "modbus_test_dev"},
        {"channels",
         json::array(
             {{{"type", "coil_output"},
               {"address", 0},
               {"enabled", true},
               {"channel", coil_ch.key}},
              {{"type", "holding_register_output"},
               {"address", 1},
               {"enabled", true},
               {"channel", reg_ch.key},
               {"data_type", "uint16"}}}
         )}
    };

    auto p = xjson::Parser(task_cfg);
    cfg = std::make_unique<modbus::WriteTaskConfig>(sy, p);
    ASSERT_NIL(p.error());

    const auto reads = std::make_shared<std::vector<synnax::Frame>>();
    synnax::Frame fr(2);
    fr.emplace(coil_ch.key, telem::Series(static_cast<uint8_t>(1)));
    fr.emplace(reg_ch.key, telem::Series(static_cast<uint16_t>(42)));
    reads->push_back(std::move(fr));

    mock_streamer_factory = pipeline::mock::simple_streamer_factory(
        {coil_ch.key, reg_ch.key},
        reads
    );

    auto dev = ASSERT_NIL_P(devs->acquire(cfg->conn));

    auto wt = std::make_unique<common::WriteTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<modbus::WriteTaskSink>(dev, std::move(*cfg)),
        nullptr,
        mock_streamer_factory
    );

    ASSERT_EQ(slave.get_coil(0), 0);
    ASSERT_EQ(slave.get_holding_register(1), 0);

    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_streamer_factory->streamer_opens, 1);

    ASSERT_EVENTUALLY_EQ(slave.get_coil(0), 1);
    ASSERT_EVENTUALLY_EQ(slave.get_holding_register(1), 42);

    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto first_state = ctx->states[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.variant, "success");

    wt->stop("stop_cmd", true);

    ASSERT_EQ(ctx->states.size(), 2);
    const auto second_state = ctx->states[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.variant, "success");
}
