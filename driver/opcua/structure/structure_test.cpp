// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/opcua/connection/connection.h"
#include "driver/opcua/structure/structure.h"
#include "driver/opcua/testutil/server.h"
#include "driver/opcua/testutil/testutil.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua::structure {
/// @brief it should split a field into member names.
TEST(Parse, testSplitsMembers) {
    const auto path = ASSERT_NIL_P(parse("Position.Y"));
    EXPECT_EQ(path, (Path{"Position", "Y"}));
}

/// @brief it should read the character after '&' as part of a name.
TEST(Parse, testEscapes) {
    const auto path = ASSERT_NIL_P(parse("a&.b.c&&d"));
    EXPECT_EQ(path, (Path{"a.b", "c&d"}));
}

/// @brief it should reject a field with an empty name.
TEST(Parse, testRejectsEmptyName) {
    for (const auto *field: {"", "a..b", ".a", "a."}) {
        auto [path, err] = parse(field);
        ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
        EXPECT_EQ(err.data, "the field " + std::string(field) + " has an empty name");
    }
}

/// @brief it should reject a field that ends in '&'.
TEST(Parse, testRejectsTrailingEscape) {
    auto [path, err] = parse("a&");
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "the field a& ends in &");
}

/// @brief it should escape names so that parse returns the same path.
TEST(ToString, testRoundTrips) {
    const Path path{"a.b", "c&d", "e"};
    EXPECT_EQ(to_string(path), "a&.b.c&&d.e");
    EXPECT_EQ(ASSERT_NIL_P(parse(to_string(path))), path);
}

class TypesTest : public ::testing::Test {
protected:
    void SetUp() override {
        auto server_cfg = testutil::ServerConfig::create_default();
        server_cfg.port = 4880;
        server_cfg.motor = testutil::Motor{
            .speed = 1500.0,
            .running = true,
            .position = {.x = 1.5F, .y = 2.5F},
        };
        this->server = std::make_unique<testutil::Server>(server_cfg);
        this->server->start();
        ASSERT_TRUE(this->server->wait_until_ready());
        connection::Config cfg;
        cfg.endpoint = "opc.tcp://localhost:4880";
        cfg.security_mode = "None";
        cfg.security_policy = "None";
        this->pool = std::make_unique<connection::Pool>(testutil::create_store());
        this->conn = ASSERT_NIL_P(this->pool->acquire(cfg, "[test] "));
    }

    void TearDown() override {
        this->conn = connection::Pool::Connection(nullptr, nullptr, "");
        this->server->stop();
    }

    /// @returns the value of the node.
    [[nodiscard]] types::ReadResponse read(const UA_NodeId &node) const {
        types::ReadRequestBuilder req;
        req.add_node(node);
        auto res = ASSERT_NIL_P(this->conn.read(req.build()));
        EXPECT_EQ(res.get().resultsSize, 1);
        return res;
    }

    /// @returns the data type in namespace 1 with the numeric ID.
    [[nodiscard]] static UA_NodeId type_id(const UA_UInt32 id) {
        return UA_NODEID_NUMERIC(1, id);
    }

    std::unique_ptr<testutil::Server> server;
    std::unique_ptr<connection::Pool> pool;
    connection::Pool::Connection conn{nullptr, nullptr, ""};
    const UA_NodeId motor = UA_NODEID_STRING(1, const_cast<char *>("TestMotor"));
    const UA_NodeId motor_type = type_id(testutil::MOTOR_TYPE_ID);
};

/// @brief it should define a structure type that the client does not know, with the
/// types of its members.
TEST_F(TypesTest, testReadsDefinitions) {
    const auto types = ASSERT_NIL_P(Types::read(this->conn, {this->motor_type}));
    const auto *motor = types.find(this->motor_type);
    ASSERT_NE(motor, nullptr);
    const auto members = Types::members(*motor);
    ASSERT_EQ(members.size(), 3);
    EXPECT_EQ(members[0].name, "Speed");
    EXPECT_EQ(members[0].type, &UA_TYPES[UA_TYPES_DOUBLE]);
    EXPECT_EQ(members[1].name, "Running");
    EXPECT_EQ(members[2].name, "Position");
    EXPECT_FALSE(members[2].is_array);
    EXPECT_EQ(members[2].type, types.find(type_id(testutil::POINT_TYPE_ID)));
    const auto y = ASSERT_NIL_P(types.field(this->motor_type, {"Position", "Y"}));
    EXPECT_EQ(&y.type(), &UA_TYPES[UA_TYPES_FLOAT]);
}

/// @brief it should reject a path to a member that does not exist.
TEST_F(TypesTest, testRejectsUnknownMember) {
    const auto types = ASSERT_NIL_P(Types::read(this->conn, {this->motor_type}));
    auto [field, err] = types.field(this->motor_type, {"Position", "Z"});
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "Point has no member Z");
}

/// @brief it should give the reason that a type the server defined does not build.
TEST_F(TypesTest, testReportsTypeThatDoesNotBuild) {
    const auto broken = type_id(testutil::BROKEN_TYPE_ID);
    const auto types = ASSERT_NIL_P(Types::read(this->conn, {broken}));
    EXPECT_EQ(types.find(broken), nullptr);
    auto [field, err] = types.field(broken, {"Inner"});
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_FALSE(err.matches(UNDEFINED));
    EXPECT_EQ(
        err.data,
        "cannot decode Broken: the server did not define the type of a member"
    );
}

/// @brief it should reject a field of a type that the server gives no definition of.
TEST_F(TypesTest, testRejectsUndefinedType) {
    const auto opaque = type_id(testutil::OPAQUE_TYPE_ID);
    const auto types = ASSERT_NIL_P(Types::read(this->conn, {opaque}));
    auto [field, err] = types.field(opaque, {"X"});
    ASSERT_OCCURRED_AS(err, UNDEFINED);
    EXPECT_EQ(err.data, "the server gives no definition of the data type of the node");
}

/// @brief it should decode a structure value and convert the member at a field.
TEST_F(TypesTest, testSamplesMembers) {
    const auto types = ASSERT_NIL_P(Types::read(this->conn, {this->motor_type}));
    const auto res = this->read(this->motor);
    const auto value = ASSERT_NIL_P(types.decode(res.get().results[0].value));
    const auto sample = [&](const Path &path, const x::telem::DataType &dt) {
        const auto field = ASSERT_NIL_P(types.field(this->motor_type, path));
        return ASSERT_NIL_P(Types::sample(value.get(), field, dt));
    };
    EXPECT_EQ(std::get<double>(sample({"Speed"}, x::telem::FLOAT64_T)), 1500.0);
    EXPECT_EQ(std::get<uint8_t>(sample({"Running"}, x::telem::BOOLEAN_T)), 1);
    EXPECT_EQ(std::get<float>(sample({"Position", "Y"}, x::telem::FLOAT32_T)), 2.5F);
}

/// @brief it should reject a structure of another type than the field's.
TEST_F(TypesTest, testRejectsStructureOfOtherType) {
    const auto point = type_id(testutil::POINT_TYPE_ID);
    const auto types = ASSERT_NIL_P(Types::read(this->conn, {this->motor_type}));
    const auto res = this->read(this->motor);
    const auto value = ASSERT_NIL_P(types.decode(res.get().results[0].value));
    const auto field = ASSERT_NIL_P(types.field(point, {"X"}));
    auto [sample, err] = Types::sample(value.get(), field, x::telem::FLOAT32_T);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "the value is a Motor, not a Point");
}

/// @brief it should reject a value whose type the server did not define.
TEST_F(TypesTest, testRejectsUndefinedValue) {
    const Types types;
    const auto res = this->read(this->motor);
    auto [value, err] = types.decode(res.get().results[0].value);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "the server did not define the type of the value");
}

/// @brief it should convert members of a structure type that the client knows.
TEST_F(TypesTest, testSamplesKnownStructure) {
    const auto status_type = UA_NODEID_NUMERIC(0, UA_NS0ID_SERVERSTATUSDATATYPE);
    const auto types = ASSERT_NIL_P(Types::read(this->conn, {status_type}));
    const auto res = this->read(UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_SERVERSTATUS));
    const auto value = ASSERT_NIL_P(types.decode(res.get().results[0].value));
    const auto state_field = ASSERT_NIL_P(types.field(status_type, {"State"}));
    const auto state = ASSERT_NIL_P(
        Types::sample(value.get(), state_field, x::telem::INT32_T)
    );
    EXPECT_EQ(std::get<int32_t>(state), UA_SERVERSTATE_RUNNING);
    const auto product_field = ASSERT_NIL_P(
        types.field(status_type, {"BuildInfo", "ProductName"})
    );
    const auto product = ASSERT_NIL_P(
        Types::sample(value.get(), product_field, x::telem::STRING_T)
    );
    EXPECT_FALSE(std::get<std::string>(product).empty());
}
}
