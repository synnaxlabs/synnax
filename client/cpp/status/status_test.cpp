// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/status/status.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/xtest/xtest.h"

using namespace synnax;

/// @brief it should set a single status in the cluster.
TEST(StatusTest, SetSingleStatus) {
    const auto client = new_test_client();
    Status s;
    s.key = "test-status-1";
    s.variant = status::variant::INFO;
    s.message = "Test message";
    s.time = telem::TimeStamp::now();
    ASSERT_NIL(client.statuses.set(s));
    EXPECT_EQ(s.key, "test-status-1");
}

/// @brief it should retrieve a status by its key.
TEST(StatusTest, RetrieveStatus) {
    const auto client = new_test_client();
    Status s;
    s.key = "test-status-retrieve";
    s.variant = status::variant::SUCCESS;
    s.message = "Retrievable";
    s.time = telem::TimeStamp::now();
    ASSERT_NIL(client.statuses.set(s));
    const auto retrieved = ASSERT_NIL_P(client.statuses.retrieve(s.key));
    EXPECT_EQ(retrieved.key, s.key);
    EXPECT_EQ(retrieved.message, s.message);
    EXPECT_EQ(retrieved.variant, s.variant);
}

/// @brief it should delete a status from the cluster.
TEST(StatusTest, DeleteStatus) {
    const auto client = new_test_client();
    Status s;
    s.key = "test-status-delete";
    s.variant = status::variant::INFO;
    s.message = "To delete";
    s.time = telem::TimeStamp::now();
    ASSERT_NIL(client.statuses.set(s));
    ASSERT_NIL(client.statuses.del(s.key));
    ASSERT_OCCURRED_AS_P(client.statuses.retrieve(s.key), xerrors::NOT_FOUND);
}

/// @brief it should set multiple statuses in a batch.
TEST(StatusTest, SetMultipleStatuses) {
    const auto client = new_test_client();
    std::vector<Status> statuses;
    for (int i = 0; i < 3; i++) {
        Status s;
        s.key = "test-batch-" + std::to_string(i);
        s.variant = status::variant::INFO;
        s.message = "Batch status " + std::to_string(i);
        s.time = telem::TimeStamp::now();
        statuses.push_back(s);
    }
    ASSERT_NIL(client.statuses.set(statuses));
    EXPECT_EQ(statuses.size(), 3);
    for (size_t i = 0; i < statuses.size(); i++)
        EXPECT_EQ(statuses[i].key, "test-batch-" + std::to_string(i));
}

/// @brief it should retrieve multiple statuses by their keys.
TEST(StatusTest, RetrieveMultipleStatuses) {
    const auto client = new_test_client();
    std::vector<Status> to_create;
    for (int i = 0; i < 3; i++) {
        Status s;
        s.key = "test-multi-retrieve-" + std::to_string(i);
        s.variant = status::variant::SUCCESS;
        s.message = "Multi retrieve " + std::to_string(i);
        s.time = telem::TimeStamp::now();
        to_create.push_back(s);
    }
    ASSERT_NIL(client.statuses.set(to_create));
    const std::vector<std::string> keys = {
        "test-multi-retrieve-0",
        "test-multi-retrieve-1",
        "test-multi-retrieve-2"
    };
    const auto retrieved = ASSERT_NIL_P(client.statuses.retrieve(keys));
    EXPECT_EQ(retrieved.size(), 3);
    for (size_t i = 0; i < retrieved.size(); i++) {
        EXPECT_EQ(retrieved[i].key, "test-multi-retrieve-" + std::to_string(i));
        EXPECT_EQ(retrieved[i].variant, status::variant::SUCCESS);
    }
}

/// @brief it should update an existing status with new values.
TEST(StatusTest, UpdateExistingStatus) {
    const auto client = new_test_client();
    Status s;
    s.key = "test-status-update";
    s.variant = status::variant::INFO;
    s.message = "Original message";
    s.time = telem::TimeStamp::now();
    ASSERT_NIL(client.statuses.set(s));
    s.variant = status::variant::WARNING;
    s.message = "Updated message";
    s.description = "Added description";
    s.time = telem::TimeStamp::now();
    ASSERT_NIL(client.statuses.set(s));
    const auto retrieved = ASSERT_NIL_P(client.statuses.retrieve(s.key));
    EXPECT_EQ(retrieved.key, "test-status-update");
    EXPECT_EQ(retrieved.variant, status::variant::WARNING);
    EXPECT_EQ(retrieved.message, "Updated message");
    EXPECT_EQ(retrieved.description, "Added description");
}

/// @brief it should return a not found error for non-existent status.
TEST(StatusTest, RetrieveNonExistentStatus) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        client.statuses.retrieve("non-existent-status-key"),
        xerrors::NOT_FOUND
    );
}

/// @brief it should delete multiple statuses in a batch.
TEST(StatusTest, DeleteMultipleStatuses) {
    const auto client = new_test_client();
    std::vector<Status> to_create;
    for (int i = 0; i < 3; i++) {
        Status s;
        s.key = "test-multi-delete-" + std::to_string(i);
        s.variant = status::variant::INFO;
        s.message = "To be deleted " + std::to_string(i);
        s.time = telem::TimeStamp::now();
        to_create.push_back(s);
    }
    ASSERT_NIL(client.statuses.set(to_create));
    std::vector<std::string> keys = {
        "test-multi-delete-0",
        "test-multi-delete-1",
        "test-multi-delete-2"
    };
    ASSERT_NIL(client.statuses.del(keys));
    ASSERT_OCCURRED_AS_P(client.statuses.retrieve(keys), xerrors::NOT_FOUND);
}

/// @brief it should round-trip status details through JSON.
TEST(StatusTest, DetailsRoundTrip) {
    const auto client = new_test_client();
    Status s;
    s.key = "test-status-details";
    s.variant = status::variant::INFO;
    s.message = "Testing details";
    s.time = telem::TimeStamp::now();
    ASSERT_NIL(client.statuses.set(s));
    const auto retrieved = ASSERT_NIL_P(client.statuses.retrieve(s.key));
    EXPECT_EQ(retrieved.key, s.key);
    EXPECT_EQ(retrieved.message, s.message);
    const auto details_json = retrieved.details.to_json();
    EXPECT_TRUE(details_json.is_object());
    EXPECT_TRUE(details_json.empty());
}

// Custom details type for testing templated status client
struct CustomStatusDetails {
    std::string device_id;
    int error_code = 0;
    bool critical = false;

    [[nodiscard]] json to_json() const {
        return json{
            {"device_id", device_id},
            {"error_code", error_code},
            {"critical", critical}
        };
    }

    static CustomStatusDetails parse(xjson::Parser &parser) {
        return CustomStatusDetails{
            .device_id = parser.field<std::string>("device_id", ""),
            .error_code = parser.field<int>("error_code", 0),
            .critical = parser.field<bool>("critical", false),
        };
    }
};

/// @brief it should set and retrieve a status with custom details type.
TEST(StatusTest, CustomDetailsSetAndRetrieve) {
    const auto client = new_test_client();
    status::Status<CustomStatusDetails> s;
    s.key = "test-custom-details-1";
    s.variant = status::variant::ERR;
    s.message = "Device error occurred";
    s.description = "Critical device failure";
    s.time = telem::TimeStamp::now();
    s.details.device_id = "device-alpha-123";
    s.details.error_code = 42;
    s.details.critical = true;

    ASSERT_NIL(client.statuses.set<CustomStatusDetails>(s));

    const auto retrieved = ASSERT_NIL_P(
        client.statuses.retrieve<CustomStatusDetails>(s.key)
    );
    EXPECT_EQ(retrieved.key, s.key);
    EXPECT_EQ(retrieved.variant, s.variant);
    EXPECT_EQ(retrieved.message, s.message);
    EXPECT_EQ(retrieved.description, s.description);
    EXPECT_EQ(retrieved.details.device_id, "device-alpha-123");
    EXPECT_EQ(retrieved.details.error_code, 42);
    EXPECT_EQ(retrieved.details.critical, true);
}

/// @brief it should set and retrieve multiple statuses with custom details.
TEST(StatusTest, CustomDetailsSetMultiple) {
    const auto client = new_test_client();
    std::vector<status::Status<CustomStatusDetails>> statuses;

    for (int i = 0; i < 3; i++) {
        status::Status<CustomStatusDetails> s;
        s.key = "test-custom-batch-" + std::to_string(i);
        s.variant = status::variant::WARNING;
        s.message = "Warning " + std::to_string(i);
        s.time = telem::TimeStamp::now();
        s.details.device_id = "device-" + std::to_string(i);
        s.details.error_code = i * 10;
        s.details.critical = (i % 2 == 0);
        statuses.push_back(s);
    }

    ASSERT_NIL(client.statuses.set<CustomStatusDetails>(statuses));
    EXPECT_EQ(statuses.size(), 3);

    std::vector<std::string> keys;
    for (const auto &s: statuses)
        keys.push_back(s.key);

    const auto retrieved = ASSERT_NIL_P(
        client.statuses.retrieve<CustomStatusDetails>(keys)
    );
    EXPECT_EQ(retrieved.size(), 3);

    for (size_t i = 0; i < retrieved.size(); i++) {
        EXPECT_EQ(retrieved[i].key, "test-custom-batch-" + std::to_string(i));
        EXPECT_EQ(retrieved[i].variant, status::variant::WARNING);
        EXPECT_EQ(retrieved[i].details.device_id, "device-" + std::to_string(i));
        EXPECT_EQ(retrieved[i].details.error_code, static_cast<int>(i * 10));
        EXPECT_EQ(retrieved[i].details.critical, (i % 2 == 0));
    }
}

/// @brief it should update a status with custom details.
TEST(StatusTest, CustomDetailsUpdate) {
    const auto client = new_test_client();
    status::Status<CustomStatusDetails> s;
    s.key = "test-custom-update";
    s.variant = status::variant::WARNING;
    s.message = "Initial warning";
    s.time = telem::TimeStamp::now();
    s.details.device_id = "device-xyz";
    s.details.error_code = 100;
    s.details.critical = false;

    ASSERT_NIL(client.statuses.set<CustomStatusDetails>(s));

    // Update the status with new details
    s.variant = status::variant::ERR;
    s.message = "Escalated to error";
    s.details.error_code = 500;
    s.details.critical = true;

    ASSERT_NIL(client.statuses.set<CustomStatusDetails>(s));

    const auto retrieved = ASSERT_NIL_P(
        client.statuses.retrieve<CustomStatusDetails>(s.key)
    );
    EXPECT_EQ(retrieved.key, "test-custom-update");
    EXPECT_EQ(retrieved.variant, status::variant::ERR);
    EXPECT_EQ(retrieved.message, "Escalated to error");
    EXPECT_EQ(retrieved.details.device_id, "device-xyz");
    EXPECT_EQ(retrieved.details.error_code, 500);
    EXPECT_EQ(retrieved.details.critical, true);
}

/// @brief it should handle custom details with empty or default fields.
TEST(StatusTest, CustomDetailsEmptyFields) {
    const auto client = new_test_client();
    status::Status<CustomStatusDetails> s;
    s.key = "test-custom-empty";
    s.variant = status::variant::INFO;
    s.message = "Empty details test";
    s.time = telem::TimeStamp::now();
    // Leave details with default values

    ASSERT_NIL(client.statuses.set<CustomStatusDetails>(s));

    const auto retrieved = ASSERT_NIL_P(
        client.statuses.retrieve<CustomStatusDetails>(s.key)
    );
    EXPECT_EQ(retrieved.details.device_id, "");
    EXPECT_EQ(retrieved.details.error_code, 0);
    EXPECT_EQ(retrieved.details.critical, false);
}
