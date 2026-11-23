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

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/arc/status/keys.h"

namespace driver::arc::status {
class KeysTest : public ::testing::Test {
protected:
    std::filesystem::path dir;
    std::filesystem::path path;

    void SetUp() override {
        this->dir = std::filesystem::temp_directory_path() /
                    ("status_keys_" + x::uuid::create().to_string());
        std::filesystem::create_directories(this->dir);
        this->path = this->dir / KEYS_FILE;
    }

    void TearDown() override { std::filesystem::remove_all(this->dir); }
};

/// @brief it should return the key of a known name after a restart.
TEST_F(KeysTest, KnownNameAfterRestart) {
    {
        const auto keys = ASSERT_NIL_P(Keys::open(this->path));
        ASSERT_NIL(keys->set("pressure", "key_1"));
        ASSERT_NIL(keys->set("pressure", "key_2"));
    }
    const auto keys = ASSERT_NIL_P(Keys::open(this->path));
    ASSERT_EQ(keys->get("pressure"), "key_2");
}

/// @brief it should return an empty key for a name the Core never resolved.
TEST_F(KeysTest, UnknownName) {
    const auto keys = ASSERT_NIL_P(Keys::open(this->path));
    ASSERT_EQ(keys->get("never_sent"), "");
    ASSERT_FALSE(std::filesystem::exists(this->path));
}

/// @brief it should forget the least recently used name past the maximum, also
/// across a restart.
TEST_F(KeysTest, EvictsLeastRecentlyUsed) {
    {
        const auto keys = ASSERT_NIL_P(Keys::open(this->path, 2));
        ASSERT_NIL(keys->set("a", "key_a"));
        ASSERT_NIL(keys->set("b", "key_b"));
        ASSERT_EQ(keys->get("a"), "key_a");
        ASSERT_NIL(keys->set("c", "key_c"));
        ASSERT_EQ(keys->get("b"), "");
    }
    const auto keys = ASSERT_NIL_P(Keys::open(this->path, 2));
    ASSERT_EQ(keys->get("a"), "key_a");
    ASSERT_EQ(keys->get("b"), "");
    ASSERT_EQ(keys->get("c"), "key_c");
}

/// @brief it should keep only the most recently used names when the maximum drops.
TEST_F(KeysTest, KeepsMostRecentWhenTheMaximumDrops) {
    {
        const auto keys = ASSERT_NIL_P(Keys::open(this->path, 3));
        ASSERT_NIL(keys->set("a", "key_a"));
        ASSERT_NIL(keys->set("b", "key_b"));
        ASSERT_NIL(keys->set("c", "key_c"));
    }
    const auto keys = ASSERT_NIL_P(Keys::open(this->path, 1));
    ASSERT_EQ(keys->get("a"), "");
    ASSERT_EQ(keys->get("b"), "");
    ASSERT_EQ(keys->get("c"), "key_c");
}

/// @brief it should refuse a file that a torn write cut off.
TEST_F(KeysTest, RefusesTornFile) {
    {
        const auto keys = ASSERT_NIL_P(Keys::open(this->path));
        ASSERT_NIL(keys->set("a", "key_a"));
        ASSERT_NIL(keys->set("b", "key_b"));
    }
    std::filesystem::resize_file(
        this->path,
        std::filesystem::file_size(this->path) - 1
    );
    ASSERT_OCCURRED_AS_P(Keys::open(this->path), journal::CORRUPT);
}

/// @brief it should refuse a file that is not a status keys file.
TEST_F(KeysTest, RefusesForeignFile) {
    {
        std::ofstream file(this->path, std::ios::binary);
        file << "not a status keys file";
    }
    ASSERT_OCCURRED_AS_P(Keys::open(this->path), journal::UNKNOWN_MAGIC);
}

/// @brief it should remove a temp file that a crash during a save left, and keep the
/// last saved keys.
TEST_F(KeysTest, RemovesTempFileOfACrashedSave) {
    {
        const auto keys = ASSERT_NIL_P(Keys::open(this->path));
        ASSERT_NIL(keys->set("a", "key_a"));
    }
    auto tmp = this->path;
    tmp += ".tmp";
    {
        std::ofstream file(tmp, std::ios::binary);
        file << "partial";
    }
    const auto keys = ASSERT_NIL_P(Keys::open(this->path));
    ASSERT_FALSE(std::filesystem::exists(tmp));
    ASSERT_EQ(keys->get("a"), "key_a");
}
}
