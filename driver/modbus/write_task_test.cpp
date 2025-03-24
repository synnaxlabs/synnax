// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// module
#include "x/cpp/xtest/xtest.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"

/// internal
#include "driver/modbus/write_task.h"
#include "driver/modbus/mock/slave.h"
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

    void setupTaskConfig() {
        this->sy = std::make_shared<synnax::Synnax>(new_test_client());
        this->devs = std::make_shared<modbus::device::Manager>();
        this->ctx = std::make_shared<task::MockContext>(sy);
        if (this->coil_ch.name.empty()) this->coil_ch.name = "coil";
        if (this->coil_ch.data_type == telem::UNKNOWN_T) this->coil_ch.data_type = telem::UINT8_T;
        ASSERT_NIL(sy->channels.create(this->coil_ch));
        if (this->reg_ch.name.empty()) this->reg_ch.name = "register";
        if (this->reg_ch.data_type == telem::UNKNOWN_T) this->reg_ch.data_type = telem::UINT16_T;
        this->reg_ch.is_virtual = true;
        ASSERT_NIL(sy->channels.create(this->reg_ch));
        auto rack = ASSERT_NIL_P(sy->hardware.create_rack("test_rack"));
        json properties{
            {
                "connection", {
                    {"host", "127.0.0.1"},
                    {"port", 1502},
                    {"swap_bytes", false},
                    {"swap_words", false}
                }
            }
        };

        synnax::Device dev(
            "modbus_test_dev",
            "modbus_test_dev",
            rack.key,
            "dev1",
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
    this->setupTaskConfig();
    modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer stop_slave([&slave] { slave.stop(); });

    json task_cfg{
        {"device", "modbus_test_dev"},
        {
            "channels", json::array({
                {
                    {"type", "coil_output"},
                    {"address", 0},
                    {"enabled", true},
                    {"channel", coil_ch.key}
                },
                {
                    {"type", "holding_register_output"},
                    {"address", 1},
                    {"enabled", true},
                    {"channel", reg_ch.key},
                    {"data_type", "uint16"}
                }
            })
        }
    };

    auto p = xjson::Parser(task_cfg);
    cfg = std::make_unique<modbus::WriteTaskConfig>(sy, p);
    ASSERT_NIL(p.error());
    const auto reads = std::make_shared<std::vector<synnax::Frame> >();
    synnax::Frame fr(2); // Pre-allocate for 2 series (coil and register)
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

// TEST_F(ModbusWriteTest, testFloat32Write) {
//     this->setupTaskConfig();
//     modbus::mock::SlaveConfig slave_cfg;
//     slave_cfg.host = "127.0.0.1";
//     slave_cfg.port = 1502;
//
//     auto slave = modbus::mock::Slave(slave_cfg);
//     ASSERT_NIL(slave.start());
//     x::defer stop_slave([&slave] { slave.stop(); });
//
//
//     json task_cfg{
//         {"device", "modbus_test_dev"},
//         {
//             "channels", json::array({
//                 {
//                     {"type", "holding_register_output"},
//                     {"address", 1},
//                     {"enabled", true},
//                     {"channel", reg_ch.key},
//                     {"data_type", "float32"}
//                 }
//             })
//         }
//     };
// }

TEST_F(ModbusWriteTest, testInvalidChannelType) {
    const json task_cfg{
        {"device", "modbus_test_dev"},
        {
            "channels", json::array({
                {
                    {"type", "invalid_type"},
                    {"address", 0},
                    {"enabled", true},
                    {"channel", coil_ch.key}
                }
            })
        }
    };

    auto p = xjson::Parser(task_cfg);
    auto invalid_cfg = std::make_unique<modbus::WriteTaskConfig>(sy, p);
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}
