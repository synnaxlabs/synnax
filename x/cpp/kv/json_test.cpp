// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <gtest/gtest.h>
#include "x/cpp/kv/kv.h"

class JSONTest : public ::testing::Test {
protected:
    std::string temp_path;

    void SetUp() override {
        // Use system temp directory as base
        auto temp_dir = std::filesystem::temp_directory_path();
        temp_path = (temp_dir / "json_test" / "test.json").string();
    }

    void TearDown() override {
        // Clean up test files after each test
        try {
            std::filesystem::remove_all(std::filesystem::path(temp_path).parent_path());
        } catch (const std::filesystem::filesystem_error &e) {
            // Log error but don't fail the test
            std::cerr << "Cleanup failed: " << e.what() << std::endl;
        }
    }
};

TEST_F(JSONTest, CreateNewFile) {
    kv::JSONFileConfig config;
    config.path = temp_path;
    config.dir_mode = std::filesystem::perms::owner_read |
                      std::filesystem::perms::owner_write |
                      std::filesystem::perms::owner_exec;
    config.file_mode = std::filesystem::perms::owner_read |
                       std::filesystem::perms::owner_write;

    auto [kv, err] = kv::JSONFile::open(config);
    ASSERT_FALSE(err) << err.message();
    ASSERT_TRUE(std::filesystem::exists(temp_path));
}

TEST_F(JSONTest, SetGetDelete) {
    kv::JSONFileConfig config;
    config.path = temp_path;
    config.dir_mode = std::filesystem::perms::owner_read |
                      std::filesystem::perms::owner_write |
                      std::filesystem::perms::owner_exec;
    config.file_mode = std::filesystem::perms::owner_read |
                       std::filesystem::perms::owner_write;

    auto [kv, err] = kv::JSONFile::open(config);
    ASSERT_FALSE(err) << err.message();

    // Test set
    err = kv->set("key1", "value1");
    ASSERT_FALSE(err) << err.message();

    // Test get
    std::string value;
    err = kv->get("key1", value);
    ASSERT_FALSE(err) << err.message();
    ASSERT_EQ(value, "value1");

    // Test get non-existent key
    err = kv->get("nonexistent", value);
    ASSERT_TRUE(err);

    // Test delete
    err = kv->del("key1");
    ASSERT_FALSE(err) << err.message();

    // Verify key was deleted
    err = kv->get("key1", value);
    ASSERT_TRUE(err);

    // Test delete non-existent key (should not error)
    err = kv->del("nonexistent");
    ASSERT_FALSE(err) << err.message();
}

TEST_F(JSONTest, Persistence) {
    kv::JSONFileConfig config;
    config.path = temp_path;
    config.dir_mode = std::filesystem::perms::owner_read |
                      std::filesystem::perms::owner_write |
                      std::filesystem::perms::owner_exec;
    config.file_mode = std::filesystem::perms::owner_read |
                       std::filesystem::perms::owner_write;
    // Write some data
    {
        auto [kv, err] = kv::JSONFile::open(config);
        ASSERT_FALSE(err) << err.message();
        err = kv->set("persistent", "data");
        ASSERT_FALSE(err) << err.message();
    }

    // Read it back in a new instance
    {
        auto [kv, err] = kv::JSONFile::open(config);
        ASSERT_FALSE(err) << err.message();
        std::string value;
        err = kv->get("persistent", value);
        ASSERT_FALSE(err) << err.message();
        ASSERT_EQ(value, "data");
    }
}
