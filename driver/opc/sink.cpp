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
    const std::shared_ptr<UA_Client> &ua_client2,
    const std::shared_ptr<task::Context> &ctx,
    synnax::Task task
) : cfg(std::move(cfg)),
    ua_client(ua_client),
    ua_client2(ua_client2),
    ctx(ctx),
    task(std::move(task))
{
    // iterate through cfg channels
    for(auto &ch : this->cfg.channels){
        this->cmd_channel_map[ch.cmd_channel] = ch;
    }
};

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

    // attempt 2
    auto client = this->ua_client.get();
    /* Read attribute for "the.answer3" */
//    UA_Byte value3 = 0;
//    printf("\nReading the value of node (1, \"the.answer3\"):\n");
//    UA_Variant *val3 = UA_Variant_new();
//    UA_StatusCode retval = UA_Client_readValueAttribute(client, UA_NODEID_STRING(1, "the.answer3"), val3);
//    if(retval == UA_STATUSCODE_GOOD && UA_Variant_isScalar(val3) &&
//       val3->type == &UA_TYPES[UA_TYPES_BYTE]) {
//        value3 = *(UA_Byte*)val3->data;
//        printf("the value of the.answer3 is: %u\n", value3);
//    }
//    UA_Variant_delete(val3);

    /* Toggle and write node attribute for "the.answer3" */
//    value3 = value3 == 0 ? 1 : 0;  // Toggle between 0 and 1
//    UA_Variant *myVariant3 = UA_Variant_new();
//    UA_Variant_setScalarCopy(myVariant3, &value3, &UA_TYPES[UA_TYPES_BYTE]);
//    UA_StatusCode retval = UA_Client_writeValueAttribute(client, UA_NODEID_STRING(1, "the.answer3"), myVariant3);
//    if(retval == UA_STATUSCODE_GOOD) {
//        printf("Successfully wrote %u to the.answer3\n", value3);
//    } else {
//        printf("Failed to write to the.answer3. Status code %s\n", UA_StatusCode_name(retval));
//    }
//    UA_Variant_delete(myVariant3);
   ///
    auto frame_index = 0;
    for(const auto key : *(frame.channels)){

        //check key is in map
        if(this->cmd_channel_map.find(key) == this->cmd_channel_map.end()){
            LOG(ERROR) << "[opc.sink] Channel key not found in map";
            continue;
        }
        auto ch = this->cmd_channel_map[key];

        UA_Variant *val = UA_Variant_new();
        auto data_Type = frame.series->at(frame_index).data_type;
        this->set_variant(val, frame, frame_index, data_Type);
        UA_StatusCode retval = UA_Client_writeValueAttribute(client, ch.node, val);
        if(retval != UA_STATUSCODE_GOOD){
            auto err = this->communicate_response_error(retval);
            UA_Variant_delete(val);
            LOG(ERROR) << "[opc.sink] Failed to write to node: " << key;
            return;
        }
        UA_Variant_delete(val);
        frame_index++;
    }
//    frame.series->at(series_index).data_type;

    return;
    // the number of frames we are writing corresponds to the number of values in the frame


//    UA_WriteRequest_init(&this->req);
//    std::vector<UA_WriteValue> nodes_to_write(frame.channels->size());
//    req.nodesToWriteSize = frame.channels->size();
//
//    for(const auto key : *(frame.channels)){
//        //check key is in map
//        if(this->cmd_channel_map.find(key) == this->cmd_channel_map.end()){
//            LOG(ERROR) << "[opc.sink] Channel key not found in map";
//            continue;
//        }
//        auto ch = this->cmd_channel_map[key];
////        this->initialize_write_value(
////            frame,
////            frame_index,
////            ch,
////            &nodes_to_write[frame_index]
////        );
//        frame_index++;
//    }
//    req.nodesToWrite = nodes_to_write.data();
};

void opc::Sink::set_variant(UA_Variant *val, const synnax::Frame &frame, const uint32_t &series_index, const synnax::DataType &type) {
    UA_StatusCode status = UA_STATUSCODE_GOOD;
    if(type == synnax::FLOAT64){
        double * data = &(frame.series->at(series_index).values<double>())[0];
        UA_Variant_setScalar(val, data, &UA_TYPES[UA_TYPES_DOUBLE]);
    }else if(type == synnax::FLOAT32) {
        float * data = &(frame.series->at(series_index).values<float>())[0];
        UA_Variant_setScalar(val, data, &UA_TYPES[UA_TYPES_FLOAT]);
    }else if(type == synnax::INT32){
        void* data = &(frame.series->at(series_index).values<int32_t>())[0];
        UA_Variant_setScalar(val, data, &UA_TYPES[UA_TYPES_INT32]);
    }else if(type == synnax::INT16) {
        void * data = &(frame.series->at(series_index).values<int16_t>())[0];
        UA_Variant_setScalar(val, data, &UA_TYPES[UA_TYPES_INT16]);
    }else if(type == synnax::INT8){
        void * data = &(frame.series->at(series_index).values<int8_t>())[0];
        UA_Variant_setScalar(val, data, &UA_TYPES[UA_TYPES_SBYTE]);
    }else if(type == synnax::UINT64){
        void * data = &(frame.series->at(series_index).values<uint64_t>())[0];
        UA_Variant_setScalar(val, data, &UA_TYPES[UA_TYPES_UINT64]);
    }else if(type == synnax::UINT32){
        void * data = &(frame.series->at(series_index).values<uint32_t>())[0];
        UA_Variant_setScalar(val, data, &UA_TYPES[UA_TYPES_UINT32]);
    }else if(type == synnax::UINT16){
        void * data = &(frame.series->at(series_index).values<uint16_t>())[0];
        UA_Variant_setScalar(val, data, &UA_TYPES[UA_TYPES_UINT16]);
    }else if(type == synnax::UINT8){
        uint8_t data = frame.series->at(series_index).values<uint8_t>()[0];
        status = UA_Variant_setScalarCopy(val, &data, &UA_TYPES[UA_TYPES_BYTE]);
    }else if(type == synnax::TIMESTAMP){
        void * data = &(frame.series->at(series_index).values<int64_t>())[0];
        UA_Variant_setScalar(val, data, &UA_TYPES[UA_TYPES_DATETIME]);
    }
    if(status != UA_STATUSCODE_GOOD){
        LOG(ERROR) << "[opc.sink] Failed to set variant";
    }
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
//    UA_WriteResponse res = UA_Client_Service_write(this->ua_client.get(), this->req);
//    UA_StatusCode status = res.responseHeader.serviceResult;

//    if(status != UA_STATUSCODE_GOOD){
//        auto err = this->communicate_response_error(status);
//        UA_WriteResponse_clear(&res);
//        return err;
//    }
//    UA_WriteRequest_clear(&this->req);
//    UA_WriteResponse_clear(&res);
    return freighter::NIL;
};
