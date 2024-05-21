// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>
#include "include/open62541/client_highlevel.h"
#include "glog/logging.h"
#include "driver/opc/reader.h"
#include "driver/opc/util.h"
#include "driver/config/config.h"
#include "driver/errors/errors.h"
#include "driver/loop/loop.h"
#include "include/open62541/client_config_default.h"
#include "include/open62541/client_subscriptions.h"
#include "include/open62541/types.h"
#include "include/open62541/plugin/log_stdout.h"
#include "driver/pipeline/acquisition.h"


using namespace opc;

ReaderConfig::ReaderConfig(
    config::Parser &parser
): device(parser.required<std::string>("device")) {
    sample_rate = Rate(parser.required<std::float_t>("sample_rate"));
    stream_rate = Rate(parser.required<std::float_t>("stream_rate"));
    parser.iter("channels", [&](config::Parser &channel_builder) {
        auto ch = ReaderChannelConfig(channel_builder);
        if (ch.enabled) channels.push_back(ch);
    });
}


std::pair<std::pair<std::vector<ChannelKey>, std::set<ChannelKey> >,
    freighter::Error> retrieveAdditionalChannelInfo(
    const std::shared_ptr<task::Context> &ctx,
    ReaderConfig &cfg,
    breaker::Breaker &breaker
) {
    auto channelKeys = cfg.channelKeys();
    if (channelKeys.empty()) return {{channelKeys, {}}, freighter::NIL};
    auto indexes = std::set<ChannelKey>();
    auto [channels, c_err] = ctx->client->channels.retrieve(cfg.channelKeys());
    if (c_err) {
        if (c_err.matches(freighter::UNREACHABLE) && breaker.wait(c_err.message()))
            return retrieveAdditionalChannelInfo(ctx, cfg, breaker);
        return {{channelKeys, indexes}, c_err};
    }
    for (auto i = 0; i < channels.size(); i++) {
        const auto ch = channels[i];
        if (std::count(channelKeys.begin(), channelKeys.end(), ch.index) == 0) {
            channelKeys.push_back(ch.index);
            indexes.insert(ch.index);
        }
        cfg.channels[i].ch = ch;
    }
    return {{channelKeys, indexes}, freighter::Error()};
}

class ReaderSource final : public pipeline::Source {
public:
    ReaderConfig cfg;
    std::shared_ptr<UA_Client> client;
    std::set<ChannelKey> indexes;
    UA_ReadRequest readRequest;
    std::vector<UA_ReadValueId> readValueIds;
    loop::Timer timer;
    size_t samples_per_read;
    synnax::Frame fr;

    ReaderSource(
        ReaderConfig cfg,
        const std::shared_ptr<UA_Client> &client,
        std::set<ChannelKey> indexes
    ) : cfg(std::move(cfg)), client(client), indexes(std::move(indexes)),
        timer(cfg.sample_rate), samples_per_read(
            static_cast<size_t>(cfg.sample_rate.value / cfg.stream_rate.value)
        ) {
        UA_ReadRequest_init(&readRequest);
        initializeReadRequest();
    }

    void allocateFrame() {
    }


    ~ReaderSource() {
        // UA_ReadRequest_clear(&readRequest);
    }

    void initializeReadRequest() {
        // Allocate and prepare readValueIds for enabled channels
        readValueIds.reserve(cfg.channels.size());
        for (const auto &ch: cfg.channels) {
            if (!ch.enabled) continue;
            UA_ReadValueId rvid;
            UA_ReadValueId_init(&rvid);
            rvid.nodeId = ch.node;
            rvid.attributeId = UA_ATTRIBUTEID_VALUE;
            readValueIds.push_back(rvid);
        }

        // Initialize the read request
        readRequest.nodesToRead = readValueIds.data();
        readRequest.nodesToReadSize = readValueIds.size();
    }

    freighter::Error start() override {
        return freighter::NIL;
    }

    freighter::Error stop() override {
        return freighter::NIL;
    }

    std::pair<Frame, freighter::Error> read() override {
        auto fr = Frame(cfg.channels.size() + indexes.size());
        size_t enabled_count = 0;
        for (const auto &ch: cfg.channels) {
            if (!ch.enabled) continue;
            enabled_count++;
            fr.add(ch.channel, Series.allocate(ch.ch.data_type, samples_per_read));
        }
        for (const auto &idx: indexes)
            fr.add(
                idx, Series.allocate(synnax::TIMESTAMP, samples_per_read));
        for (size_t i = 0; i < samples_per_read; i++) {
            UA_ReadResponse readResponse =
                    UA_Client_Service_read(client.get(), readRequest);
            auto status = readResponse.responseHeader.serviceResult;
            if (status != UA_STATUSCODE_GOOD) {
                UA_ReadResponse_clear(&readResponse);
                if (status == UA_STATUSCODE_BADCONNECTIONREJECTED || status ==
                    UA_STATUSCODE_BADSECURECHANNELCLOSED) {
                    return std::make_pair(std::move(fr), freighter::Error(
                                              driver::TYPE_TEMPORARY_HARDWARE_ERROR,
                                              "connection rejected"
                                          ));
                }
                return std::make_pair(std::move(fr), freighter::Error(
                                          driver::TYPE_CRITICAL_HARDWARE_ERROR,
                                          "failed to read value: " + std::string(
                                              UA_StatusCode_name(status))
                                      ));
            }

            // Process the read results
            for (size_t j = 0; j < readResponse.resultsSize; ++j) {
                UA_Variant *value = &readResponse.results[j].value;
                const auto &ch = cfg.channels[j];
                // Assuming the channels order hasn't changed
                if (readResponse.results[j].status != UA_STATUSCODE_GOOD) {
                    UA_ReadResponse_clear(&readResponse);
                    return std::make_pair(std::move(fr), freighter::Error(
                                              driver::TYPE_CRITICAL_HARDWARE_ERROR,
                                              "Failed to read value: " + std::string(
                                                  UA_StatusCode_name(
                                                      readResponse.results[j].
                                                      status))));
                }
                set_val_on_series(value, ch.ch.data_type, fr.series->at(j), i);
            }
            UA_ReadResponse_clear(&readResponse);
            const auto now = synnax::TimeStamp::now();
            for (size_t i = enabled_count; i < enabled_count + indexes.size(); i++) {
                auto ser = fr.series->at(i);
                ser.set(i, now.value);
            }
            timer.wait();
        }
        return std::make_pair(std::move(fr), freighter::NIL);
    }
};


std::unique_ptr<task::Task> Reader::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
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

    LOG(INFO) << "[opc.reader] successfully parsed configuration for " << task.name;

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
    auto [channelKeys, indexes] = res;

    // Connect to the OPC UA server.
    auto [ua_client, conn_err] = opc::connect(properties.connection);
    if (conn_err) {
        ctx->setState({
            .task = task.key,
            .variant = "error",
            .details = {{"message", conn_err.message()}}
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

    auto source = std::make_unique<ReaderSource>(
        cfg,
        ua_client,
        indexes
    );

    auto writer_cfg = synnax::WriterConfig{
        .channels = channelKeys,
        .start = TimeStamp::now(),
        .mode = synnax::WriterPersistStream,
        .enable_auto_commit = true,
    };

    auto pipe = pipeline::Acquisition(
        ctx,
        writer_cfg,
        std::move(source),
        breaker_config
    );
    ctx->setState({
        .task = task.key,
        .variant = "success",
        .details = {
            {"running", false}
        }
    });
    return std::make_unique<Reader>(ctx, task, cfg, breaker, std::move(pipe));
}

void Reader::exec(task::Command &cmd) {
    if (cmd.type == "start") {
        pipe.start();
        return ctx->setState({
            .task = task.key,
            .variant = "success",
            .details = {
                {"running", true}
            }
        });
    }
    if (cmd.type == "stop") {
        pipe.stop();
        return ctx->setState({
            .task = task.key,
            .variant = "success",
            .details = {
                {"running", false}
            }
        });
    }
    LOG(ERROR) << "unknown command type: " << cmd.type;
}
