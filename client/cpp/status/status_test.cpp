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

TEST(StatusTest, VariantConstants) {
    EXPECT_EQ(status::variant::SUCCESS, "success");
    EXPECT_EQ(status::variant::INFO, "info");
    EXPECT_EQ(status::variant::WARNING, "warning");
    EXPECT_EQ(status::variant::ERR, "error");
    EXPECT_EQ(status::variant::LOADING, "loading");
    EXPECT_EQ(status::variant::DISABLED, "disabled");
}

TEST(StatusTest, StatusConstruction) {
    Status s1;
    s1.key = "test-key-1";
    s1.variant = status::variant::INFO;
    s1.message = "Test message";
    EXPECT_EQ(s1.key, "test-key-1");
    EXPECT_EQ(s1.message, "Test message");
    EXPECT_EQ(s1.variant, status::variant::INFO);
    EXPECT_TRUE(s1.name.empty());
    EXPECT_TRUE(s1.description.empty());

    Status s2;
    s2.key = "custom-key";
    s2.name = "Error Name";
    s2.variant = status::variant::ERR;
    s2.message = "Error message";
    s2.description = "Detailed description";
    EXPECT_EQ(s2.key, "custom-key");
    EXPECT_EQ(s2.message, "Error message");
    EXPECT_EQ(s2.variant, status::variant::ERR);
    EXPECT_EQ(s2.name, "Error Name");
    EXPECT_EQ(s2.description, "Detailed description");
}

TEST(StatusTest, StatusClientEmpty) {
    StatusClient client;
    Status s;
    s.key = "test-key";
    s.variant = status::variant::INFO;
    s.message = "Test";
}

TEST(StatusTest, SetSingleStatus) {
    auto client = new_test_client();
    Status s;
    s.key = "test-status-1";
    s.variant = status::variant::INFO;
    s.message = "Test message";
    s.time = telem::TimeStamp::now();
    auto err = client.statuses.set(s);
    EXPECT_FALSE(err) << err.message();
    EXPECT_EQ(s.key, "test-status-1");
}

TEST(StatusTest, RetrieveStatus) {
    auto client = new_test_client();
    Status s;
    s.key = "test-status-retrieve";
    s.variant = status::variant::SUCCESS;
    s.message = "Retrievable";
    s.time = telem::TimeStamp::now();
    auto set_err = client.statuses.set(s);
    EXPECT_FALSE(set_err) << set_err.message();
    auto [retrieved, err] = client.statuses.retrieve(s.key);
    EXPECT_FALSE(err) << err.message();
    EXPECT_EQ(retrieved.key, s.key);
    EXPECT_EQ(retrieved.message, s.message);
    EXPECT_EQ(retrieved.variant, s.variant);
}

TEST(StatusTest, DeleteStatus) {
    auto client = new_test_client();
    Status s;
    s.key = "test-status-delete";
    s.variant = status::variant::INFO;
    s.message = "To delete";
    s.time = telem::TimeStamp::now();
    auto set_err = client.statuses.set(s);
    EXPECT_FALSE(set_err) << set_err.message();
    auto err = client.statuses.del(s.key);
    EXPECT_FALSE(err) << err.message();
    auto [retrieved, err2] = client.statuses.retrieve(s.key);
    EXPECT_TRUE(err2);
}

TEST(StatusTest, SetMultipleStatuses) {
    auto client = new_test_client();
    std::vector<Status> statuses;
    for (int i = 0; i < 3; i++) {
        Status s;
        s.key = "test-batch-" + std::to_string(i);
        s.variant = status::variant::INFO;
        s.message = "Batch status " + std::to_string(i);
        s.time = telem::TimeStamp::now();
        statuses.push_back(s);
    }
    auto err = client.statuses.set(statuses);
    EXPECT_FALSE(err) << err.message();
    EXPECT_EQ(statuses.size(), 3);
    for (size_t i = 0; i < statuses.size(); i++) {
        EXPECT_EQ(statuses[i].key, "test-batch-" + std::to_string(i));
    }
}

TEST(StatusTest, RetrieveMultipleStatuses) {
    auto client = new_test_client();
    std::vector<Status> to_create;
    for (int i = 0; i < 3; i++) {
        Status s;
        s.key = "test-multi-retrieve-" + std::to_string(i);
        s.variant = status::variant::SUCCESS;
        s.message = "Multi retrieve " + std::to_string(i);
        s.time = telem::TimeStamp::now();
        to_create.push_back(s);
    }
    auto set_err = client.statuses.set(to_create);
    EXPECT_FALSE(set_err) << set_err.message();
    std::vector<std::string> keys = {
        "test-multi-retrieve-0",
        "test-multi-retrieve-1",
        "test-multi-retrieve-2"
    };
    auto [retrieved, err] = client.statuses.retrieve(keys);
    EXPECT_FALSE(err) << err.message();
    EXPECT_EQ(retrieved.size(), 3);
    for (size_t i = 0; i < retrieved.size(); i++) {
        EXPECT_EQ(retrieved[i].key, "test-multi-retrieve-" + std::to_string(i));
        EXPECT_EQ(retrieved[i].variant, status::variant::SUCCESS);
    }
}

TEST(StatusTest, UpdateExistingStatus) {
    auto client = new_test_client();
    Status s;
    s.key = "test-status-update";
    s.variant = status::variant::INFO;
    s.message = "Original message";
    s.time = telem::TimeStamp::now();
    auto err1 = client.statuses.set(s);
    EXPECT_FALSE(err1) << err1.message();
    s.variant = status::variant::WARNING;
    s.message = "Updated message";
    s.description = "Added description";
    s.time = telem::TimeStamp::now();
    auto err2 = client.statuses.set(s);
    EXPECT_FALSE(err2) << err2.message();
    auto [retrieved, err3] = client.statuses.retrieve(s.key);
    EXPECT_FALSE(err3) << err3.message();
    EXPECT_EQ(retrieved.key, "test-status-update");
    EXPECT_EQ(retrieved.variant, status::variant::WARNING);
    EXPECT_EQ(retrieved.message, "Updated message");
    EXPECT_EQ(retrieved.description, "Added description");
}

TEST(StatusTest, RetrieveNonExistentStatus) {
    auto client = new_test_client();
    auto [retrieved, err] = client.statuses.retrieve("non-existent-status-key");
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches("sy.query.not_found"))
        << "Expected not_found error, got: " << err.message();
}

TEST(StatusTest, DeleteMultipleStatuses) {
    auto client = new_test_client();
    std::vector<Status> to_create;
    for (int i = 0; i < 3; i++) {
        Status s;
        s.key = "test-multi-delete-" + std::to_string(i);
        s.variant = status::variant::INFO;
        s.message = "To be deleted " + std::to_string(i);
        s.time = telem::TimeStamp::now();
        to_create.push_back(s);
    }
    auto set_err = client.statuses.set(to_create);
    EXPECT_FALSE(set_err) << set_err.message();
    std::vector<std::string> keys = {
        "test-multi-delete-0",
        "test-multi-delete-1",
        "test-multi-delete-2"
    };
    auto del_err = client.statuses.del(keys);
    EXPECT_FALSE(del_err) << del_err.message();
    auto [retrieved, err] = client.statuses.retrieve(keys);
    EXPECT_TRUE(err);
}

TEST(StatusTest, DetailsRoundTrip) {
    auto client = new_test_client();
    Status s;
    s.key = "test-status-details";
    s.variant = status::variant::INFO;
    s.message = "Testing details";
    s.time = telem::TimeStamp::now();
    // DefaultDetails has a default to_json that returns empty object
    // Verify it round-trips correctly
    auto set_err = client.statuses.set(s);
    EXPECT_FALSE(set_err) << set_err.message();
    auto [retrieved, err] = client.statuses.retrieve(s.key);
    EXPECT_FALSE(err) << err.message();
    EXPECT_EQ(retrieved.key, s.key);
    EXPECT_EQ(retrieved.message, s.message);
    // DefaultDetails should serialize as empty JSON object and deserialize back
    auto details_json = retrieved.details.to_json();
    EXPECT_TRUE(details_json.is_object());
    EXPECT_TRUE(details_json.empty());
}
