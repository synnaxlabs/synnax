// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
// File added by Elham Islam

#include "driver/opc/writer.h"

using namespace opc;

WriterChannelConfig::WriterChannelConfig(
    config::Parser &parser
) : node_id(parser.required<std::string>("node_id")),
    node(parseNodeId("node_id", parser)),
    channel(parser.required<ChannelKey>("channel")),
    enabled(parser.optional<bool>("enabled", true)) {
}

opc::WriterConfig::WriterConfig(
    config::Parser &parser
) : device(parser.required<std::string>("device")),
    update_rate(parser.required<std::float_t>("update_rate")){

    parser.iter("channels", [&](config::Parser &channel_builder) {
        auto channel = WriterChannelConfig(channel_builder);
        if (channel.enabled) channels.push_back(channel);
    });
}

///////////////////////////////////////////////////////////////////////////////////
//                                    WriterSink                                 //
///////////////////////////////////////////////////////////////////////////////////
WriterSink::WriterSink(
        std::shared_ptr<task::Context> ctx,
        synnax::Task task,
//        WriterConfig cfg,
        const std::shared_ptr<UA_Client> &client,
//        std::set<ChannelKey> indexes
) : client(client),
    ctx(std::move(ctx)),
    task(std::move(task)){

    this->initializeWriteRequest(); // TODO: IMPL

    this->curr_state.task = this->task.key;
    this->curr_state.variant = "success";
    this->curr_state.details = json{
        {"message", "Task configured successfully"},
        {"running", true}
    };
    this->ctx->setState(this->curr_state);
}

void WriterSink::ParseConfig(config::Parser &parser){
    this->cfg = WriterConfig(parser);

    assert(parser.ok())
}



freighter::Error WriterSink::start(){
    this->curr_state.variant = "success";
    this->curr_state.details = json{
            {"message", "Task started successfully"},
            {"running", true}
    };
    this->ctx->setState(curr_state);
    return freighter::NIL;
}

freighter::Error WriterSink::stop(){
    this->curr_state.variant = "success";
    this->curr_state.details = json{
            {"message", "Task stopped successfully"},
            {"running", false}
    };
    this->ctx->setState(curr_state);
    return freighter::NIL;
}

void WriterSink::initializeWriteRequest(){

    UA_WriteRequest_init(&this->request); // allocates memory for request

    // Write request has
    // 1. rwuest header             -> no work for this
    // 2. nodes to write            -> array of UA_WriteValues which we need to prepare
    // 3. nodes to write size

    // UA_WriteValue has
    // 1. NodeId                     -> the node we care about
    // 2. AttributeId                -> the attribute of that node
    // 3. IndexRange                 -> optional field used when attribute is array and we only want to write a subset of values in that array (if empty, the entire array is written to)
    // 4. Value                      -> the value to write to the node

    // We need to prepare the nodes_to_write array with the required values
    // needs to happen every time we get a call to write from ctrl pipeline with a new frame to write

    return;
}

freighter::Error WriterSink::write(synnax::Frame frame){
    return freighter::NIL;
}



///////////////////////////////////////////////////////////////////////////////////
//TODO:



std::vector<synnax::ChannelKey> WriterSink::getCmdChannelKeys(){
    return std::vector<synnax::ChannelKey>();
}

std::vector<synnax::ChannelKey> WriterSink::getStateChannelKeys(){
    return std::vector<synnax::ChannelKey>();
}

freighter::Error WriterSink::communicateResError(const UA_StatusCode status){
    return freighter::NIL;
}
freighter::Error WriterSink::communicateValueError(const std::string &channel, const UA_StatusCode &status){
    return freighter::NIL;
}

