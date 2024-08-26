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
#include <string>
#include <utility>
#include "glog/logging.h"

#include "driver/opc/writer.h"
#include "driver/opc/util.h"
#include "driver/config/config.h"
#include "driver/loop/loop.h"
#include "driver/pipeline/acquisition.h"
#include "driver/errors/errors.h"

#include "include/open62541/types.h"
#include "include/open62541/types_generated.h"
#include "include/open62541/statuscodes.h"
#include "include/open62541/client_highlevel.h"
#include "include/open62541/common.h"
#include "include/open62541/client_subscriptions.h"

///////////////////////////////////////////////////////////////////////////////////
//                                      Sink                                     //
///////////////////////////////////////////////////////////////////////////////////
opc::Sink::Sink(
    WriterConfig cfg,
    const std::shared_ptr<UA_Client> &ua_client,
    const std::shared_ptr<task::Context> &ctx,
    synnax::Task task
){}

/*
    typedef struct {
        UA_RequestHeader requestHeader;
        size_t nodesToWriteSize;
        UA_WriteValue *nodesToWrite;
    } UA_WriteRequest;

    typedef struct {
        UA_NodeId nodeId;
        UA_UInt32 attributeId;
        UA_String indexRange;
        UA_DataValue value;
    } UA_WriteValue;
 */
void opc::Sink::initialize_write_request(const synnax::Frame &frame){
    // the number of frames we are writing corresponds to the number of values in the frame
    uint32_t frame_index = 0;

    UA_WriteRequest_init(&this->req);
    std::vector<UA_WriteValue> nodes_to_write(frame.channels->size());
    req.nodesToWriteSize = frame.channels->size();

    for(const auto key : *(frame.channels)){
        // get channel config using key
        auto ch = this->cmd_channel_map[key];
        this->initialize_write_value(
            frame, 
            frame_index, 
            ch, 
            &nodes_to_write[frame_index]
        );
        frame_index++;
    }
    req.nodesToWrite = nodes_to_write.data();
};


void opc::Sink::initialize_write_value(
        const synnax::Frame &frame,
        uint32_t &index,
        WriterChannelConfig &ch,
        UA_WriteValue *write_value
){
    write_value->nodeId = ch.node;
    write_value->attributeId = UA_ATTRIBUTEID_VALUE;
    write_value->value.hasValue = true; // TODO what is this
    write_value->value.value.storageType = UA_VARIANT_DATA_NODELETE; // do not free integer on deletion

    this->cast_and_set_type(frame, index, ch, write_value);
}

///@brief Sets the appropriate type for the write value and casts the series data as necessary to one of the supported types
/// on Synnax
void opc::Sink::cast_and_set_type(const synnax::Frame &frame, const uint32_t &series_index,
                                  const opc::WriterChannelConfig &ch, UA_WriteValue *write_value) {
    auto &type = frame.series->at(series_index).data_type;

    // TODO: Need to test if I could specify writing float32 if the actual node type is float64 or real, etc
    if(type == synnax::FLOAT64) {
        write_value->value.value.data = &frame.series->at(series_index).values<double>()[0];
        write_value->value.value.type = &UA_TYPES[UA_TYPES_DOUBLE];
    }else if(type == synnax::FLOAT32){
        write_value->value.value.data = &frame.series->at(series_index).values<float>()[0];;
        write_value->value.value.type = &UA_TYPES[UA_TYPES_FLOAT];
    }
    else if(type == synnax::INT32){
        write_value->value.value.data = &frame.series->at(series_index).values<int32_t>()[0];
        write_value->value.value.type = &UA_TYPES[UA_TYPES_INT32];
    }
    else if(type == synnax::INT16){
        write_value->value.value.data = &frame.series->at(series_index).values<int16_t>()[0];
        write_value->value.value.type = &UA_TYPES[UA_TYPES_INT16];
    }
    else if(type == synnax::INT8){
        write_value->value.value.data = &frame.series->at(series_index).values<int8_t>()[0];
        write_value->value.value.type = &UA_TYPES[UA_TYPES_SBYTE];
    }
    else if(type == synnax::UINT64){
        write_value->value.value.data = &frame.series->at(series_index).values<uint64_t>()[0];
        write_value->value.value.type = &UA_TYPES[UA_TYPES_UINT64];
    }
    else if(type == synnax::UINT32){
        write_value->value.value.data = &frame.series->at(series_index).values<uint32_t>()[0];
        write_value->value.value.type = &UA_TYPES[UA_TYPES_UINT32];
    }
    else if(type == synnax::UINT16){
        write_value->value.value.data = &frame.series->at(series_index).values<uint16_t>()[0];
        write_value->value.value.type = &UA_TYPES[UA_TYPES_UINT16];
    }
    else if(type == synnax::UINT8){
        write_value->value.value.data = &frame.series->at(series_index).values<uint8_t>()[0];
        write_value->value.value.type = &UA_TYPES[UA_TYPES_BYTE];
    }
    else if(type == synnax::TIMESTAMP){
        write_value->value.value.data = &frame.series->at(series_index).values<int64_t>()[0];
        write_value->value.value.type = &UA_TYPES[UA_TYPES_DATETIME];
    }
    // TODO: add uint128 (compiler has issues with uint128_t)
//    LOG(ERROR) << "[opc.sink] Unsupported data type" << type;
}

void opc::Sink::stoppedWithErr(const freighter::Error &err){
    curr_state.variant = "error";
    curr_state.details = json{
            {"message", err.message()},
            {"running", false}
    };
    ctx->setState(curr_state);
};

// TODO: identical to impl in reader.cpp -> move to util.cpp
freighter::Error opc::Sink::communicate_response_error(const UA_StatusCode &status){
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
                {"running", true}
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
                {"running", false}
        };
    }
    ctx->setState(curr_state);
    return err;
};

/// @brief sends out write request to the OPC server.
freighter::Error opc::Sink::write(synnax::Frame frame){
    this->initialize_write_request(frame);
    UA_WriteResponse res = UA_Client_Service_write(this->ua_client.get(), this->req);
    auto status = res.responseHeader.serviceResult;

    if(status != UA_STATUSCODE_GOOD){
        auto err = this->communicate_response_error(status);
        UA_WriteResponse_clear(&res);
        return err;
    }
    UA_WriteRequest_clear(&this->req);
    UA_WriteResponse_clear(&res);
    return freighter::NIL;
};
