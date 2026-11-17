// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>
#include <string>

#include "x/cpp/kv/kv.h"
#include "x/cpp/testutil/testutil.h"

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

    auto kv = ASSERT_NIL_P(JSONFile::open_file(config));
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

    auto kv = ASSERT_NIL_P(JSONFile::open_file(config));

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
        const auto kv = ASSERT_NIL_P(JSONFile::open_file(config));
        ASSERT_NIL(kv->set("persistent", "data"));
    }
    {
        const auto kv = ASSERT_NIL_P(JSONFile::open_file(config));
        std::string value;
        ASSERT_NIL(kv->get("persistent", value));
        ASSERT_EQ(value, "data");
    }
}

/// @brief it should keep the mode of an existing file when it opens and writes it.
TEST_F(JSONTest, ExistingFileMode) {
#ifdef _WIN32
    GTEST_SKIP() << "Windows files have no POSIX mode";
#endif
    JSONFileConfig config;
    config.path = temp_path;
    config.dir_mode = std::filesystem::perms::owner_all;
    const auto mode = std::filesystem::perms::owner_read |
                      std::filesystem::perms::owner_write |
                      std::filesystem::perms::others_read;
    config.file_mode = mode;
    ASSERT_NIL_P(JSONFile::open_file(config));
    config.file_mode = std::filesystem::perms::owner_read |
                       std::filesystem::perms::owner_write;
    const auto kv = ASSERT_NIL_P(JSONFile::open_file(config));
    ASSERT_NIL(kv->set("key", "value"));
    ASSERT_EQ(
        std::filesystem::status(temp_path).permissions() & std::filesystem::perms::all,
        mode
    );
}

/// @brief it should refuse to write through a link at the path.
TEST_F(JSONTest, WriteRefusesLink) {
#ifdef _WIN32
    GTEST_SKIP() << "Windows writes through the path";
#endif
    JSONFileConfig config;
    config.path = temp_path;
    config.dir_mode = std::filesystem::perms::owner_all;
    config.file_mode = std::filesystem::perms::owner_read |
                       std::filesystem::perms::owner_write;
    const auto kv = ASSERT_NIL_P(JSONFile::open_file(config));
    const auto target = std::filesystem::path(temp_path).parent_path() / "target.json";
    std::ofstream(target) << "target";
    std::filesystem::remove(temp_path);
    std::filesystem::create_symlink(target, temp_path);
    const auto err = kv->set("key", "value");
    ASSERT_NE(err.message().find("failed to open file"), std::string::npos);
    std::ifstream file(target);
    std::string content;
    file >> content;
    ASSERT_EQ(content, "target");
}

/// @brief it should write a file in a directory that the process cannot write.
TEST_F(JSONTest, WriteInReadOnlyDirectory) {
#ifdef _WIN32
    GTEST_SKIP() << "Windows directories have no POSIX mode";
#endif
    JSONFileConfig config;
    config.path = temp_path;
    config.dir_mode = std::filesystem::perms::owner_all;
    config.file_mode = std::filesystem::perms::owner_read |
                       std::filesystem::perms::owner_write;
    const auto kv = ASSERT_NIL_P(JSONFile::open_file(config));
    const auto dir = std::filesystem::path(temp_path).parent_path();
    std::filesystem::permissions(
        dir,
        std::filesystem::perms::owner_read | std::filesystem::perms::owner_exec
    );
    const auto err = kv->set("key", "value");
    std::filesystem::permissions(dir, std::filesystem::perms::owner_all);
    ASSERT_NIL(err);
    const auto reopened = ASSERT_NIL_P(JSONFile::open_file(config));
    std::string value;
    ASSERT_NIL(reopened->get("key", value));
    ASSERT_EQ(value, "value");
}

/// @brief it should refuse to open a link at the path.
TEST_F(JSONTest, OpenRefusesLink) {
#ifdef _WIN32
    GTEST_SKIP() << "Windows reads through the path";
#endif
    const auto dir = std::filesystem::path(temp_path).parent_path();
    std::filesystem::create_directories(dir);
    std::ofstream(dir / "target.json") << "{}";
    std::filesystem::create_symlink(dir / "target.json", temp_path);
    JSONFileConfig config;
    config.path = temp_path;
    config.dir_mode = std::filesystem::perms::owner_all;
    config.file_mode = std::filesystem::perms::owner_read |
                       std::filesystem::perms::owner_write;
    auto [kv, err] = JSONFile::open_file(config);
    ASSERT_NE(err.message().find("failed to open file"), std::string::npos);
}
}
