// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/opc/writer.h"
#include "driver/opc/util.h"
#include "driver/config/config.h"
#include "driver/loop/loop.h"

///////////////////////////////////////////////////////////////////////////////////
//                                     WriterConfig                              //
///////////////////////////////////////////////////////////////////////////////////
opc::WriterConfig::WriterConfig(
    config::Parser &parser
) : device(parser.required<std::string>("device")) {
    parser.iter("channels", [&](config::Parser &channel_parser) {
        const auto ch = WriterChannelConfig(channel_parser);
        if (ch.enabled) channels.push_back(ch);
    });
}

///////////////////////////////////////////////////////////////////////////////////
//                                     Writer Task                               //
///////////////////////////////////////////////////////////////////////////////////
void opc::WriterTask::exec(task::Command &cmd) {
    if (cmd.type == "start") this->start();
    else if (cmd.type == "stop") return stop();
    else
        LOG(ERROR) << "unknown command type: " << cmd.type;
}

void opc::WriterTask::start() {
    freighter::Error conn_err = test_connection(this->ua_client,
                                                device_props.connection.endpoint);
    if (conn_err) {
        ctx->setState({
                          .task = task.key,
                          .variant = "error",
                          .details = json{
                              {"message", conn_err.message()}
                          }
                      });
        LOG(ERROR) << "[opc.writer] failed to connect to OPC UA server: "
                   << conn_err.message();
        return;
    }
    this->cmd_pipe.start();
    ctx->setState({
                      .task = task.key,
                      .variant = "success",
                      .details = json{
                          {"running", true},
                          {"message", "Task started successfully"}
                      }
                  });
}

void opc::WriterTask::stop() {
    ctx->setState({
                      .task = task.key,
                      .variant = "success",
                      .details = json{
                          {"running", false},
                          {"message", "Task stopped successfully"}
                      }
                  });
    this->cmd_pipe.stop();
}


std::unique_ptr<task::Task> opc::WriterTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto config_parser = config::Parser(task.config);
    auto cfg = WriterConfig(config_parser);
    if (!config_parser.ok()) {
        LOG(ERROR) << "[opc.writer] failed to parse configuration for " << task.name;
        ctx->setState({
                          .task = task.key,
                          .variant = "error",
                          .details = config_parser.error_json(),
                      });
        return nullptr;
    }

    auto [device, dev_err] = ctx->client->hardware.retrieveDevice(cfg.device);
    if (dev_err) {
        LOG(ERROR) << "[opc.writer] failed to retrieve device " << cfg.device <<
                   " error: " << dev_err.message();
        ctx->setState({
                          .task = task.key,
                          .variant = "error",
                          .details = json{
                              {"message", dev_err.message()}
                          }
                      });
        return nullptr;
    }

    auto properties_parser = config::Parser(device.properties);
    auto properties = DeviceProperties(properties_parser);

    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };
    auto breaker = breaker::Breaker(breaker_config);

    auto [ua_client, conn_err] = opc::connect(properties.connection,
                                              "[opc.writer.cmd] ");
    if (conn_err) {
        ctx->setState({
                          .variant = "error",
                          .details = json{{"message", conn_err.message()}}
                      });
        return nullptr;
    }

    auto sink = std::make_shared<opc::WriterSink>(
        cfg,
        ua_client,
        ctx,
        task,
        properties
    );


    auto cmd_streamer_config = synnax::StreamerConfig{
        .channels = cfg.cmd_keys(),
        .start = synnax::TimeStamp::now(),
    };

    ctx->setState({
                      .task = task.key,
                      .variant = "success",
                      .details = json{
                          {"running", false},
                          {"message", "Task configured successfully"}
                      }
                  });

    return std::make_unique<opc::WriterTask>(
        ctx,
        task,
        cfg,
        breaker_config,
        std::move(sink),
        cmd_streamer_config,
        ua_client,
        properties
    );
}
