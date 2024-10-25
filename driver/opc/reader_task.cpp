// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>
#include <memory>
#include <utility>
#include "glog/logging.h"

#include "driver/opc/reader.h"
#include "driver/opc/util.h"
#include "driver/config/config.h"
#include "driver/errors/errors.h"
#include "driver/loop/loop.h"
#include "driver/pipeline/acquisition.h"

#include "include/open62541/types.h"
#include "include/open62541/types_generated.h"
#include "include/open62541/statuscodes.h"
#include "include/open62541/client_highlevel.h"
#include "include/open62541/common.h"


using namespace opc;
///////////////////////////////////////////////////////////////////////////////////
//                                     ReaderConfig                              //
///////////////////////////////////////////////////////////////////////////////////
ReaderConfig::ReaderConfig(
    config::Parser &parser
): device(parser.required<std::string>("device")),
   sample_rate(parser.required<std::float_t>("sample_rate")),
   stream_rate(parser.required<std::float_t>("stream_rate")),
   array_size(parser.optional<std::size_t>("array_size", 1)),
   data_saving(parser.optional<bool>("data_saving", true)) {
    if (stream_rate.value <= 0) stream_rate = Rate(1);
    parser.iter("channels", [&](config::Parser &channel_builder) {
        const auto ch = ReaderChannelConfig(channel_builder);
        if (ch.enabled) channels.push_back(ch);
    });
}

///////////////////////////////////////////////////////////////////////////////////
//                                     Helper Functions                          //
///////////////////////////////////////////////////////////////////////////////////
///@brief retrieves index channel information for given set of channels
std::pair<std::pair<std::vector<ChannelKey>, std::set<ChannelKey> >,
    freighter::Error> retrieveAdditionalChannelInfo(
    const std::shared_ptr<task::Context> &ctx,
    ReaderConfig &cfg,
    breaker::Breaker &breaker
) {
    auto channel_keys = cfg.channel_keys();
    if (channel_keys.empty()) return {{channel_keys, {}}, freighter::NIL};
    auto indexes = std::set<ChannelKey>();
    auto [channels, c_err] = ctx->client->channels.retrieve(cfg.channel_keys());
    if (c_err) {
        if (c_err.matches(freighter::UNREACHABLE) && breaker.wait(c_err.message()))
            return retrieveAdditionalChannelInfo(ctx, cfg, breaker);
        return {{channel_keys, indexes}, c_err};
    }
    for (auto i = 0; i < channels.size(); i++) {
        const auto ch = channels[i];
        if (std::count(channel_keys.begin(), channel_keys.end(), ch.index) == 0) {
            if (ch.index != 0) {
                channel_keys.push_back(ch.index);
                indexes.insert(ch.index);
            }
        }
        cfg.channels[i].ch = ch;
    }
    return {{channel_keys, indexes}, freighter::Error()};
}

///////////////////////////////////////////////////////////////////////////////////
//                                    Reader Task                                //
///////////////////////////////////////////////////////////////////////////////////
std::unique_ptr<task::Task> ReaderTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    VLOG(2) << "[opc.reader] configuring task " << task.name;
    auto config_parser = config::Parser(task.config);
    auto cfg = ReaderConfig(config_parser);
    if (!config_parser.ok()) {
        LOG(ERROR) << "[opc.reader] failed to parse configuration for " << task.name;
        ctx->setState({
            .task = task.key,
            .variant = "error",
            .details = config_parser.error_json(),
        });
        return nullptr;
    }
    VLOG(2) << "[opc.reader] successfully parsed configuration for " << task.name;
    auto [device, dev_err] = ctx->client->hardware.retrieveDevice(cfg.device);
    if (dev_err) {
        LOG(ERROR) << "[opc.reader] failed to retrieve device " << cfg.device <<
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
    // Fetch additional index channels we also need as part of the configuration.
    auto [res, err] = retrieveAdditionalChannelInfo(ctx, cfg, breaker);
    if (err) {
        ctx->setState({
            .task = task.key,
            .variant = "error",
            .details = json{{"message", err.message()}}
        });
        return nullptr;
    }
    auto [channel_keys, indexes] = res;

    auto [ua_client, conn_err] = opc::connect(properties.connection, "[opc.reader] ");
    if (conn_err) {
        ctx->setState({
            .task = task.key,
            .variant = "error",
            .details = json{{"message", conn_err.message()}}
        });
        return nullptr;
    }

    for (auto i = 0; i < cfg.channels.size(); i++) {
        auto ch = cfg.channels[i];
        UA_Variant *value = UA_Variant_new();
        const UA_StatusCode status = UA_Client_readValueAttribute(
            ua_client.get(),
            ch.node,
            value
        );
        if (status != UA_STATUSCODE_GOOD) {
            if (status == UA_STATUSCODE_BADNODEIDUNKNOWN) {
                config_parser.field_err("channels." + std::to_string(i),
                                        "opc node not found");
            } else {
                config_parser.field_err("channels." + std::to_string(i),
                                        "failed to read value" + std::string(
                                            UA_StatusCode_name(status)));
            }
            LOG(ERROR) << "failed to read value for channel " << ch.node_id;
        }
        UA_Variant_delete(value);
    }

    if (!config_parser.ok()) {
        ctx->setState({
            .task = task.key,
            .variant = "error",
            .details = config_parser.error_json(),
        });
        return nullptr;
    }

    auto source = std::make_shared<opc::ReaderSource>(
        cfg,
        ua_client,
        indexes,
        ctx,
        task
    );

    auto writer_cfg = synnax::WriterConfig{
        .channels = channel_keys,
        .start = TimeStamp::now(),
        .subject = synnax::ControlSubject{
            .name = task.name,
            .key = std::to_string(task.key)
        },
        .mode = cfg.data_saving
                    ? synnax::WriterMode::PersistStream
                    : synnax::WriterMode::StreamOnly,
        .enable_auto_commit = true
    };

    ctx->setState({
        .task = task.key,
        .variant = "success",
        .details = json{
            {"running", false},
            {"message", "Task configured successfully"}
        }
    });
    return std::make_unique<ReaderTask>(
        ctx,
        task,
        cfg,
        breaker_config,
        std::move(source),
        writer_cfg,
        ua_client,
        properties
    );
}

void ReaderTask::exec(task::Command &cmd) {
    if (cmd.type == "start") this->start();
    else if (cmd.type == "stop") return stop();
//    else
//        LOG(ERROR) << "unknown command type: " << cmd.type;
}

void ReaderTask::stop() {
    ctx->setState({
        .task = task.key,
        .variant = "success",
        .details = json{
            {"running", false},
            {"message", "Task stopped successfully"}
        }
    });
    pipe.stop();
}

void ReaderTask::start() {
    freighter::Error conn_err = refresh_connection(this->ua_client, device_props.connection.endpoint);
    if (conn_err) {
        ctx->setState({
            .task = task.key,
            .variant = "error",
            .details = json{
                {"message", conn_err.message()}
            }
        });
        LOG(ERROR) << "[opc.reader] connection failed: " << conn_err.message();
        return;
    }
    pipe.start();
    ctx->setState({
        .task = task.key,
        .variant = "success",
        .details = json{
            {"running", true},
            {"message", "Task started successfully"}
        }
    });
}
