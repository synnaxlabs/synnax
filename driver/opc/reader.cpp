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
#include "driver/vendor/open62541/open62541/out/include/open62541/types.h"
#include "driver/vendor/open62541/open62541/out/include/open62541/types_generated.h"
#include "driver/vendor/open62541/open62541/out/include/open62541/statuscodes.h"
#include "driver/vendor/open62541/open62541/out/include/open62541/client.h"
#include "driver/pipeline/acquisition.h"
#include "driver/vendor/open62541/open62541/out/include/open62541/common.h"


using namespace opc;


ReaderConfig::ReaderConfig(
    config::Parser &parser
): device(parser.required<std::string>("device")) {
    sample_rate = Rate(parser.required<std::float_t>("sample_rate"));
    stream_rate = Rate(parser.required<std::float_t>("stream_rate"));
    array_size = parser.optional<std::std::size_t>("array_size", 1);
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
            if (ch.index != 0) {
                channelKeys.push_back(ch.index);
                indexes.insert(ch.index);
            }
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
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;

    UA_ReadRequest req;
    std::vector<UA_ReadValueId> readValueIds;
    loop::Timer timer;
    synnax::Frame fr;
    std::unique_ptr<int64_t[]> timestamp_buf;

    ReaderSource(
        ReaderConfig cfg,
        const std::shared_ptr<UA_Client> &client,
        std::set<ChannelKey> indexes,
        std::shared_ptr<task::Context> ctx,
        synnax::Task task
    ) : cfg(std::move(cfg)),
        client(client),
        indexes(std::move(indexes)),
        ctx(std::move(ctx)),
        task(std::move(task)),
        timer(cfg.sample_rate / cfg.array_size) {
        if (cfg.array_size > 1)
            timestamp_buf = std::make_unique<int64_t[]>(cfg.array_size);
        initializeReadRequest();
    }

    void initializeReadRequest() {
        UA_ReadRequest_init(&req);
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
        req.nodesToRead = readValueIds.data();
        req.nodesToReadSize = readValueIds.size();
    }

    freighter::Error start() override {
        return freighter::NIL;
    }

    freighter::Error stop() override {
        return freighter::NIL;
    }

    freighter::Error communicate_res_error(
        UA_StatusCode status
    ) const {
        std::string variant = "error";
        freighter::Error err;
        if (
            status == UA_STATUSCODE_BADCONNECTIONREJECTED ||
            status == UA_STATUSCODE_BADSECURECHANNELCLOSED
        ) {
            err.type = driver::TYPE_TEMPORARY_HARDWARE_ERROR;
            err.data = "connection rejected";
        } else {
            err.type = driver::TYPE_CRITICAL_HARDWARE_ERROR;
            err.data = "failed to execute read: " + std::string(
                           UA_StatusCode_name(status));
        }
        ctx->setState({
            .task = task.key,
            .variant = err.type == driver::TYPE_TEMPORARY_HARDWARE_ERROR
                           ? "warning"
                           : "error",
            .details = json{{"message", err.message()}}
        });
        return err;
    }

    freighter::Error communicate_value_error(
        const std::string &channel,
        const UA_StatusCode &status
    ) const {
        std::string status_name = UA_StatusCode_name(status);
        std::string message = "Failed to read value from channel " + channel + ": " +
                              status_name;
        LOG(ERROR) << "[opc.reader]" << message;
        ctx->setState({
            .task = task.key,
            .variant = "error",
            .details = json{{"message", message}}
        });
        return {
            driver::TYPE_CRITICAL_HARDWARE_ERROR,
            message
        };
    }


    std::pair<Frame, freighter::Error> read() override {
        auto fr = Frame(cfg.channels.size() + indexes.size());
        const auto read_calls_per_cycle = static_cast<std::size_t>(
                                              cfg.sample_rate.value / (cfg.stream_rate.
                                                  value)) / cfg.array_size;
        const auto series_size = cfg.array_size * read_calls_per_cycle;

        std::size_t en_count = 0;
        for (const auto &ch: cfg.channels)
            if (ch.enabled) {
                fr.add(ch.channel, Series(ch.ch.data_type, series_size));
                en_count++;
            }
        for (const auto &idx: indexes)
            fr.add(idx, Series(synnax::TIMESTAMP, series_size));

        for (std::size_t i = 0; i < read_calls_per_cycle; i++) {
            UA_ReadResponse res = UA_Client_Service_read(client.get(), req);
            auto status = res.responseHeader.serviceResult;

            if (status != UA_STATUSCODE_GOOD) {
                auto err = communicate_res_error(status);
                UA_ReadResponse_clear(&res);
                return std::make_pair(std::move(fr), err);
            }

            for (std::std::size_t j = 0; j < res.resultsSize; ++j) {
                UA_Variant *value = &res.results[j].value;
                const auto &ch = cfg.channels[j];
                auto stat = res.results[j].status;
                if (stat != UA_STATUSCODE_GOOD) {
                    auto err = communicate_value_error(ch.ch.name, stat);
                    UA_ReadResponse_clear(&res);
                    return std::make_pair(std::move(fr), err);
                }
                set_val_on_series(value, i * cfg.array_size, fr.series->at(j));
            }
            UA_ReadResponse_clear(&res);
            if (cfg.array_size == 1) {
                const auto now = synnax::TimeStamp::now();
                for (std::size_t j = en_count; j < en_count + indexes.size(); j++)
                    fr.series->at(j).set(i, now.value);
            } else {
                // In this case we don't know the exact spacing between the timestamps,
                // so we just back it out from the sample rate.
                const auto now = synnax::TimeStamp::now();
                const auto spacing = cfg.sample_rate.period();
                // make an array of timestamps with the same spacing
                for (std::size_t k = 0; k < cfg.array_size; k++)
                    timestamp_buf[k] = (now + (spacing * k)).value;
                for (std::size_t j = en_count; j < en_count + indexes.size(); j++)
                    fr.series->at(j).set_array(timestamp_buf.get(), i, cfg.array_size);
            }
            std::cout << fr << std::endl;
            timer.wait();
        }
        return std::make_pair(std::move(fr), freighter::NIL);
    }
};


std::unique_ptr<task::Task> Reader::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    LOG(INFO) << "[opc.reader] configuring task " << task.name;
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
        .mode = synnax::WriterPersistStream,
        .enable_auto_commit = true
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
