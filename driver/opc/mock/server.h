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

/// internal
#include "driver/opc/types/types.h"

namespace mock {
struct TestNode {
    std::int32_t ns;
    std::string node_id;
    UA_DataType *data_type;
    UA_Variant initial_value;
    std::string description;
    bool return_invalid_data = false;

    // Constructor
    TestNode(
        std::int32_t ns,
        std::string node_id,
        UA_DataType *data_type,
        const UA_Variant &initial_value,
        std::string description,
        bool return_invalid_data = false
    ):
        ns(ns),
        node_id(std::move(node_id)),
        data_type(data_type),
        description(std::move(description)),
        return_invalid_data(return_invalid_data) {
        UA_Variant_init(&this->initial_value);
        UA_Variant_copy(&initial_value, &this->initial_value);
    }

    // Destructor - clean up the variant
    ~TestNode() { UA_Variant_clear(&initial_value); }

    // Copy constructor
    TestNode(const TestNode &other):
        ns(other.ns),
        node_id(other.node_id),
        data_type(other.data_type),
        description(other.description),
        return_invalid_data(other.return_invalid_data) {
        UA_Variant_init(&initial_value);
        UA_Variant_copy(&other.initial_value, &initial_value);
    }

    // Copy assignment operator
    TestNode &operator=(const TestNode &other) {
        if (this != &other) {
            ns = other.ns;
            node_id = other.node_id;
            data_type = other.data_type;
            description = other.description;
            return_invalid_data = other.return_invalid_data;
            UA_Variant_clear(&initial_value);
            UA_Variant_copy(&other.initial_value, &initial_value);
        }
        return *this;
    }

    // Move constructor
    TestNode(TestNode &&other) noexcept:
        ns(other.ns),
        node_id(std::move(other.node_id)),
        data_type(other.data_type),
        initial_value(other.initial_value),
        description(std::move(other.description)),
        return_invalid_data(other.return_invalid_data) {
        UA_Variant_init(&other.initial_value);
    }

    // Move assignment operator
    TestNode &operator=(TestNode &&other) noexcept {
        if (this != &other) {
            ns = other.ns;
            node_id = std::move(other.node_id);
            data_type = other.data_type;
            description = std::move(other.description);
            return_invalid_data = other.return_invalid_data;
            UA_Variant_clear(&initial_value);
            initial_value = other.initial_value;
            UA_Variant_init(&other.initial_value);
        }
        return *this;
    }
};

struct ServerConfig {
    std::vector<TestNode> test_nodes;
    std::uint16_t port = 4840; // Default OPC UA port

    // Create default test nodes for comprehensive testing
    static ServerConfig create_default() {
        ServerConfig cfg;

        // Static storage for variant data (needed because UA_Variant_setScalar doesn't
        // copy)
        static UA_Boolean bool_data = true;
        static UA_UInt16 uint16_data = 42;
        static UA_UInt32 uint32_data = 12345;
        static UA_UInt64 uint64_data = 12345;
        static UA_SByte int8_data = 42;
        static UA_Int16 int16_data = 42;
        static UA_Int32 int32_data = 12345;
        static UA_Int64 int64_data = 12345;
        static UA_Float float_data = 3.14159f;
        static UA_Double double_data = 2.71828;
        static UA_Guid guid_data = {
            0x12345678,
            0x1234,
            0x5678,
            {0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
        };

        // Boolean node
        UA_Variant bool_val;
        UA_Variant_init(&bool_val);
        UA_Variant_setScalar(&bool_val, &bool_data, &UA_TYPES[UA_TYPES_BOOLEAN]);

        // uint16 node
        UA_Variant uint16_val;
        UA_Variant_init(&uint16_val);
        UA_Variant_setScalar(&uint16_val, &uint16_data, &UA_TYPES[UA_TYPES_UINT16]);

        // uint32 node
        UA_Variant uint32_val;
        UA_Variant_init(&uint32_val);
        UA_Variant_setScalar(&uint32_val, &uint32_data, &UA_TYPES[UA_TYPES_UINT32]);

        // uint64 node
        UA_Variant uint64_val;
        UA_Variant_init(&uint64_val);
        UA_Variant_setScalar(&uint64_val, &uint64_data, &UA_TYPES[UA_TYPES_UINT64]);

        // int8 node (using SByte in OPC UA)
        UA_Variant int8_val;
        UA_Variant_init(&int8_val);
        UA_Variant_setScalar(&int8_val, &int8_data, &UA_TYPES[UA_TYPES_SBYTE]);

        // int16 node
        UA_Variant int16_val;
        UA_Variant_init(&int16_val);
        UA_Variant_setScalar(&int16_val, &int16_data, &UA_TYPES[UA_TYPES_INT16]);

        // int32 node
        UA_Variant int32_val;
        UA_Variant_init(&int32_val);
        UA_Variant_setScalar(&int32_val, &int32_data, &UA_TYPES[UA_TYPES_INT32]);

        // int64 node
        UA_Variant int64_val;
        UA_Variant_init(&int64_val);
        UA_Variant_setScalar(&int64_val, &int64_data, &UA_TYPES[UA_TYPES_INT64]);

        // float node
        UA_Variant float_val;
        UA_Variant_init(&float_val);
        UA_Variant_setScalar(&float_val, &float_data, &UA_TYPES[UA_TYPES_FLOAT]);

        // double node
        UA_Variant double_val;
        UA_Variant_init(&double_val);
        UA_Variant_setScalar(&double_val, &double_data, &UA_TYPES[UA_TYPES_DOUBLE]);

        // guid node
        UA_Variant guid_val;
        UA_Variant_init(&guid_val);
        UA_Variant_setScalar(&guid_val, &guid_data, &UA_TYPES[UA_TYPES_GUID]);

        cfg.test_nodes = {
            {1,
             "TestBoolean",
             &UA_TYPES[UA_TYPES_BOOLEAN],
             bool_val,
             "Test Boolean Node"},
            {1,
             "TestUInt16",
             &UA_TYPES[UA_TYPES_UINT16],
             uint16_val,
             "Test UInt16 Node"},
            {1,
             "TestUInt32",
             &UA_TYPES[UA_TYPES_UINT32],
             uint32_val,
             "Test UInt32 Node"},
            {1,
             "TestUInt64",
             &UA_TYPES[UA_TYPES_UINT64],
             uint64_val,
             "Test UInt64 Node"},
            {1, "TestInt8", &UA_TYPES[UA_TYPES_SBYTE], int8_val, "Test Int8 Node"},
            {1, "TestInt16", &UA_TYPES[UA_TYPES_INT16], int16_val, "Test Int16 Node"},
            {1, "TestInt32", &UA_TYPES[UA_TYPES_INT32], int32_val, "Test Int32 Node"},
            {1, "TestInt64", &UA_TYPES[UA_TYPES_INT64], int64_val, "Test Int64 Node"},
            {1, "TestFloat", &UA_TYPES[UA_TYPES_FLOAT], float_val, "Test Float Node"},
            {1,
             "TestDouble",
             &UA_TYPES[UA_TYPES_DOUBLE],
             double_val,
             "Test Double Node"},
            {1, "TestGuid", &UA_TYPES[UA_TYPES_GUID], guid_val, "Test GUID Node"},
        };
        return cfg;
    }

    // Create a configuration with nodes that return invalid/null data for testing error
    // handling
    static ServerConfig create_with_invalid_data() {
        ServerConfig cfg;

        // Invalid boolean node - null type
        UA_Variant invalid_bool_val;
        UA_Variant_init(&invalid_bool_val);
        invalid_bool_val.type = nullptr;
        invalid_bool_val.data = nullptr;

        // Invalid float node - null data
        UA_Variant invalid_float_val;
        UA_Variant_init(&invalid_float_val);
        invalid_float_val.type = &UA_TYPES[UA_TYPES_FLOAT];
        invalid_float_val.data = nullptr;

        // Invalid double node - zero length array
        UA_Variant invalid_double_val;
        UA_Variant_init(&invalid_double_val);
        invalid_double_val.type = &UA_TYPES[UA_TYPES_DOUBLE];
        invalid_double_val.arrayLength = 0;
        invalid_double_val.data = UA_EMPTY_ARRAY_SENTINEL;

        cfg.test_nodes = {
            {1,
             "InvalidBoolean",
             nullptr,
             invalid_bool_val,
             "Test Invalid Boolean Node",
             true},
            {1,
             "InvalidFloat",
             &UA_TYPES[UA_TYPES_FLOAT],
             invalid_float_val,
             "Test Invalid Float Node",
             true},
            {1,
             "InvalidDouble",
             &UA_TYPES[UA_TYPES_DOUBLE],
             invalid_double_val,
             "Test Invalid Double Node",
             true},
        };
        return cfg;
    }
};

class Server {
public:
    ServerConfig cfg;
    std::atomic<bool> running{false};
    std::thread thread;

    explicit Server(const ServerConfig &cfg): cfg(cfg) {}

    void start() {
        running = true;
        thread = std::thread(&Server::run, this);
    }

    void stop() {
        running = false;
        thread.join();
    }

    ~Server() {
        if (running) this->stop();
    }

    void run() {
        UA_Server *server = UA_Server_new();
        auto server_config = UA_Server_getConfig(server);
        server_config->maxSessionTimeout = 3600000;
        UA_ServerConfig_setMinimal(server_config, cfg.port, nullptr);

        for (const auto &node: cfg.test_nodes) {
            UA_VariableAttributes attr = UA_VariableAttributes_default;

            // Set the variant as the value to be exposed by the server
            attr.value = node.initial_value;

            // Set access level to allow reading and writing
            attr.accessLevel = UA_ACCESSLEVELMASK_READ | UA_ACCESSLEVELMASK_WRITE;

            opc::LocalizedText description("en-US", node.description.c_str());
            opc::LocalizedText displayName("en-US", node.node_id.c_str());
            attr.description = description.get();
            attr.displayName = displayName.get();

            UA_NodeId raw_node_id = UA_NODEID_STRING_ALLOC(
                node.ns,
                node.node_id.c_str()
            );
            opc::NodeId nodeId(raw_node_id);
            UA_NodeId_clear(&raw_node_id);
            LOG(INFO) << "Creating OPC UA node: "
                      << opc::NodeId::to_string(nodeId.get());

            opc::QualifiedName nodeName(node.ns, node.node_id.c_str());
            UA_NodeId parentNodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER);
            UA_NodeId parentReferenceNodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_ORGANIZES);

            UA_Server_addVariableNode(
                server,
                nodeId.get(),
                parentNodeId,
                parentReferenceNodeId,
                nodeName.get(),
                UA_NODEID_NULL,
                attr,
                NULL,
                NULL
            );
        }

        UA_StatusCode status = UA_Server_run_startup(server);
        if (status != UA_STATUSCODE_GOOD) {
            LOG(WARNING) << "Mock OPC UA server stopped with status: "
                         << UA_StatusCode_name(status);
            UA_Server_delete(server);
            return;
        }

        while (running.load())
            UA_Server_run_iterate(server, true);

        UA_Server_run_shutdown(server);
        UA_Server_delete(server);
    }
};
}
