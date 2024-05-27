// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <set>
#include <memory>
#include <vector>
#include <utility>
#include <queue>
#include <glog/logging.h>
#include "driver/opc/opc.h"
#include "driver/opc/util.h"
#include "driver/task/task.h"
#include "driver/config/config.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/errors/errors.h"

#include "include/open62541/types.h"
#include "include/open62541/types_generated.h"
#include "include/open62541/statuscodes.h"
#include "include/open62541/client_highlevel.h"
#include "driver/pipeline/acquisition.h"
#include "include/open62541/common.h"

namespace opc {

///////////////////////////////////////////////////////////////////////////////////
//                                    WriterConfigs                              //
///////////////////////////////////////////////////////////////////////////////////

typedef struct {
    ///@brief the node id.
    std::string node_id;
    UA_NodeId node;
    ///@brief the corresponding channel key to write the variable for the node to.
    ChannelKey channel;
    ///@brief the channel fetched from the Synnax server. Does not need to be proivded via json
    Channel ch;
    bool enabled;

} WriterChannelConfig;

typedef struct {
    /// @brief the device representing the OPC UA server to write to.
    std::string device;
    /// @brief sets the rate at which states/acknowledgements are written to server
    Rate update_rate;
    /// @brief the list of channels to write to
    std::vector<WriterChannelConfig> channels;
} WriterConfig;


///////////////////////////////////////////////////////////////////////////////////
//                                    WriterSink                                 //
///////////////////////////////////////////////////////////////////////////////////
class WriterSink final : public pipeline::Sink {
public:
    WriterSink(
            std::shared_ptr<task::Context> ctx,
            synnax::Task task,
            WriterConfig cfg,
            const std::shared_ptr<UA_Client> &client,
            std::set<ChannelKey> indexes
            );
    freighter::Error start();
    freighter::Error stop();
    freighter::Error communicateResError(const UA_StatusCode status);
    freighter::Error communicateValueError(const std::string &channel, const UA_StatusCode &status);
    std::vector<synnax::ChannelKey> getCmdChannelKeys();
    std::vector<synnax::ChannelKey> getStateChannelKeys();
    freighter::Error write(synnax::Frame frame);
    void initializeWriteRequest();

private:
    WriterConfig cfg;
    std::shared_ptr<UA_Client> client;
    std::set<ChannelKey> keys;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    task::State curr_state;

    UA_WriteRequest request;
    std::vector<UA_WriteValue> nodes_to_write;

    std::vector<synnax::ChannelKey> state_channel_keys;
    std::vector<synnax::ChannelKey> cmd_channel_keys;

    synnax::ChannelKey drive_state_index_key;
    std::queue<synnax::ChannelKey> cmd_index_keys;
    std::queue<std::uint8_t> modified_state_valies;
};

///////////////////////////////////////////////////////////////////////////////////
//                                Writer Task                                    //
///////////////////////////////////////////////////////////////////////////////////


//class Writer final : public task::Task {
//
//};


} // namespace opc