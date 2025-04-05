// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* This work is licensed under a Creative Commons CCZero 1.0 Universal License.
 * See http://creativecommons.org/publicdomain/zero/1.0/ for more information. */

#pragma once

/// std
#include <atomic>
#include <string>
#include <thread>

/// external
#include "glog/logging.h"
#include "open62541/server.h"
#include "open62541/server_config_default.h"

/// module
#include "client/cpp/channel/channel.h"

/// internal
#include "driver/opc/util/util.h"


namespace mock {
struct ServerChannel {
    std::int32_t ns;
    std::string node;
    synnax::Channel ch;
};

struct ServerConfig {
    std::vector<ServerChannel> channels;
};

class Server {
public:
    ServerConfig cfg;
    volatile bool *running = new bool(false);
    std::thread thread;

    explicit Server(const ServerConfig &cfg): cfg(cfg) {}

    void start() {
        running = new bool(true);
        thread = std::thread(&Server::run, this);
    }

    void stop() {
        *running = false;
        thread.join();
    }

    ~Server() {
        if (*running) this->stop();
    }

    void run() const {
        UA_Server *server = UA_Server_new();
        auto server_config = UA_Server_getConfig(server);
        server_config->maxSessionTimeout = 3600000;
        UA_ServerConfig_setDefault(server_config);


        for (auto &ch: cfg.channels) {
            UA_VariableAttributes attr = UA_VariableAttributes_default;
            auto [variant, err] = util::series_to_variant(
                telem::Series(ch.ch.data_type.cast(5))
            );

            // Set the variant as the value to be exposed by the server
            attr.value = variant;

            // Set access level to allow reading and writing
            attr.accessLevel = UA_ACCESSLEVELMASK_READ | UA_ACCESSLEVELMASK_WRITE;

            attr.description = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer");
            attr.displayName = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer");
            UA_NodeId myIntegerNodeId = UA_NODEID_STRING_ALLOC(ch.ns, ch.node.c_str());
            LOG(INFO) << util::node_id_to_string(myIntegerNodeId);
            UA_QualifiedName myIntegerName = UA_QUALIFIEDNAME_ALLOC(
                ch.ns,
                ch.node.c_str()
            );
            UA_NodeId parentNodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER);
            UA_NodeId parentReferenceNodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_ORGANIZES);
            UA_Server_addVariableNode(
                server,
                myIntegerNodeId,
                parentNodeId,
                parentReferenceNodeId,
                myIntegerName,
                UA_NODEID_NULL,
                attr,
                NULL,
                NULL
            );
        }
        UA_StatusCode retval = UA_Server_run(server, running);
        UA_Server_delete(server);
    }
};
} 
