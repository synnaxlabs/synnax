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

#include "include/open62541/types.h"
#include "include/open62541/types_generated.h"
#include "include/open62541/statuscodes.h"
#include "include/open62541/client_highlevel.h"
#include "include/open62541/common.h"


opc::Sink::Sink(
    WriterConfig cfg,
    const std::shared_ptr<UA_Client> &ua_client,
    std::set<ChannelKey> indexes,
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
void opc::Sink::initializeWriteRequest(const synnax::Frame &frame){

    // the number of frames we are writing corresponds to the number of values in the frame
    this->nodes_to_write.reserve(frames.channels->size());
    uint32_t frame_index = 0;

    UA_WriteRequest_init(&this->req);
    req.nodesToWriteSize = frame.channels->size();
    
    for(const auto key : *(frame.channels)){
        // get channel config using key
        auto ch = this->channel_map[key];
        this->initializerWriteValue(    
            frame, 
            frame_index, 
            ch, 
            this->nodes_to_write[frame_index]
        );
        
        frame_index++;
    }
};

UA_DataValue opc::initializeWriteValue( const synnax::Frame &frame,
                                        uint32_t &index,
                                        WriterChannelConfig &ch,
                                        UA_WriteValue &write_value){
    write_value = UA_WriteValue_new();
    write_value.nodeId = ch.node;
    write_value.attributeId = UA_ATTRIBUTEID_VALUE;
    write_value.value.hasValue = true; // TODO what is this
    write_value.value.storageType = UA_VARIANT_DATA_NODELETE; // do not free integer on deletion

    write_value.value.type = &UA_TYPES[UA_TYPES_INT32];// GET THIS IN A SEC;
    write_value.value.data = &value; // need to cast this to the correct type
}

// Sets the appropriate type for the write value and casts the series data as necessary to one of the supported types
// on Synnax
void opc::Sink::CastAndSetType( const synnax::Frame &frame,
                                uint32_t &series_index,
                                WriterChannelConfig &ch,
                                UA_WriteValue &write_value){
    auto &type = frame.series->at(series_index).data_type;

    // TODO: Need to test if I could specify writing float32 if the actual node type is float64 or real, etc
    if(type == synnax::FLOAT64) {
        write_value.value.data = &frame.series->at(series_index).values<double>()[0];
        write_value.value.type = &UA_TYPES[UA_TYPES_DOUBLE];
    }else if(type == synnax::FLOAT32){
        write_value.value.data = &frame.series->at(series_index).values<float>()[0];;
        write_value.value.type = &UA_TYPES[UA_TYPES_FLOAT];
    }
    else if(type == synnax::INT32){
        write_value.value.data = &frame.series->at(series_index).values<int32_t>()[0];
        write_value.value.type = &UA_TYPES[UA_TYPES_INT32];
    }
    else if(type == synnax::INT16){
        write_value.value.data = &frame.series->at(series_index).values<int16_t>()[0];
        write_value.value.type = &UA_TYPES[UA_TYPES_INT16];
    }
    else if(type == synnax::INT8){
        write_value.value.data = &frame.series->at(series_index).values<int8_t>()[0];
        write_value.value.type = &UA_TYPES[UA_TYPES_INT8];
    }
    else if(type == synnax::UINT64){
        write_value.value.data = &frame.series->at(series_index).values<uint64_t>()[0];
        write_value.value.type = &UA_TYPES[UA_TYPES_UINT64];
    }
    else if(type == synnax::UINT32){
        write_value.value.data = &frame.series->at(series_index).values<uint32_t>()[0];
        write_value.value.type = &UA_TYPES[UA_TYPES_UINT32];
    }
    else if(type == synnax::UINT16){
        write_value.value.data = &frame.series->at(series_index).values<uint16_t>()[0];
        write_value.value.type = &UA_TYPES[UA_TYPES_UINT16];
    }
    else if(type == synnax::UINT8){
        write_value.value.data = &frame.series->at(series_index).values<uint8_t>()[0];
        write_value.value.type = &UA_TYPES[UA_TYPES_UINT8];
    }
    else if(type == synnax::TIMESTAMP){
        write_value.value.data = &frame.series->at(series_index).values<int64_t>()[0];
        write_value.value.type = &UA_TYPES[UA_TYPES_DATETIME];
    }
//     else if(type == synnax::UINT128){ TODO: Compiler can't find uint128_t
//         write_value.value.data = &frame.series->at(series_index).values<uint128_t>()[0];
//        write_value.value.type = &UA_TYPES[UA_TYPES_UINT128];
//     }

        LOG(ERROR) << "[opc.sink] Unsupported data type " << type;
}

void stoppedWithErr(const freighter::Error &err){

};

freighter::Error communicateResponseError(const UA_StatusCode &status){

};

freighter::Error communciateValueError(const std::string &channel, const UA_StatusCode &status){

};

freighter::Error write(synnax::Frame frame){

};
