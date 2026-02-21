// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>

#include "x/cpp/kv/kv.h"
#include "x/cpp/test/test.h"

namespace x::kv {
class JSONTest : public ::testing::Test {
protected:
    std::string temp_path;

    void SetUp() override {
        const auto temp_dir = std::filesystem::temp_directory_path();
        temp_path = (temp_dir / "json_test" / "test.json").string();
    }

    void TearDown() override {
        try {
            std::filesystem::remove_all(std::filesystem::path(temp_path).parent_path());
        } catch (const std::filesystem::filesystem_error &e) {
            std::cerr << "Cleanup failed: " << e.what() << std::endl;
        }
    }
};

/// @brief it should create a new JSON file when it does not exist.
TEST_F(JSONTest, CreateNewFile) {
    JSONFileConfig config;
    config.path = temp_path;
    config.dir_mode = std::filesystem::perms::owner_read |
                      std::filesystem::perms::owner_write |
                      std::filesystem::perms::owner_exec;
    config.file_mode = std::filesystem::perms::owner_read |
                       std::filesystem::perms::owner_write;

    auto kv = ASSERT_NIL_P(JSONFile::open(config));
    ASSERT_TRUE(std::filesystem::exists(temp_path));
}

/// @brief it should correctly set, get, and delete key-value pairs.
TEST_F(JSONTest, SetGetDelete) {
    JSONFileConfig config;
    config.path = temp_path;
    config.dir_mode = std::filesystem::perms::owner_read |
                      std::filesystem::perms::owner_write |
                      std::filesystem::perms::owner_exec;
    config.file_mode = std::filesystem::perms::owner_read |
                       std::filesystem::perms::owner_write;

    auto kv = ASSERT_NIL_P(JSONFile::open(config));

    ASSERT_NIL(kv->set("key1", "value1"));

    // Test get
    std::string value;
    ASSERT_NIL(kv->get("key1", value));
    ASSERT_OCCURRED_AS(kv->get("nonexistent", value), errors::NOT_FOUND);

    // Test delete
    ASSERT_NIL(kv->del("key1"));

    // Verify key was deleted
    ASSERT_OCCURRED_AS(kv->get("key1", value), errors::NOT_FOUND);

    // Test delete non-existent key (should not error)
    ASSERT_NIL(kv->del("nonexistent"));
}

/// @brief it should persist data across multiple file instances.
TEST_F(JSONTest, Persistence) {
    JSONFileConfig config;
    config.path = temp_path;
    config.dir_mode = std::filesystem::perms::owner_read |
                      std::filesystem::perms::owner_write |
                      std::filesystem::perms::owner_exec;
    config.file_mode = std::filesystem::perms::owner_read |
                       std::filesystem::perms::owner_write;
    {
        const auto kv = ASSERT_NIL_P(JSONFile::open(config));
        ASSERT_NIL(kv->set("persistent", "data"));
    }
    {
        const auto kv = ASSERT_NIL_P(JSONFile::open(config));
        std::string value;
        ASSERT_NIL(kv->get("persistent", value));
        ASSERT_EQ(value, "data");
    }
}
}
