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


ReaderConfig::ReaderConfig(
    config::Parser &parser
) : device(parser.required<std::string>("device")),
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
    int exceed_time_count = 0;
    task::State curr_state;

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
        curr_state.task = task.key;
        curr_state.variant = "success";
        curr_state.details = json{
            {"message", "Task configured successfully"},
            {"running", true}
        };
        this->ctx->setState(curr_state);
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

    void stoppedWithErr(const freighter::Error &err) override {
        curr_state.variant = "error";
        curr_state.details = json{
            {"message", err.message()},
            {"running", false}
        };
        ctx->setState(curr_state);
    }

    freighter::Error communicate_res_error(const UA_StatusCode &status) {
        freighter::Error err;
        if (
            status == UA_STATUSCODE_BADCONNECTIONREJECTED ||
            status == UA_STATUSCODE_BADSECURECHANNELCLOSED
            ) {
            err.type = driver::TEMPORARY_HARDWARE_ERROR.type;
            err.data = "connection rejected";
            curr_state.variant = "warning";
            curr_state.details = json{
                {
                    "message",
                               "Temporarily unable to reach OPC UA server. Will keep trying."
                },
                {   "running", true}
            };
        } else {
            err.type = driver::CRITICAL_HARDWARE_ERROR.type;
            err.data = "failed to execute read: " + std::string(
                UA_StatusCode_name(status));
            curr_state.variant = "error";
            curr_state.details = json{
                {
                    "message", "Failed to read from OPC UA server: " + std::string(
                    UA_StatusCode_name(status))
                },
                {   "running", false}
            };
        }
        ctx->setState(curr_state);
        return err;
    }

    [[nodiscard]] freighter::Error communicate_value_error(
        const std::string &channel,
        const UA_StatusCode &status
    ) const {
        const std::string status_name = UA_StatusCode_name(status);
        const std::string message =
            "Failed to read value from channel " + channel + ": " +
            status_name;
        LOG(ERROR) << "[opc.reader]" << message;
        ctx->setState({
                          .task = task.key,
                          .variant = "error",
                          .details = json{
                              {"message", message},
                              {"running", false}
                          }
                      });
        return {
            driver::CRITICAL_HARDWARE_ERROR.type,
            message
        };
    }

    size_t cap_array_length(
        const size_t i,
        const size_t length
    ) {
        if (i + length > cfg.array_size) {
            if (curr_state.variant != "warning") {
                curr_state.variant = "warning";
                curr_state.details = json{
                    {
                        "message",
                                   "Received array of length " +
                                   std::to_string(length) +
                                   " from OPC UA server, which is larger than the configured size of "
                                   + std::to_string(cfg.array_size) +
                                   ". Truncating array."
                    },
                    {   "running", true}
                };
                ctx->setState(curr_state);
            }
            return cfg.array_size - i;
        }
        return length;
    }

    size_t write_to_series(
        const UA_Variant *val,
        const size_t i,
        synnax::Series &s
    ) {
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_FLOAT])) {
            const auto *data = static_cast<UA_Float *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            if (s.data_type == synnax::FLOAT32) return s.write(data, length);
        }
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_DOUBLE])) {
            const UA_Double *data = static_cast<UA_Double *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            if (s.data_type == synnax::FLOAT64) return s.write(data, length);
        }
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_INT16])) {
            const UA_Int16 *data = static_cast<UA_Int16 *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            if (s.data_type == synnax::INT16) return s.write(data, length);
        }
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_INT32])) {
            const UA_Int32 *data = static_cast<UA_Int32 *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            if (s.data_type == synnax::INT32) return s.write(data, length);
        }
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_INT64])) {
            const UA_Int64 *data = static_cast<UA_Int64 *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            if (s.data_type == synnax::INT64) return s.write(data, length);
        }
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_UINT32])) {
            const UA_UInt32 *data = static_cast<UA_UInt32 *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            if (s.data_type == synnax::UINT32) return s.write(data, length);
        }
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_UINT64])) {
            const UA_UInt64 *data = static_cast<UA_UInt64 *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            if (s.data_type == synnax::UINT64) return s.write(data, length);
        }
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_BYTE])) {
            const UA_Byte *data = static_cast<UA_Byte *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            if (s.data_type == synnax::UINT8) return s.write(data, length);
        }
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_SBYTE])) {
            const UA_SByte *data = static_cast<UA_SByte *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            if (s.data_type == synnax::INT8) return s.write(data, length);
        }
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_BOOLEAN])) {
            const UA_Boolean *data = static_cast<UA_Boolean *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            if (s.data_type == synnax::UINT8) return s.write(data, length);
        }
        if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_DATETIME])) {
            const UA_DateTime *data = static_cast<UA_DateTime *>(val->data);
            const size_t length = cap_array_length(i, val->arrayLength);
            size_t acc = 0;
            for (size_t j = 0; j < length; ++j)
                acc += s.write(ua_datetime_to_unix_nano(data[j]));
            return acc;
        }
        if (val->type == &UA_TYPES[UA_TYPES_FLOAT]) {
            const auto value = *static_cast<UA_Float *>(val->data);
            if (s.data_type == synnax::FLOAT32) return s.write(value);
            if (s.data_type == synnax::FLOAT64)
                return s.write(
                    static_cast<double>(value));
            if (s.data_type == synnax::INT32)
                return s.write(
                    static_cast<int32_t>(value));
            if (s.data_type == synnax::INT64)
                return s.write(
                    static_cast<int64_t>(value));
        }
        if (val->type == &UA_TYPES[UA_TYPES_DOUBLE]) {
            const auto value = *static_cast<UA_Double *>(val->data);
            if (s.data_type == synnax::FLOAT32)
                return s.write(
                    static_cast<float>(value));
            if (s.data_type == synnax::FLOAT64) return s.write(value);
            if (s.data_type == synnax::INT32)
                return s.write(
                    static_cast<int32_t>(value));
            if (s.data_type == synnax::INT64)
                return s.write(
                    static_cast<int64_t>(value));
        }
        if (val->type == &UA_TYPES[UA_TYPES_INT16]) {
            const auto value = *static_cast<UA_Int16 *>(val->data);
            if (s.data_type == synnax::INT16) return s.write(value);
            if (s.data_type == synnax::INT32)
                return s.write(
                    static_cast<int16_t>(value));
            if (s.data_type == synnax::INT64)
                return s.write(
                    static_cast<int64_t>(value));
            if (s.data_type == synnax::UINT16)
                return s.write(
                    static_cast<uint16_t>(value));
            if (s.data_type == synnax::UINT32)
                return s.write(
                    static_cast<uint32_t>(value));
            if (s.data_type == synnax::UINT64)
                return s.write(
                    static_cast<uint64_t>(value));
        }
        if (val->type == &UA_TYPES[UA_TYPES_INT32]) {
            const auto value = *static_cast<UA_Int32 *>(val->data);
            if (s.data_type == synnax::INT32) return s.write(value);
            if (s.data_type == synnax::INT64)
                return s.write(
                    static_cast<int64_t>(value));
            if (s.data_type == synnax::UINT32)
                return s.write(
                    static_cast<uint32_t>(value));
            if (s.data_type == synnax::UINT64)
                return s.write(
                    static_cast<uint64_t>(value));
        }
        if (val->type == &UA_TYPES[UA_TYPES_INT64]) {
            const auto value = *static_cast<UA_Int64 *>(val->data);
            if (s.data_type == synnax::INT32)
                return s.write(
                    static_cast<int32_t>(value));
            if (s.data_type == synnax::INT64) return s.write(value);
            if (s.data_type == synnax::UINT32)
                return s.write(
                    static_cast<uint32_t>(value));
            if (s.data_type == synnax::UINT64)
                return s.write(
                    static_cast<uint64_t>(value));
            if (s.data_type == synnax::TIMESTAMP)
                return s.write(static_cast<uint64_t>(value));
        }
        if (val->type == &UA_TYPES[UA_TYPES_UINT32]) {
            const auto value = *static_cast<UA_UInt32 *>(val->data);
            if (s.data_type == synnax::INT32)
                return s.write(
                    static_cast<int32_t>(value));
            // Potential data loss
            if (s.data_type == synnax::INT64)
                return s.write(
                    static_cast<int64_t>(value));
            if (s.data_type == synnax::UINT32) return s.write(value);
            if (s.data_type == synnax::UINT64)
                return s.write(
                    static_cast<uint64_t>(value));
        }
        if (val->type == &UA_TYPES[UA_TYPES_UINT64]) {
            const auto value = *static_cast<UA_UInt64 *>(val->data);
            if (s.data_type == synnax::UINT64) return s.write(value);
            if (s.data_type == synnax::INT32)
                return s.write(
                    static_cast<int32_t>(value));
            // Potential data loss
            if (s.data_type == synnax::INT64)
                return s.write(
                    static_cast<int64_t>(value));
            if (s.data_type == synnax::UINT32)
                return s.write(
                    static_cast<uint32_t>(value));
            // Potential data loss
            if (s.data_type == synnax::TIMESTAMP)
                s.write(static_cast<uint64_t>(value));
        }
        if (val->type == &UA_TYPES[UA_TYPES_BYTE]) {
            const auto value = *static_cast<UA_Byte *>(val->data);
            if (s.data_type == synnax::UINT8) return s.write(value);
            if (s.data_type == synnax::UINT16)
                return s.write(
                    static_cast<uint16_t>(value));
            if (s.data_type == synnax::UINT32)
                return s.write(
                    static_cast<uint32_t>(value));
            if (s.data_type == synnax::UINT64)
                return s.write(
                    static_cast<uint64_t>(value));
            if (s.data_type == synnax::INT8) return s.write(static_cast<int8_t>(value));
            if (s.data_type == synnax::INT16)
                return s.write(
                    static_cast<int16_t>(value));
            if (s.data_type == synnax::INT32)
                return s.write(
                    static_cast<int32_t>(value));
            if (s.data_type == synnax::INT64)
                return s.write(
                    static_cast<int64_t>(value));
            if (s.data_type == synnax::FLOAT32)
                return s.write(
                    static_cast<float>(value));
            if (s.data_type == synnax::FLOAT64)
                return s.write(
                    static_cast<double>(value));
        }
        if (val->type == &UA_TYPES[UA_TYPES_SBYTE]) {
            const auto value = *static_cast<UA_SByte *>(val->data);
            if (s.data_type == synnax::INT8) return s.write(value);
            if (s.data_type == synnax::INT16)
                return s.write(
                    static_cast<int16_t>(value));
            if (s.data_type == synnax::INT32)
                return s.write(
                    static_cast<int32_t>(value));
            if (s.data_type == synnax::INT64)
                return s.write(
                    static_cast<int64_t>(value));
            if (s.data_type == synnax::FLOAT32)
                return s.write(
                    static_cast<float>(value));
            if (s.data_type == synnax::FLOAT64)
                return s.write(
                    static_cast<double>(value));
        }
        if (val->type == &UA_TYPES[UA_TYPES_BOOLEAN]) {
            const auto value = *static_cast<UA_Boolean *>(val->data);
            if (s.data_type == synnax::UINT8)
                return s.write(
                    static_cast<uint8_t>(value));
            if (s.data_type == synnax::UINT16)
                return s.write(
                    static_cast<uint16_t>(value));
            if (s.data_type == synnax::UINT32)
                return s.write(
                    static_cast<uint32_t>(value));
            if (s.data_type == synnax::UINT64)
                return s.write(
                    static_cast<uint64_t>(value));
            if (s.data_type == synnax::INT8) return s.write(static_cast<int8_t>(value));
            if (s.data_type == synnax::INT16)
                return s.write(
                    static_cast<int16_t>(value));
            if (s.data_type == synnax::INT32)
                return s.write(
                    static_cast<int32_t>(value));
            if (s.data_type == synnax::INT64)
                return s.write(
                    static_cast<int64_t>(value));
            if (s.data_type == synnax::FLOAT32)
                return s.write(
                    static_cast<float>(value));
            if (s.data_type == synnax::FLOAT64)
                return s.write(
                    static_cast<double>(value));
        }
        if (val->type == &UA_TYPES[UA_TYPES_DATETIME]) {
            const auto value = *static_cast<UA_DateTime *>(val->data);
            if (s.data_type == synnax::INT64)
                return s.write(
                    ua_datetime_to_unix_nano(value));
            if (s.data_type == synnax::TIMESTAMP)
                return s.write(
                    ua_datetime_to_unix_nano(value));
            if (s.data_type == synnax::UINT64)
                return s.write(
                    static_cast<uint64_t>(ua_datetime_to_unix_nano(value)));
            if (s.data_type == synnax::FLOAT32)
                return s.write(
                    static_cast<float>(value));
            if (s.data_type == synnax::FLOAT64)
                return s.write(
                    static_cast<double>(value));
        }
        LOG(ERROR) << "[opc.reader] unsupported data type: " << val->type->typeName
                   << " for task " << task.name;
    }

    std::pair<Frame, freighter::Error> read(breaker::Breaker &breaker) override {
        auto fr = Frame(cfg.channels.size() + indexes.size());
        auto read_calls_per_cycle = static_cast<std::size_t>(
            cfg.sample_rate.value / cfg.stream_rate.value
        );
        auto series_size = read_calls_per_cycle;
        if (cfg.array_size > 1) {
            read_calls_per_cycle = 1;
            series_size = cfg.array_size * read_calls_per_cycle;
        }

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

            size_t curr_arr_size = 0;
            for (std::size_t j = 0; j < res.resultsSize; ++j) {
                UA_Variant *value = &res.results[j].value;
                const auto &ch = cfg.channels[j];
                auto stat = res.results[j].status;
                if (stat != UA_STATUSCODE_GOOD) {
                    auto err = communicate_value_error(ch.ch.name, stat);
                    UA_ReadResponse_clear(&res);
                    return std::make_pair(std::move(fr), err);
                }
                const auto next_arr_size = write_to_series(
                    value, i * cfg.array_size, fr.series->at(j));
                if (j != 0 && curr_arr_size != next_arr_size) {
                    curr_state.variant = "warning";
                    curr_state.details = json{
                        {
                            "message",
                                       "Received array of length " +
                                       std::to_string(next_arr_size)
                                       +
                                       " from OPC UA server, which is different from the previous array length of "
                                       + std::to_string(curr_arr_size) +
                                       ". Skipping write."
                        },
                        {   "running", true}
                    };
                    UA_ReadResponse_clear(&res);
                    return std::make_pair(std::move(fr),
                                          driver::TEMPORARY_HARDWARE_ERROR);
                }
                curr_arr_size = next_arr_size;
            }

            UA_ReadResponse_clear(&res);

            if (cfg.array_size == 1) {
                const auto now = synnax::TimeStamp::now();
                for (std::size_t j = en_count; j < en_count + indexes.size(); j++)
                    fr.series->at(j).write(now.value);
            } else if (indexes.size() > 0) {
                // In this case we don't know the exact spacing between the timestamps,
                // so we just back it out from the sample rate.
                const auto now = synnax::TimeStamp::now();
                const auto spacing = cfg.sample_rate.period();
                // make an array of timestamps with the same spacing
                auto to_generate = std::min(series_size, curr_arr_size);
                for (std::size_t k = 0; k < to_generate; k++)
                    timestamp_buf[k] = (now + (spacing * k)).value;
                for (std::size_t j = en_count; j < en_count + indexes.size(); j++)
                    fr.series->at(j).write(timestamp_buf.get(), cfg.array_size);
            }
            auto [elapsed, ok] = timer.wait(breaker);
            if (!ok && exceed_time_count <= 5) {
                exceed_time_count++;
                if (exceed_time_count == 5) {
                    curr_state.variant = "warning";
                    curr_state.details = json{
                        {
                            "message",
                                       "Sample rate exceeds OPC UA server throughput. samples may be delayed"
                        },
                        {   "running", true}
                    };
                    ctx->setState(curr_state);
                }
            }
        }
        if (exceed_time_count < 5 && curr_state.variant != "success") {
            curr_state.variant = "success";
            curr_state.details = json{
                {"message", "Operating normally"},
                {"running", true}
            };
            ctx->setState(curr_state);
        }
        return std::make_pair(std::move(fr), freighter::NIL);
    }
};


std::unique_ptr<task::Task> Reader::configure(
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
    auto [channelKeys, indexes] = res;

    // Connect to the OPC UA server.
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

void Reader::exec(task::Command &cmd) {
    if (cmd.type == "start") {
        this->start();
    } else if (cmd.type == "stop") return stop();
    else
        LOG(ERROR) << "unknown command type: " << cmd.type;
}

void Reader::stop() {
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

void Reader::start() {
    // first try to check for timeout
    UA_StatusCode status = UA_Client_connect(this->ua_client.get(),
                                             device_props.connection.endpoint.c_str());
    if (status != UA_STATUSCODE_GOOD) {
        // attempt again to reestablish if timed out
        UA_StatusCode status_retry = UA_Client_connect(this->ua_client.get(),
                                                       device_props.connection.endpoint.c_str());
        if (status_retry != UA_STATUSCODE_GOOD) {
            ctx->setState({
                              .task = task.key,
                              .variant = "error",
                              .details = json{
                                  {"message",
                                   "Failed to connect to OPC UA server: " + std::string(
                                       UA_StatusCode_name(status))}
                              }
                          });
            LOG(ERROR) << "[opc.reader] connection failed: "
                       << UA_StatusCode_name(status);
        }
    }
    VLOG(1) << "[opc.reader] Connection Established";
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
