// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


void opc::WriterTask::exec(task::Command &cmd) {
    if (cmd.type == "start") {
        this->start();
    } else if (cmd.type == "stop") return stop();
    else
        LOG(ERROR) << "unknown command type: " << cmd.type;
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
    pipe.stop(); 
}

void opc::WriterTask::start(){
    // first try to check for timeout
    UA_StatusCode status = UA_Client_connect(this->ua_client.get(), device_props.connection.endpoint.c_str());
    if (status != UA_STATUSCODE_GOOD) {
        // attempt again to reestablish if timed out
        UA_StatusCode status_retry = UA_Client_connect(this->ua_client.get(), device_props.connection.endpoint.c_str());
        if(status_retry != UA_STATUSCODE_GOOD){
            ctx->setState({
                .task = task.key,
                .variant = "error",
                .details = json{
                    {"message", "Failed to connect to OPC UA server: " + std::string(
                        UA_StatusCode_name(status))}
                }
            });
            LOG(ERROR) << "[opc.writer] connection failed: " << UA_StatusCode_name(status);
        }
    }
    VLOG(1) << "[opc.writer] Connection Established";
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



std::unique_ptr<task::Task> opc::WriterTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    VLOG(2) << "[opc.writer] configuring task " << task.name;
    auto config_parser = config::Parser(task.config);
    auto cfg = ReaderConfig(config_parser);
    if (!config_parser.ok()) {
        LOG(ERROR) << "[opc.writer] failed to parse configuration for " << task.name;
        ctx->setState({
            .task = task.key,
            .variant = "error",
            .details = config_parser.error_json(),
        });
        return nullptr;
    }
    VLOG(2) << "[opc.writer] successfully parsed configuration for " << task.name;
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
    auto [channelKeys, indexes] = res;

    // Connect to the OPC UA server.
    auto [ua_client, conn_err] = opc::connect(properties.connection, "[opc.writer] ");
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

    auto source = std::make_shared<ReaderSource>(
        cfg,
        ua_client,
        indexes,
        ctx,
        task
    );

    auto writer_cfg = synnax::WriterConfig{
        .channels = channelKeys,
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
    return std::make_unique<Reader>( 
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