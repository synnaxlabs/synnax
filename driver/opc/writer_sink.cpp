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
#include <string>
#include <utility>
#include "glog/logging.h"

#include "driver/opc/writer.h"
#include "driver/opc/util.h"
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/loop/loop.h"
#include "driver/pipeline/acquisition.h"
#include "driver/errors/errors.h"

#include "include/open62541/types.h"
#include "include/open62541/types_generated.h"
#include "include/open62541/statuscodes.h"
#include "include/open62541/client_highlevel.h"
#include "include/open62541/common.h"
#include "include/open62541/client_subscriptions.h"

opc::WriterSink::WriterSink(
    WriterConfig cfg,
    const std::shared_ptr<UA_Client> &ua_client,
    const std::shared_ptr<task::Context> &ctx,
    synnax::Task task,
    opc::DeviceProperties device_props
) : cfg(std::move(cfg)),
    ua_client(ua_client),
    ctx(ctx),
    task(std::move(task)),
    device_props(std::move(device_props)),
    breaker(breaker::Breaker(breaker::default_config(task.name))) {
    for (auto &ch: this->cfg.channels)
        this->cmd_channel_map[ch.cmd_channel] = ch;
    this->breaker.start();
    this->keep_alive_thread = std::thread(&opc::WriterSink::maintain_connection, this);
};

inline void opc::WriterSink::set_variant(
    UA_Variant *val, const synnax::Frame &frame,
    const uint32_t &series_index,
    const telem::DataType &type) {
    UA_StatusCode status = UA_STATUSCODE_GOOD;
    if (type == telem::FLOAT64_T) {
        double data = frame.series->at(series_index).values<double>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_DOUBLE]);
    } else if (type == telem::FLOAT32_T) {
        float data = frame.series->at(series_index).values<float>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_FLOAT]);
    } else if (type == telem::INT32_T) {
        int32_t data = frame.series->at(series_index).values<int32_t>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_INT32]);
    } else if (type == telem::INT16_T) {
        int16_t data = frame.series->at(series_index).values<int16_t>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_INT16]);
    } else if (type == telem::INT8_T) {
        int8_t data = frame.series->at(series_index).values<int8_t>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_SBYTE]);
    } else if (type == telem::UINT64_T) {
        uint64_t data = frame.series->at(series_index).values<uint64_t>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_UINT64]);
    } else if (type == telem::UINT32_T) {
        uint32_t data = frame.series->at(series_index).values<uint32_t>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_UINT32]);
    } else if (type == telem::UINT16_T) {
        uint16_t data = frame.series->at(series_index).values<uint16_t>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_UINT16]);
    } else if (type == telem::UINT8_T) {
        uint8_t data = frame.series->at(series_index).values<uint8_t>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_BYTE]);
    } else if (type == telem::TIMESTAMP_T) {
        uint64_t data = frame.series->at(series_index).values<uint64_t>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_DATETIME]);
    }
    if (status != UA_STATUSCODE_GOOD) {
        LOG(ERROR) << "[opc.sink] Failed to set variant";
    }
};

void opc::WriterSink::stopped_with_err(const xerrors::Error &err) {
    LOG(ERROR) << "[opc.sink] Stopped with error: " << err.message();
    curr_state.variant = "error";
    curr_state.details = json{
        {"message", err.message()},
        {"running", false}
    };
    ctx->set_state(curr_state);
};


/// @brief sends out write request to the OPC server.
xerrors::Error opc::WriterSink::write(const synnax::Frame &frame) {
    auto client = this->ua_client.get();
    auto frame_index = 0;
    for (const auto key: *(frame.channels)) {
        if (this->cmd_channel_map.find(key) == this->cmd_channel_map.end()) {
            LOG(ERROR) << "[opc.sink] Channel key not found in map";
            continue;
        }
        auto ch = this->cmd_channel_map[key];

        UA_Variant *val = UA_Variant_new();
        auto data_Type = frame.series->at(frame_index).data_type;
        this->set_variant(val, frame, frame_index, data_Type);
        UA_StatusCode retval; {
            std::lock_guard<std::mutex> lock(this->client_mutex);
            retval = UA_Client_writeValueAttribute(client, ch.node, val);
        }
        if (retval != UA_STATUSCODE_GOOD) {
            auto err = opc::communicate_response_error(
                retval,
                this->ctx,
                this->curr_state
            );
            UA_Variant_delete(val);
            LOG(ERROR) << "[opc.sink] Failed to write to node: " << key;
            return err;
        }
        UA_Variant_delete(val);
        frame_index++;
    }
    return xerrors::NIL;
};

void opc::WriterSink::maintain_connection() {
    while (this->breaker.running()) {
        this->breaker.wait_for(this->ping_rate.period().chrono());
        UA_Variant value;
        UA_Variant_init(&value); {
            std::lock_guard<std::mutex> lock(this->client_mutex);
            UA_StatusCode retval = UA_Client_readValueAttribute(
                this->ua_client.get(),
                UA_NODEID_NUMERIC(
                    0,
                    UA_NS0ID_SERVER_SERVERSTATUS_STATE
                ),
                &value);
        }
        UA_Variant_clear(&value);
    }
}
