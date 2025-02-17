// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/xjson/xjson.h"
#include "driver/errors/errors.h"
#include "x/cpp/loop/loop.h"
#include "driver/pipeline/acquisition.h"

#include "include/open62541/types.h"
#include "include/open62541/types_generated.h"
#include "include/open62541/statuscodes.h"
#include "include/open62541/client_highlevel.h"
#include "include/open62541/common.h"

using namespace opc;

opc::ReaderSource::ReaderSource(
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
    initialize_read_request();
    curr_state.task = task.key;
    curr_state.variant = "success";
    curr_state.details = json{
        {"message", "Task configured successfully"},
        {"running", true}
    };
    this->ctx->set_state(curr_state);
}

void opc::ReaderSource::initialize_read_request() {
    UA_ReadRequest_init(&req);
    // Allocate and prepare readValueIds for enabled channels
    readValueIds.reserve(cfg.channels.size()); // TODO: is this reserving every time?
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

void opc::ReaderSource::stopped_with_err(const xerrors::Error &err) {
    curr_state.variant = "error";
    curr_state.details = json{
        {"message", err.message()},
        {"running", false}
    };
    ctx->set_state(curr_state);
}

xerrors::Error opc::ReaderSource::communicate_value_error(
    const std::string &channel,
    const UA_StatusCode &status
) const {
    const std::string status_name = UA_StatusCode_name(status);
    const std::string message =
            "Failed to read value from channel " + channel + ": " +
            status_name;
    LOG(ERROR) << "[opc.reader]" << message;
    ctx->set_state({
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

size_t opc::ReaderSource::cap_array_length(
    const size_t i,
    const size_t length
) {
    if (i + length > cfg.array_size) {
        if (curr_state.variant != "warning") {
            curr_state.variant = "warning";
            curr_state.details = json{
                {
                    "message",
                    "Received array of length " + std::to_string(length) +
                    " from OPC UA server, which is larger than the configured size of "
                    + std::to_string(cfg.array_size) + ". Truncating array."
                },
                {"running", true}
            };
            ctx->set_state(curr_state);
        }
        return cfg.array_size - i;
    }
    return length;
}

size_t opc::ReaderSource::write_to_series(
    const UA_Variant *val,
    const size_t i,
    telem::Series &s
) {
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_FLOAT])) {
        const auto *data = static_cast<UA_Float *>(val->data);
        const size_t length = cap_array_length(i, val->arrayLength);
        if (s.data_type == telem::FLOAT32_T) return s.write(data, length);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_DOUBLE])) {
        const UA_Double *data = static_cast<UA_Double *>(val->data);
        const size_t length = cap_array_length(i, val->arrayLength);
        if (s.data_type == telem::FLOAT64_T) return s.write(data, length);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_INT16])) {
        const UA_Int16 *data = static_cast<UA_Int16 *>(val->data);
        const size_t length = cap_array_length(i, val->arrayLength);
        if (s.data_type == telem::INT16_T) return s.write(data, length);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_INT32])) {
        const UA_Int32 *data = static_cast<UA_Int32 *>(val->data);
        const size_t length = cap_array_length(i, val->arrayLength);
        if (s.data_type == telem::INT32_T) return s.write(data, length);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_INT64])) {
        const UA_Int64 *data = static_cast<UA_Int64 *>(val->data);
        const size_t length = cap_array_length(i, val->arrayLength);
        if (s.data_type == telem::INT64_T) return s.write(data, length);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_UINT32])) {
        const UA_UInt32 *data = static_cast<UA_UInt32 *>(val->data);
        const size_t length = cap_array_length(i, val->arrayLength);
        if (s.data_type == telem::UINT32_T) return s.write(data, length);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_UINT64])) {
        const UA_UInt64 *data = static_cast<UA_UInt64 *>(val->data);
        const size_t length = cap_array_length(i, val->arrayLength);
        if (s.data_type == telem::UINT64_T) return s.write(data, length);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_BYTE])) {
        const UA_Byte *data = static_cast<UA_Byte *>(val->data);
        const size_t length = cap_array_length(i, val->arrayLength);
        if (s.data_type == telem::UINT8_T) return s.write(data, length);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_SBYTE])) {
        const UA_SByte *data = static_cast<UA_SByte *>(val->data);
        const size_t length = cap_array_length(i, val->arrayLength);
        if (s.data_type == telem::INT8_T) return s.write(data, length);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_BOOLEAN])) {
        const UA_Boolean *data = static_cast<UA_Boolean *>(val->data);
        const size_t length = cap_array_length(i, val->arrayLength);
        if (s.data_type == telem::UINT8_T) return s.write(data, length);
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
        if (s.data_type == telem::FLOAT32_T) return s.write(value);
        if (s.data_type == telem::FLOAT64_T)
            return s.write(
                static_cast<double>(value));
        if (s.data_type == telem::INT32_T)
            return s.write(
                static_cast<int32_t>(value));
        if (s.data_type == telem::INT64_T)
            return s.write(
                static_cast<int64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_DOUBLE]) {
        const auto value = *static_cast<UA_Double *>(val->data);
        if (s.data_type == telem::FLOAT32_T)
            return s.write(
                static_cast<float>(value));
        if (s.data_type == telem::FLOAT64_T) return s.write(value);
        if (s.data_type == telem::INT32_T)
            return s.write(
                static_cast<int32_t>(value));
        if (s.data_type == telem::INT64_T)
            return s.write(
                static_cast<int64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT16]) {
        const auto value = *static_cast<UA_Int16 *>(val->data);
        if (s.data_type == telem::INT16_T) return s.write(value);
        if (s.data_type == telem::INT32_T)
            return s.write(
                static_cast<int16_t>(value));
        if (s.data_type == telem::INT64_T)
            return s.write(
                static_cast<int64_t>(value));
        if (s.data_type == telem::UINT16_T)
            return s.write(
                static_cast<uint16_t>(value));
        if (s.data_type == telem::UINT32_T)
            return s.write(
                static_cast<uint32_t>(value));
        if (s.data_type == telem::UINT64_T)
            return s.write(
                static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT32]) {
        const auto value = *static_cast<UA_Int32 *>(val->data);
        if (s.data_type == telem::INT32_T) return s.write(value);
        if (s.data_type == telem::INT64_T)
            return s.write(
                static_cast<int64_t>(value));
        if (s.data_type == telem::UINT32_T)
            return s.write(
                static_cast<uint32_t>(value));
        if (s.data_type == telem::UINT64_T)
            return s.write(
                static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT64]) {
        const auto value = *static_cast<UA_Int64 *>(val->data);
        if (s.data_type == telem::INT32_T)
            return s.write(
                static_cast<int32_t>(value));
        if (s.data_type == telem::INT64_T) return s.write(value);
        if (s.data_type == telem::UINT32_T)
            return s.write(
                static_cast<uint32_t>(value));
        if (s.data_type == telem::UINT64_T)
            return s.write(
                static_cast<uint64_t>(value));
        if (s.data_type == telem::TIMESTAMP_T)
            return s.write(static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_UINT32]) {
        const auto value = *static_cast<UA_UInt32 *>(val->data);
        if (s.data_type == telem::INT32_T)
            return s.write(
                static_cast<int32_t>(value));
        // Potential data loss
        if (s.data_type == telem::INT64_T)
            return s.write(
                static_cast<int64_t>(value));
        if (s.data_type == telem::UINT32_T) return s.write(value);
        if (s.data_type == telem::UINT64_T)
            return s.write(
                static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_UINT64]) {
        const auto value = *static_cast<UA_UInt64 *>(val->data);
        if (s.data_type == telem::UINT64_T) return s.write(value);
        if (s.data_type == telem::INT32_T)
            return s.write(
                static_cast<int32_t>(value));
        // Potential data loss
        if (s.data_type == telem::INT64_T)
            return s.write(
                static_cast<int64_t>(value));
        if (s.data_type == telem::UINT32_T)
            return s.write(
                static_cast<uint32_t>(value));
        // Potential data loss
        if (s.data_type == telem::TIMESTAMP_T)
            s.write(static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_BYTE]) {
        const auto value = *static_cast<UA_Byte *>(val->data);
        if (s.data_type == telem::UINT8_T) return s.write(value);
        if (s.data_type == telem::UINT16_T)
            return s.write(
                static_cast<uint16_t>(value));
        if (s.data_type == telem::UINT32_T)
            return s.write(
                static_cast<uint32_t>(value));
        if (s.data_type == telem::UINT64_T)
            return s.write(
                static_cast<uint64_t>(value));
        if (s.data_type == telem::INT8_T) return s.write(static_cast<int8_t>(value));
        if (s.data_type == telem::INT16_T)
            return s.write(
                static_cast<int16_t>(value));
        if (s.data_type == telem::INT32_T)
            return s.write(
                static_cast<int32_t>(value));
        if (s.data_type == telem::INT64_T)
            return s.write(
                static_cast<int64_t>(value));
        if (s.data_type == telem::FLOAT32_T)
            return s.write(
                static_cast<float>(value));
        if (s.data_type == telem::FLOAT64_T)
            return s.write(
                static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_SBYTE]) {
        const auto value = *static_cast<UA_SByte *>(val->data);
        if (s.data_type == telem::INT8_T) return s.write(value);
        if (s.data_type == telem::INT16_T)
            return s.write(
                static_cast<int16_t>(value));
        if (s.data_type == telem::INT32_T)
            return s.write(
                static_cast<int32_t>(value));
        if (s.data_type == telem::INT64_T)
            return s.write(
                static_cast<int64_t>(value));
        if (s.data_type == telem::FLOAT32_T)
            return s.write(
                static_cast<float>(value));
        if (s.data_type == telem::FLOAT64_T)
            return s.write(
                static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_BOOLEAN]) {
        const auto value = *static_cast<UA_Boolean *>(val->data);
        if (s.data_type == telem::UINT8_T)
            return s.write(
                static_cast<uint8_t>(value));
        if (s.data_type == telem::UINT16_T)
            return s.write(
                static_cast<uint16_t>(value));
        if (s.data_type == telem::UINT32_T)
            return s.write(
                static_cast<uint32_t>(value));
        if (s.data_type == telem::UINT64_T)
            return s.write(
                static_cast<uint64_t>(value));
        if (s.data_type == telem::INT8_T) return s.write(static_cast<int8_t>(value));
        if (s.data_type == telem::INT16_T)
            return s.write(
                static_cast<int16_t>(value));
        if (s.data_type == telem::INT32_T)
            return s.write(
                static_cast<int32_t>(value));
        if (s.data_type == telem::INT64_T)
            return s.write(
                static_cast<int64_t>(value));
        if (s.data_type == telem::FLOAT32_T)
            return s.write(
                static_cast<float>(value));
        if (s.data_type == telem::FLOAT64_T)
            return s.write(
                static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_DATETIME]) {
        const auto value = *static_cast<UA_DateTime *>(val->data);
        if (s.data_type == telem::INT64_T)
            return s.write(
                ua_datetime_to_unix_nano(value));
        if (s.data_type == telem::TIMESTAMP_T)
            return s.write(
                ua_datetime_to_unix_nano(value));
        if (s.data_type == telem::UINT64_T)
            return s.write(
                static_cast<uint64_t>(ua_datetime_to_unix_nano(value)));
        if (s.data_type == telem::FLOAT32_T)
            return s.write(
                static_cast<float>(value));
        if (s.data_type == telem::FLOAT64_T)
            return s.write(
                static_cast<double>(value));
    }
    LOG(ERROR) << "[opc.reader] unsupported data type: " << val->type->typeName << " for task " << task.name;
}

std::pair<Frame, xerrors::Error> opc::ReaderSource::read(breaker::Breaker &breaker) {
    auto fr = Frame(cfg.channels.size() + indexes.size());

    // TODO: what is read_calls_per_cycle? explain whats happening here
    auto read_calls_per_cycle = static_cast<std::size_t>(cfg.sample_rate.value / cfg.stream_rate.value);
    auto series_size = read_calls_per_cycle;
    if (cfg.array_size > 1) {
        read_calls_per_cycle = 1;
        series_size = cfg.array_size * read_calls_per_cycle;
    }

    std::size_t en_count = 0; // enabled channels
    for (const auto &ch: cfg.channels)
        if (ch.enabled) {
            auto ser = telem::Series(ch.ch.data_type, series_size);
            fr.emplace(ch.channel, std::move(ser));
            en_count++;
        }
    for (const auto &idx: indexes) {
        auto ser = telem::Series(telem::TIMESTAMP_T, series_size);
        fr.emplace(idx, std::move(ser));
    }

    for (std::size_t i = 0; i < read_calls_per_cycle; i++) {
        UA_ReadResponse res = UA_Client_Service_read(client.get(), req);
        auto status = res.responseHeader.serviceResult;

        if (status != UA_STATUSCODE_GOOD) {
            auto err = opc::communicate_response_error(status, this->ctx, this->curr_state);
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
                        "Received array of length " + std::to_string(next_arr_size)
                        +
                        " from OPC UA server, which is different from the previous array length of "
                        + std::to_string(curr_arr_size) + ". Skipping write."
                    },
                    {"running", true}
                };
                UA_ReadResponse_clear(&res);
                return std::make_pair(std::move(fr),
                                      driver::TEMPORARY_HARDWARE_ERROR);
            }
            curr_arr_size = next_arr_size;
        }

        UA_ReadResponse_clear(&res);

        if (cfg.array_size == 1) {
            const auto now = telem::TimeStamp::now();
            for (std::size_t j = en_count; j < en_count + indexes.size(); j++)
                fr.series->at(j).write(now.value);
        } else if (indexes.size() > 0) {
            // In this case we don't know the exact spacing between the timestamps,
            // so we just back it out from the sample rate.
            const auto now = telem::TimeStamp::now();
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
                    {"running", true}
                };
                ctx->set_state(curr_state);
            }
        }
    }
    if (exceed_time_count < 5 && curr_state.variant != "success") {
        curr_state.variant = "success";
        curr_state.details = json{
            {"message", "Operating normally"},
            {"running", true}
        };
        ctx->set_state(curr_state);
    }
    return std::make_pair(std::move(fr), xerrors::NIL);
}
