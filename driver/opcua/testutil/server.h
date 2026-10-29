// Copyright 2026 Synnax Labs, Inc.
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

#include <array>
#include <atomic>
#include <chrono>
#include <cstddef>
#include <mutex>
#include <optional>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "absl/log/log.h"
#include "open62541/plugin/certificategroup_default.h"
#include "open62541/server.h"
#include "open62541/server_config_default.h"

#include "x/cpp/telem/telem.h"

#include "driver/opcua/certificate/certificate.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua::testutil {
/// @brief the length of the TestDoubleArray node, which array mode tasks must match.
constexpr std::size_t DOUBLE_ARRAY_SIZE = 5;

struct TestNode {
    std::int32_t ns;
    std::string node_id;
    UA_DataType *data_type;
    UA_Variant initial_value{};
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

/// @brief a value of the Point structure type of the mock server.
struct Point {
    UA_Float x;
    UA_Float y;
};

/// @brief a value of the Motor structure type of the mock server.
struct Motor {
    UA_Double speed;
    UA_Boolean running;
    Point position;
};

/// @brief the starting values of the variables of the NS=1;S=Pump object.
struct Pump {
    UA_Double speed = 0;
    std::string mode = "manual";
};

/// @brief the node of the Pump object, an event notifier.
const UA_NodeId PUMP = UA_NODEID_STRING(1, const_cast<char *>("Pump"));
/// @brief the method of the Pump object that sets its Speed.
const UA_NodeId PUMP_SET_SPEED = UA_NODEID_STRING(
    1,
    const_cast<char *>("Pump.SetSpeed")
);
/// @brief the method of the Pump object that sets its Mode and Speed.
const UA_NodeId PUMP_CONFIGURE = UA_NODEID_STRING(
    1,
    const_cast<char *>("Pump.Configure")
);
/// @brief the method of the Pump object that always fails.
const UA_NodeId PUMP_FAIL = UA_NODEID_STRING(1, const_cast<char *>("Pump.Fail"));
/// @brief an event type of the mock server: a subtype of BaseEventType.
const UA_NodeId TEST_EVENT_TYPE = UA_NODEID_STRING(
    1,
    const_cast<char *>("TestEventType")
);

/// @brief an event that the mock server emits.
struct Event {
    /// @brief the node that emits the event: the Server object, or a node in the
    /// Objects folder.
    UA_NodeId source = UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER);
    /// @brief BaseEventType or TEST_EVENT_TYPE.
    UA_NodeId type = UA_NODEID_NUMERIC(0, UA_NS0ID_BASEEVENTTYPE);
    UA_UInt16 severity = 500;
    std::string message;
    /// @brief when the event occurred. Without one, the server uses its clock.
    std::optional<x::telem::TimeStamp> time;
    /// @brief the value of the event field 1:Level.
    std::optional<UA_Double> level;
};

/// @brief the numeric IDs, in namespace 1, of the Point and Motor data types and of
/// their binary encodings.
constexpr UA_UInt32 POINT_TYPE_ID = 4242;
constexpr UA_UInt32 POINT_ENCODING_ID = 4243;
constexpr UA_UInt32 MOTOR_TYPE_ID = 4244;
constexpr UA_UInt32 MOTOR_ENCODING_ID = 4245;

/// @brief the open62541 descriptions of Point and Motor. Motor points at Point, so the
/// descriptions never move.
class StructureTypes {
    std::array<UA_DataTypeMember, 2> point_members{};
    std::array<UA_DataTypeMember, 3> motor_members{};
    std::array<UA_DataType, 2> types{};

    static void set_member(
        UA_DataTypeMember &m,
        const char *name,
        const UA_DataType *type,
        const std::size_t padding
    ) {
        m.memberName = name;
        m.memberType = type;
        m.padding = static_cast<UA_Byte>(padding);
    }

    static void set_type(
        UA_DataType &t,
        const char *name,
        const UA_UInt32 id,
        const UA_UInt32 encoding_id,
        const std::size_t size,
        UA_DataTypeMember *members,
        const std::size_t members_size
    ) {
        t.typeName = name;
        t.typeId = UA_NODEID_NUMERIC(1, id);
        t.binaryEncodingId = UA_NODEID_NUMERIC(1, encoding_id);
        t.xmlEncodingId = UA_NODEID_NULL;
        t.memSize = static_cast<UA_UInt32>(size);
        t.typeKind = UA_DATATYPEKIND_STRUCTURE;
        t.pointerFree = true;
        t.overlayable = false;
        t.membersSize = static_cast<UA_UInt32>(members_size);
        t.members = members;
    }

public:
    /// @brief the list to set as the customDataTypes of a server.
    UA_DataTypeArray array{};

    StructureTypes() {
        const auto *f = &UA_TYPES[UA_TYPES_FLOAT];
        set_member(this->point_members[0], "X", f, 0);
        set_member(
            this->point_members[1],
            "Y",
            f,
            offsetof(Point, y) - offsetof(Point, x) - sizeof(UA_Float)
        );
        set_type(
            this->types[0],
            "Point",
            POINT_TYPE_ID,
            POINT_ENCODING_ID,
            sizeof(Point),
            this->point_members.data(),
            this->point_members.size()
        );
        set_member(this->motor_members[0], "Speed", &UA_TYPES[UA_TYPES_DOUBLE], 0);
        set_member(
            this->motor_members[1],
            "Running",
            &UA_TYPES[UA_TYPES_BOOLEAN],
            offsetof(Motor, running) - offsetof(Motor, speed) - sizeof(UA_Double)
        );
        set_member(
            this->motor_members[2],
            "Position",
            &this->point(),
            offsetof(Motor, position) - offsetof(Motor, running) - sizeof(UA_Boolean)
        );
        set_type(
            this->types[1],
            "Motor",
            MOTOR_TYPE_ID,
            MOTOR_ENCODING_ID,
            sizeof(Motor),
            this->motor_members.data(),
            this->motor_members.size()
        );
        this->array = {
            .next = nullptr,
            .typesSize = this->types.size(),
            .types = this->types.data(),
            .cleanup = false,
        };
    }

    StructureTypes(const StructureTypes &) = delete;
    StructureTypes &operator=(const StructureTypes &) = delete;
    StructureTypes(StructureTypes &&) = delete;
    StructureTypes &operator=(StructureTypes &&) = delete;
    ~StructureTypes() = default;

    [[nodiscard]] const UA_DataType &point() const { return this->types[0]; }
    [[nodiscard]] const UA_DataType &motor() const { return this->types[1]; }
};

/// @brief the ApplicationUri of the mock server, which its certificate must carry.
const std::string SERVER_APPLICATION_URI = "urn:open62541.server.application";

/// @brief makes a certificate for the mock server.
inline std::pair<certificate::Identity, x::errors::Error>
create_server_identity(const x::telem::TimeSpan validity = x::telem::DAY) {
    return certificate::create(
        SERVER_APPLICATION_URI,
        "Synnax Test Server",
        "localhost",
        validity
    );
}

struct ServerConfig {
    std::vector<TestNode> test_nodes;
    std::uint16_t port = 4840; // Default OPC UA port
    std::uint16_t max_sessions = 0; // 0 = unlimited (open62541 default)
    /// @brief the certificate and key that the server presents. Without one, the
    /// server offers only the None security policy.
    std::optional<certificate::Identity> identity;
    /// @brief the client certificates that the server trusts. Empty trusts every
    /// client.
    std::vector<certificate::Certificate> trusted_clients;
    /// @brief the namespace URIs that the server adds after its own, in order. The
    /// first gets index 2.
    std::vector<std::string> namespaces;
    /// @brief the most nodes that one Read request can name. 0 means no limit.
    std::uint32_t max_nodes_per_read = 0;
    /// @brief the most nodes that one Write request can name. 0 means no limit.
    std::uint32_t max_nodes_per_write = 0;
    /// @brief the most nodes that one Browse request can name. 0 means no limit.
    std::uint32_t max_nodes_per_browse = 0;
    /// @brief the most children that the server returns for a node before it returns a
    /// continuation point. 0 means no limit.
    std::uint32_t max_references_per_node = 0;
    /// @brief the number of Double variables V0, V1, ... in the NS=1;S=Generated
    /// folder. The folder also holds NS=1;S=Generated.Inner, a folder that holds the
    /// variable Deep and organizes the Generated folder again. The server adds the
    /// folders only when this or generated_folders is more than 0.
    std::uint32_t generated_variables = 0;
    /// @brief the number of empty folders F0, F1, ... in the NS=1;S=Generated folder.
    std::uint32_t generated_folders = 0;
    /// @brief the value of the NS=1;S=TestMotor node, of the Motor data type. The
    /// server adds the node only when it is set.
    std::optional<Motor> motor;
    /// @brief the starting values of the PUMP object. It holds the variables
    /// NS=1;S=Pump.Speed and NS=1;S=Pump.Mode, and the methods PUMP_SET_SPEED(speed),
    /// PUMP_CONFIGURE(mode, speed), and PUMP_FAIL(). The server adds the object only
    /// when it is set.
    std::optional<Pump> pump;

    // Create default test nodes for comprehensive testing
    static ServerConfig create_default() {
        ServerConfig cfg;

        UA_Boolean bool_data = true;
        UA_UInt16 uint16_data = 42;
        UA_UInt32 uint32_data = 12345;
        UA_UInt64 uint64_data = 12345;
        UA_SByte int8_data = 42;
        UA_Int16 int16_data = 42;
        UA_Int32 int32_data = 12345;
        UA_Int64 int64_data = 12345;
        UA_Float float_data = 3.14159f;
        UA_Double double_data = 2.71828;
        UA_Guid guid_data = {
            0x12345678,
            0x1234,
            0x5678,
            {0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
        };

        // Boolean node
        UA_Variant bool_val;
        UA_Variant_init(&bool_val);
        UA_Variant_setScalarCopy(&bool_val, &bool_data, &UA_TYPES[UA_TYPES_BOOLEAN]);

        // uint16 node
        UA_Variant uint16_val;
        UA_Variant_init(&uint16_val);
        UA_Variant_setScalarCopy(&uint16_val, &uint16_data, &UA_TYPES[UA_TYPES_UINT16]);

        // uint32 node
        UA_Variant uint32_val;
        UA_Variant_init(&uint32_val);
        UA_Variant_setScalarCopy(&uint32_val, &uint32_data, &UA_TYPES[UA_TYPES_UINT32]);

        // uint64 node
        UA_Variant uint64_val;
        UA_Variant_init(&uint64_val);
        UA_Variant_setScalarCopy(&uint64_val, &uint64_data, &UA_TYPES[UA_TYPES_UINT64]);

        // int8 node (using SByte in OPC UA)
        UA_Variant int8_val;
        UA_Variant_init(&int8_val);
        UA_Variant_setScalarCopy(&int8_val, &int8_data, &UA_TYPES[UA_TYPES_SBYTE]);

        // int16 node
        UA_Variant int16_val;
        UA_Variant_init(&int16_val);
        UA_Variant_setScalarCopy(&int16_val, &int16_data, &UA_TYPES[UA_TYPES_INT16]);

        // int32 node
        UA_Variant int32_val;
        UA_Variant_init(&int32_val);
        UA_Variant_setScalarCopy(&int32_val, &int32_data, &UA_TYPES[UA_TYPES_INT32]);

        // int64 node
        UA_Variant int64_val;
        UA_Variant_init(&int64_val);
        UA_Variant_setScalarCopy(&int64_val, &int64_data, &UA_TYPES[UA_TYPES_INT64]);

        // float node
        UA_Variant float_val;
        UA_Variant_init(&float_val);
        UA_Variant_setScalarCopy(&float_val, &float_data, &UA_TYPES[UA_TYPES_FLOAT]);

        // double node
        UA_Variant double_val;
        UA_Variant_init(&double_val);
        UA_Variant_setScalarCopy(&double_val, &double_data, &UA_TYPES[UA_TYPES_DOUBLE]);

        // double array node
        UA_Double double_array_data[DOUBLE_ARRAY_SIZE] = {1.0, 2.0, 3.0, 4.0, 5.0};
        UA_Variant double_array_val;
        UA_Variant_init(&double_array_val);
        UA_Variant_setArrayCopy(
            &double_array_val,
            double_array_data,
            DOUBLE_ARRAY_SIZE,
            &UA_TYPES[UA_TYPES_DOUBLE]
        );

        // guid node
        UA_Variant guid_val;
        UA_Variant_init(&guid_val);
        UA_Variant_setScalarCopy(&guid_val, &guid_data, &UA_TYPES[UA_TYPES_GUID]);

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
            {1,
             "TestDoubleArray",
             &UA_TYPES[UA_TYPES_DOUBLE],
             double_array_val,
             "Test Double Array Node"},
            {1, "TestGuid", &UA_TYPES[UA_TYPES_GUID], guid_val, "Test GUID Node"},
        };

        // Clean up the local variants - TestNode constructor copies them
        UA_Variant_clear(&bool_val);
        UA_Variant_clear(&uint16_val);
        UA_Variant_clear(&uint32_val);
        UA_Variant_clear(&uint64_val);
        UA_Variant_clear(&int8_val);
        UA_Variant_clear(&int16_val);
        UA_Variant_clear(&int32_val);
        UA_Variant_clear(&int64_val);
        UA_Variant_clear(&float_val);
        UA_Variant_clear(&double_val);
        UA_Variant_clear(&double_array_val);
        UA_Variant_clear(&guid_val);

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
        // NOLINTNEXTLINE(cppcoreguidelines-pro-type-cstyle-cast): open62541 macro.
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
    std::atomic<bool> ready{false};
    std::thread thread;
    /// @brief artificial delay injected before each server iterate cycle.
    /// Used in tests to simulate a slow server (e.g. network-unreachable PLC).
    /// Zero means no delay.
    x::telem::TimeSpan probe_delay;
    /// @brief the secure channels open on the server as of its last iterate.
    std::atomic<size_t> channels{0};
    /// @brief the Sessions that the server created since it started.
    std::atomic<size_t> sessions_created{0};

    explicit Server(ServerConfig cfg): cfg(std::move(cfg)) {}

    void start() {
        running = true;
        thread = std::thread(&Server::run, this);
    }

    /// @brief blocks until the server is ready to accept connections or timeout
    /// expires.
    /// @param timeout maximum time to wait for server readiness.
    /// @return true if server is ready, false if timeout expired.
    [[nodiscard]] bool wait_until_ready(
        std::chrono::milliseconds timeout = std::chrono::milliseconds(5000)
    ) const {
        const auto start = std::chrono::steady_clock::now();
        while (!ready.load() && running.load()) {
            if (std::chrono::steady_clock::now() - start >= timeout) return false;
            std::this_thread::sleep_for(std::chrono::milliseconds(10));
        }
        return ready.load();
    }

    /// @brief makes the server emit event on its next cycle. Safe to call from any
    /// thread.
    void emit(Event event) {
        std::lock_guard lock(this->events_mutex);
        this->events.push_back(std::move(event));
    }

    void stop() {
        running = false;
        if (thread.joinable()) thread.join();
        ready = false;
    }

    ~Server() {
        if (running) this->stop();
    }
    /// @brief adds a variable named name, of type, to the Pump object.
    static void add_pump_variable(
        UA_Server *server,
        const char *name,
        const void *value,
        const UA_DataType &type
    ) {
        UA_VariableAttributes attr = UA_VariableAttributes_default;
        UA_Variant_setScalar(&attr.value, const_cast<void *>(value), &type);
        attr.dataType = type.typeId;
        attr.valueRank = UA_VALUERANK_SCALAR;
        attr.accessLevel = UA_ACCESSLEVELMASK_READ;
        const auto id = std::string("Pump.") + name;
        UA_Server_addVariableNode(
            server,
            UA_NODEID_STRING(1, const_cast<char *>(id.c_str())),
            PUMP,
            UA_NS0ID(HASCOMPONENT),
            UA_QUALIFIEDNAME(1, const_cast<char *>(name)),
            UA_NODEID_NULL,
            attr,
            nullptr,
            nullptr
        );
    }

    static UA_StatusCode set_pump(
        UA_Server *server,
        const UA_NodeId *,
        void *,
        const UA_NodeId *method,
        void *,
        const UA_NodeId *,
        void *,
        const size_t input_size,
        const UA_Variant *input,
        size_t,
        UA_Variant *
    ) {
        if (UA_NodeId_equal(method, &PUMP_FAIL)) return UA_STATUSCODE_BADINVALIDSTATE;
        const auto &speed = input[input_size - 1];
        UA_Server_writeValue(
            server,
            UA_NODEID_STRING(1, const_cast<char *>("Pump.Speed")),
            speed
        );
        if (input_size == 2)
            UA_Server_writeValue(
                server,
                UA_NODEID_STRING(1, const_cast<char *>("Pump.Mode")),
                input[0]
            );
        return UA_STATUSCODE_GOOD;
    }

    /// @brief adds a method named name to the Pump object, with the input arguments
    /// args.
    static void add_pump_method(
        UA_Server *server,
        const UA_NodeId &id,
        const char *name,
        const std::vector<std::pair<const char *, const UA_DataType *>> &args
    ) {
        std::vector<UA_Argument> inputs(args.size());
        for (size_t i = 0; i < args.size(); ++i) {
            UA_Argument_init(&inputs[i]);
            inputs[i].name = UA_STRING(const_cast<char *>(args[i].first));
            inputs[i].dataType = args[i].second->typeId;
            inputs[i].valueRank = UA_VALUERANK_SCALAR;
        }
        UA_MethodAttributes attr = UA_MethodAttributes_default;
        attr.displayName = UA_LOCALIZEDTEXT(
            const_cast<char *>("en-US"),
            const_cast<char *>(name)
        );
        attr.executable = true;
        attr.userExecutable = true;
        UA_Server_addMethodNode(
            server,
            id,
            PUMP,
            UA_NS0ID(HASCOMPONENT),
            UA_QUALIFIEDNAME(1, const_cast<char *>(name)),
            attr,
            set_pump,
            inputs.size(),
            inputs.data(),
            0,
            nullptr,
            nullptr,
            nullptr
        );
    }

    /// @brief adds the object that ServerConfig::pump describes.
    static void add_pump(UA_Server *server, const Pump &pump) {
        UA_ObjectAttributes attr = UA_ObjectAttributes_default;
        attr.displayName = UA_LOCALIZEDTEXT(
            const_cast<char *>("en-US"),
            const_cast<char *>("Pump")
        );
        attr.eventNotifier = UA_EVENTNOTIFIER_SUBSCRIBE_TO_EVENT;
        UA_Server_addObjectNode(
            server,
            PUMP,
            UA_NS0ID(OBJECTSFOLDER),
            UA_NS0ID(ORGANIZES),
            UA_QUALIFIEDNAME(1, const_cast<char *>("Pump")),
            UA_NS0ID(BASEOBJECTTYPE),
            attr,
            nullptr,
            nullptr
        );
        add_pump_variable(server, "Speed", &pump.speed, UA_TYPES[UA_TYPES_DOUBLE]);
        UA_String mode = UA_STRING(const_cast<char *>(pump.mode.c_str()));
        add_pump_variable(server, "Mode", &mode, UA_TYPES[UA_TYPES_STRING]);
        const auto *dbl = &UA_TYPES[UA_TYPES_DOUBLE];
        const auto *str = &UA_TYPES[UA_TYPES_STRING];
        add_pump_method(server, PUMP_SET_SPEED, "SetSpeed", {{"speed", dbl}});
        add_pump_method(
            server,
            PUMP_CONFIGURE,
            "Configure",
            {{"mode", str}, {"speed", dbl}}
        );
        add_pump_method(server, PUMP_FAIL, "Fail", {});
    }

    /// @brief adds TEST_EVENT_TYPE.
    static void add_event_type(UA_Server *server) {
        UA_ObjectTypeAttributes attr = UA_ObjectTypeAttributes_default;
        attr.displayName = UA_LOCALIZEDTEXT(
            const_cast<char *>("en-US"),
            const_cast<char *>("TestEventType")
        );
        UA_Server_addObjectTypeNode(
            server,
            TEST_EVENT_TYPE,
            UA_NS0ID(BASEEVENTTYPE),
            UA_NS0ID(HASSUBTYPE),
            UA_QUALIFIEDNAME(1, const_cast<char *>("TestEventType")),
            attr,
            nullptr,
            nullptr
        );
    }

    /// @brief emits the events that emit queued.
    void emit_events(UA_Server *server) {
        std::vector<Event> queued;
        {
            std::lock_guard lock(this->events_mutex);
            queued.swap(this->events);
        }
        for (const auto &event: queued) {
            UA_KeyValueMap fields = UA_KEYVALUEMAP_NULL;
            if (event.time.has_value()) {
                const UA_DateTime time = event.time->nanoseconds() / 100 +
                                         UA_DATETIME_UNIX_EPOCH;
                UA_KeyValueMap_setScalar(
                    &fields,
                    UA_QUALIFIEDNAME(0, const_cast<char *>("/Time")),
                    &time,
                    &UA_TYPES[UA_TYPES_DATETIME]
                );
            }
            if (event.level.has_value())
                UA_KeyValueMap_setScalar(
                    &fields,
                    UA_QUALIFIEDNAME(0, const_cast<char *>("/1:Level")),
                    &*event.level,
                    &UA_TYPES[UA_TYPES_DOUBLE]
                );
            const auto status = UA_Server_createEvent(
                server,
                event.source,
                event.type,
                event.severity,
                UA_LOCALIZEDTEXT(
                    const_cast<char *>("en-US"),
                    const_cast<char *>(event.message.c_str())
                ),
                &fields,
                nullptr,
                nullptr
            );
            UA_KeyValueMap_clear(&fields);
            if (status != UA_STATUSCODE_GOOD)
                LOG(WARNING) << "Mock OPC UA server failed to emit an event: "
                             << UA_StatusCode_name(status);
        }
    }

    /// @brief adds a folder with the browse name name under parent.
    static void add_folder(
        UA_Server *server,
        const std::string &id,
        const std::string &name,
        const UA_NodeId &parent
    ) {
        UA_ObjectAttributes attr = UA_ObjectAttributes_default;
        attr.displayName = UA_LOCALIZEDTEXT(
            const_cast<char *>("en-US"),
            const_cast<char *>(name.c_str())
        );
        UA_Server_addObjectNode(
            server,
            UA_NODEID_STRING(1, const_cast<char *>(id.c_str())),
            parent,
            UA_NS0ID(ORGANIZES),
            UA_QUALIFIEDNAME(1, const_cast<char *>(name.c_str())),
            UA_NS0ID(FOLDERTYPE),
            attr,
            nullptr,
            nullptr
        );
    }

    /// @brief adds a Double variable with the browse name name under parent.
    static void add_double(
        UA_Server *server,
        const std::string &id,
        const std::string &name,
        const UA_NodeId &parent
    ) {
        UA_VariableAttributes attr = UA_VariableAttributes_default;
        UA_Double value = 1.0;
        UA_Variant_setScalar(&attr.value, &value, &UA_TYPES[UA_TYPES_DOUBLE]);
        attr.dataType = UA_TYPES[UA_TYPES_DOUBLE].typeId;
        attr.valueRank = UA_VALUERANK_SCALAR;
        attr.accessLevel = UA_ACCESSLEVELMASK_READ;
        UA_Server_addVariableNode(
            server,
            UA_NODEID_STRING(1, const_cast<char *>(id.c_str())),
            parent,
            UA_NS0ID(HASCOMPONENT),
            UA_QUALIFIEDNAME(1, const_cast<char *>(name.c_str())),
            UA_NODEID_NULL,
            attr,
            nullptr,
            nullptr
        );
    }

    /// @brief adds the folders that ServerConfig::generated_variables and
    /// ServerConfig::generated_folders describe.
    static void add_generated(
        UA_Server *server,
        const std::uint32_t variables,
        const std::uint32_t folders
    ) {
        const auto generated = UA_NODEID_STRING(1, const_cast<char *>("Generated"));
        const auto inner = UA_NODEID_STRING(1, const_cast<char *>("Generated.Inner"));
        add_folder(server, "Generated", "Generated", UA_NS0ID(OBJECTSFOLDER));
        for (std::uint32_t i = 0; i < variables; ++i) {
            const auto name = "V" + std::to_string(i);
            add_double(server, "Generated." + name, name, generated);
        }
        for (std::uint32_t i = 0; i < folders; ++i) {
            const auto name = "F" + std::to_string(i);
            add_folder(server, "Generated." + name, name, generated);
        }
        add_folder(server, "Generated.Inner", "Inner", generated);
        add_double(server, "Generated.Inner.Deep", "Deep", inner);
        UA_ExpandedNodeId target = UA_EXPANDEDNODEID_NULL;
        target.nodeId = generated;
        UA_Server_addReference(server, inner, UA_NS0ID(ORGANIZES), target, true);
    }

    void run() {
        UA_Server *server = UA_Server_new();
        auto *server_config = UA_Server_getConfig(server);
        if (cfg.identity.has_value()) {
            const auto bytes = [](const std::string &data) {
                return UA_ByteString{
                    data.size(),
                    reinterpret_cast<UA_Byte *>(const_cast<char *>(data.data()))
                };
            };
            std::vector<UA_ByteString> trust_list;
            trust_list.reserve(cfg.trusted_clients.size());
            for (const auto &c: cfg.trusted_clients)
                trust_list.push_back(bytes(c.der()));
            const auto cert = bytes(cfg.identity->certificate.der());
            const auto key = bytes(cfg.identity->private_key);
            const auto status = UA_ServerConfig_setDefaultWithSecurityPolicies(
                server_config,
                cfg.port,
                &cert,
                &key,
                trust_list.data(),
                trust_list.size(),
                nullptr,
                0,
                nullptr,
                0
            );
            if (status != UA_STATUSCODE_GOOD) {
                LOG(WARNING) << "Mock OPC UA server failed to configure security: "
                             << UA_StatusCode_name(status);
                UA_Server_delete(server);
                return;
            }
            if (cfg.trusted_clients.empty()) {
                UA_CertificateGroup_AcceptAll(&server_config->secureChannelPKI);
                UA_CertificateGroup_AcceptAll(&server_config->sessionPKI);
            }
        } else {
            UA_ServerConfig_setMinimal(server_config, cfg.port, nullptr);
            auto &uri = server_config->applicationDescription.applicationUri;
            UA_String_clear(&uri);
            uri = UA_STRING_ALLOC(SERVER_APPLICATION_URI.c_str());
        }
        server_config->maxSessionTimeout = 3600000;
        if (cfg.max_sessions > 0) server_config->maxSessions = cfg.max_sessions;
        server_config->maxNodesPerRead = cfg.max_nodes_per_read;
        server_config->maxNodesPerWrite = cfg.max_nodes_per_write;
        server_config->maxNodesPerBrowse = cfg.max_nodes_per_browse;
        server_config->maxReferencesPerNode = cfg.max_references_per_node;
        for (const auto &uri: cfg.namespaces)
            UA_Server_addNamespace(server, uri.c_str());
        const StructureTypes structures;
        server_config->customDataTypes = const_cast<UA_DataTypeArray *>(
            &structures.array
        );
        for (const auto *type: {&structures.point(), &structures.motor()}) {
            UA_DataTypeAttributes attr = UA_DataTypeAttributes_default;
            attr.displayName = UA_LOCALIZEDTEXT(
                const_cast<char *>("en-US"),
                const_cast<char *>(type->typeName)
            );
            UA_Server_addDataTypeNode(
                server,
                type->typeId,
                UA_NS0ID(STRUCTURE),
                UA_NS0ID(HASSUBTYPE),
                UA_QUALIFIEDNAME(1, const_cast<char *>(type->typeName)),
                attr,
                nullptr,
                nullptr
            );
        }
        if (cfg.motor.has_value()) {
            UA_VariableAttributes attr = UA_VariableAttributes_default;
            attr.dataType = structures.motor().typeId;
            attr.valueRank = UA_VALUERANK_SCALAR;
            attr.accessLevel = UA_ACCESSLEVELMASK_READ;
            attr.displayName = UA_LOCALIZEDTEXT(
                const_cast<char *>("en-US"),
                const_cast<char *>("TestMotor")
            );
            UA_Variant_setScalar(&attr.value, &*cfg.motor, &structures.motor());
            UA_Server_addVariableNode(
                server,
                UA_NODEID_STRING(1, const_cast<char *>("TestMotor")),
                UA_NS0ID(OBJECTSFOLDER),
                UA_NS0ID(ORGANIZES),
                UA_QUALIFIEDNAME(1, const_cast<char *>("TestMotor")),
                UA_NODEID_NULL,
                attr,
                nullptr,
                nullptr
            );
        }

        for (const auto &node: cfg.test_nodes) {
            UA_VariableAttributes attr = UA_VariableAttributes_default;

            // Set the variant as the value to be exposed by the server
            attr.value = node.initial_value;
            if (node.data_type != nullptr) attr.dataType = node.data_type->typeId;
            auto length = static_cast<UA_UInt32>(node.initial_value.arrayLength);
            if (UA_Variant_isScalar(&node.initial_value))
                attr.valueRank = UA_VALUERANK_SCALAR;
            else if (node.initial_value.data != nullptr) {
                attr.valueRank = UA_VALUERANK_ONE_DIMENSION;
                attr.arrayDimensionsSize = 1;
                attr.arrayDimensions = &length;
            }

            // Set access level to allow reading and writing
            attr.accessLevel = UA_ACCESSLEVELMASK_READ | UA_ACCESSLEVELMASK_WRITE;

            driver::opcua::types::LocalizedText description(
                "en-US",
                node.description.c_str()
            );
            driver::opcua::types::LocalizedText displayName(
                "en-US",
                node.node_id.c_str()
            );
            attr.description = description.get();
            attr.displayName = displayName.get();

            UA_NodeId raw_node_id = UA_NODEID_STRING_ALLOC(
                node.ns,
                node.node_id.c_str()
            );
            driver::opcua::types::NodeId nodeId(raw_node_id);
            UA_NodeId_clear(&raw_node_id);
            LOG(INFO) << "Creating OPC UA node: "
                      << driver::opcua::types::NodeId::to_string(nodeId.get());

            driver::opcua::types::QualifiedName nodeName(node.ns, node.node_id.c_str());
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
                nullptr,
                nullptr
            );
        }

        if (cfg.generated_variables > 0 || cfg.generated_folders > 0)
            add_generated(server, cfg.generated_variables, cfg.generated_folders);
        if (cfg.pump.has_value()) add_pump(server, *cfg.pump);
        add_event_type(server);

        UA_StatusCode status = UA_Server_run_startup(server);
        if (status != UA_STATUSCODE_GOOD) {
            LOG(WARNING) << "Mock OPC UA server stopped with status: "
                         << UA_StatusCode_name(status);
            UA_Server_delete(server);
            return;
        }

        ready = true;

        while (running.load()) {
            if (probe_delay.nanoseconds() > 0)
                std::this_thread::sleep_for(
                    std::chrono::nanoseconds(probe_delay.nanoseconds())
                );
            this->emit_events(server);
            UA_Server_run_iterate(server, true);
            const auto stats = UA_Server_getStatistics(server);
            this->channels = stats.scs.currentChannelCount;
            this->sessions_created = stats.ss.cumulatedSessionCount;
        }

        ready = false;
        UA_Server_run_shutdown(server);
        UA_Server_delete(server);
    }

private:
    std::mutex events_mutex;
    std::vector<Event> events;
};
}
