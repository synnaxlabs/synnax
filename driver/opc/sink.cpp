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
//                                    StateSource                                //
///////////////////////////////////////////////////////////////////////////////////
opc::StateSource::StateSource(
    synnax::Rate state_rate,
    const std::shared_ptr<UA_Client> &ua_client,
    const std::shared_ptr<task::Context> &ctx,
    const WriterConfig &cfg
) : state_rate(state_rate),
    timer(state_rate),
    ua_client(ua_client),
    ctx(ctx),
    cfg(cfg),
    state_index_key(cfg.state_index_key){
    // TODO: might move state_index_key initialization
    //  to inside the constructor body

    // read each value from the opc ua server to get initial states
    // and write them to the map

    // create thread to subscribe to the state channel
}

std::pair<synnax::Frame, freighter::Error> opc::StateSource::read(
        breaker::Breaker &breaker){
    this->timer.wait(breaker);
    std::unique_lock<std::mutex> lock(this->state_mutex);
    // sleep for state period
    waiting_reader.wait_for(lock, this->state_rate.period().chrono());
    return std::make_pair(this->get_state(), freighter::NIL);
}

 void opc::StateSource::update_state( const synnax::ChannelKey &channel_key, const UA_Variant &value){
    std::unique_lock<std::mutex> lock(this->state_mutex);
    this->state_map[channel_key] = value;
    waiting_reader.notify_all();
 }

 synnax::Frame opc::StateSource::get_state() {
   // TODO: parse through map and write the states
     // frame size = # monitored states + 1 state index channel
   auto state_frame = synnax::Frame(this->state_map.size() + 1);
   state_frame.add(
           this->state_index_key,
           synnax::Series(
                   synnax::TimeStamp::now().value,
                   synnax::TIMESTAMP
                   )
               );
   for (auto &[key,value] : this->state_map){


   }
}


UA_StatusCode opc::StateSource::add_monitored_item(const UA_NodeId& node_id, const synnax::ChannelKey& channel_key){
    UA_MonitoredItemCreateRequest mon_request = UA_MonitoredItemCreateRequest_default(node_id);

    auto context = std::make_unique<MonitoredItemContext>();
    context->source = this;
    context->channelKey = channel_key;

    UA_MonitoredItemCreateResult mon_response = UA_Client_MonitoredItems_createDataChange(
        this->ua_client.get(),
        this->subscription_id,
        UA_TIMESTAMPSTORETURN_BOTH,
        mon_request,
        context.get(),
        data_change_handler,
        NULL
    );

    if(mon_response.statusCode != UA_STATUSCODE_GOOD) return mon_response.statusCode;

    return UA_STATUSCODE_GOOD;
}

 static void data_change_handler(
    UA_Client *client,
    UA_UInt32 subId,
    void *subContext,
    UA_UInt32 monId,
    void *monContext,
    UA_DataValue *value){

    if(monContext && value){
        auto context = static_cast<opc::StateSource::MonitoredItemContext*>(monContext);
        auto source = context->source;
        auto key = context->channelKey;
        source->update_state(key, value->value);
    }
}

// TODO: is a subset fo the same function in reader.cpp code DEDUP
size_t write_to_series(
        const UA_Variant *val,
        const size_t i,
        synnax::Series &s
) {
    if (val->type == &UA_TYPES[UA_TYPES_FLOAT]) {
        const auto value = *static_cast<UA_Float *>(val->data);
        if (s.data_type == synnax::FLOAT32) return s.write(value);
        if (s.data_type == synnax::FLOAT64) return s.write(
                    static_cast<double>(value));
        if (s.data_type == synnax::INT32) return s.write(
                    static_cast<int32_t>(value));
        if (s.data_type == synnax::INT64) return s.write(
                    static_cast<int64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_DOUBLE]) {
        const auto value = *static_cast<UA_Double *>(val->data);
        if (s.data_type == synnax::FLOAT32) return s.write(
                    static_cast<float>(value));
        if (s.data_type == synnax::FLOAT64) return s.write(value);
        if (s.data_type == synnax::INT32) return s.write(
                    static_cast<int32_t>(value));
        if (s.data_type == synnax::INT64) return s.write(
                    static_cast<int64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT16]) {
        const auto value = *static_cast<UA_Int16 *>(val->data);
        if(s.data_type == synnax::INT16) return s.write(value);
        if (s.data_type == synnax::INT32) return s.write(
                    static_cast<int16_t>(value));
        if (s.data_type == synnax::INT64) return s.write(
                    static_cast<int64_t>(value));
        if (s.data_type == synnax::UINT16) return s.write(
                    static_cast<uint16_t>(value));
        if (s.data_type == synnax::UINT32) return s.write(
                    static_cast<uint32_t>(value));
        if (s.data_type == synnax::UINT64) return s.write(
                    static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT32]) {
        const auto value = *static_cast<UA_Int32 *>(val->data);
        if (s.data_type == synnax::INT32) return s.write(value);
        if (s.data_type == synnax::INT64) return s.write(
                    static_cast<int64_t>(value));
        if (s.data_type == synnax::UINT32) return s.write(
                    static_cast<uint32_t>(value));
        if (s.data_type == synnax::UINT64) return s.write(
                    static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT64]) {
        const auto value = *static_cast<UA_Int64 *>(val->data);
        if (s.data_type == synnax::INT32) return s.write(
                    static_cast<int32_t>(value));
        if (s.data_type == synnax::INT64) return s.write(value);
        if (s.data_type == synnax::UINT32) return s.write(
                    static_cast<uint32_t>(value));
        if (s.data_type == synnax::UINT64) return s.write(
                    static_cast<uint64_t>(value));
        if (s.data_type == synnax::TIMESTAMP)
            return s.write(static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_UINT32]) {
        const auto value = *static_cast<UA_UInt32 *>(val->data);
        if (s.data_type == synnax::INT32) return s.write(
                    static_cast<int32_t>(value));
        // Potential data loss
        if (s.data_type == synnax::INT64) return s.write(
                    static_cast<int64_t>(value));
        if (s.data_type == synnax::UINT32) return s.write(value);
        if (s.data_type == synnax::UINT64) return s.write(
                    static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_UINT64]) {
        const auto value = *static_cast<UA_UInt64 *>(val->data);
        if (s.data_type == synnax::UINT64) return s.write(value);
        if (s.data_type == synnax::INT32) return s.write(
                    static_cast<int32_t>(value));
        // Potential data loss
        if (s.data_type == synnax::INT64) return s.write(
                    static_cast<int64_t>(value));
        if (s.data_type == synnax::UINT32) return s.write(
                    static_cast<uint32_t>(value));
        // Potential data loss
        if (s.data_type == synnax::TIMESTAMP)
            s.write(static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_BYTE]) {
        const auto value = *static_cast<UA_Byte *>(val->data);
        if (s.data_type == synnax::UINT8) return s.write(value);
        if (s.data_type == synnax::UINT16) return s.write(
                    static_cast<uint16_t>(value));
        if (s.data_type == synnax::UINT32) return s.write(
                    static_cast<uint32_t>(value));
        if (s.data_type == synnax::UINT64) return s.write(
                    static_cast<uint64_t>(value));
        if (s.data_type == synnax::INT8) return s.write(static_cast<int8_t>(value));
        if (s.data_type == synnax::INT16) return s.write(
                    static_cast<int16_t>(value));
        if (s.data_type == synnax::INT32) return s.write(
                    static_cast<int32_t>(value));
        if (s.data_type == synnax::INT64) return s.write(
                    static_cast<int64_t>(value));
        if (s.data_type == synnax::FLOAT32) return s.write(
                    static_cast<float>(value));
        if (s.data_type == synnax::FLOAT64) return s.write(
                    static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_SBYTE]) {
        const auto value = *static_cast<UA_SByte *>(val->data);
        if (s.data_type == synnax::INT8) return s.write(value);
        if (s.data_type == synnax::INT16) return s.write(
                    static_cast<int16_t>(value));
        if (s.data_type == synnax::INT32) return s.write(
                    static_cast<int32_t>(value));
        if (s.data_type == synnax::INT64) return s.write(
                    static_cast<int64_t>(value));
        if (s.data_type == synnax::FLOAT32) return s.write(
                    static_cast<float>(value));
        if (s.data_type == synnax::FLOAT64) return s.write(
                    static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_BOOLEAN]) {
        const auto value = *static_cast<UA_Boolean *>(val->data);
        if (s.data_type == synnax::UINT8) return s.write(
                    static_cast<uint8_t>(value));
        if (s.data_type == synnax::UINT16) return s.write(
                    static_cast<uint16_t>(value));
        if (s.data_type == synnax::UINT32) return s.write(
                    static_cast<uint32_t>(value));
        if (s.data_type == synnax::UINT64) return s.write(
                    static_cast<uint64_t>(value));
        if (s.data_type == synnax::INT8) return s.write(static_cast<int8_t>(value));
        if (s.data_type == synnax::INT16) return s.write(
                    static_cast<int16_t>(value));
        if (s.data_type == synnax::INT32) return s.write(
                    static_cast<int32_t>(value));
        if (s.data_type == synnax::INT64) return s.write(
                    static_cast<int64_t>(value));
        if (s.data_type == synnax::FLOAT32) return s.write(
                    static_cast<float>(value));
        if (s.data_type == synnax::FLOAT64) return s.write(
                    static_cast<double>(value));
    }
    LOG(ERROR) << "[opc.reader] unsupported data type: " << val->type->typeName;
}


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
