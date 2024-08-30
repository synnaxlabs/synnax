// Copyright 2024 Synnax Labs, Inc.
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

#include <string>
#include <atomic>
#include <thread>
#include <open62541/server.h>

using namespace std;

struct MockServerChannel {
    std::int32_t ns;
    std::string node;
};

struct MockServerConfig {
    std::vector<MockServerChannel> channels;
};

class MockServer {
public:
    MockServerConfig cfg;
    volatile bool *running = new bool(false);
    std::thread thread;

    explicit MockServer(const MockServerConfig &cfg) : cfg(cfg) {
    }

    void start() {
        running = new bool(true);
        thread = std::thread(&MockServer::run, this);
    }

    void stop() {
        *running = false;
        thread.join();
    }

    void run() const {
        UA_Server *server = UA_Server_new();

        for (auto &ch: cfg.channels) {
            UA_VariableAttributes attr = UA_VariableAttributes_default;
            UA_Int32 myInteger = 42;
            UA_Variant_setScalarCopy(&attr.value, &myInteger,
                                     &UA_TYPES[UA_TYPES_INT32]);
            attr.description = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer");
            attr.displayName = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer");
            UA_NodeId myIntegerNodeId = UA_NODEID_STRING_ALLOC(ch.ns, ch.node.c_str());
            UA_QualifiedName myIntegerName = UA_QUALIFIEDNAME_ALLOC(
                    ch.ns, ch.node.c_str());
            UA_NodeId parentNodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER);
            UA_NodeId parentReferenceNodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_ORGANIZES);
            UA_Server_addVariableNode(server, myIntegerNodeId, parentNodeId,
                                      parentReferenceNodeId, myIntegerName,
                                      UA_NODEID_NULL, attr, NULL, NULL);
        }
        UA_StatusCode retval = UA_Server_run(server, running);
        UA_Server_delete(server);
    }
};
