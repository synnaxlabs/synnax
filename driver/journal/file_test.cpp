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
#include <iterator>
#include <random>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/journal/file.h"

namespace driver::journal {
namespace {
std::string read_file(const std::filesystem::path &path) {
    std::ifstream f(path, std::ios::binary);
    return {std::istreambuf_iterator<char>(f), std::istreambuf_iterator<char>()};
}
}

class FileTest : public ::testing::Test {
protected:
    std::filesystem::path dir;

    void SetUp() override {
        std::random_device rd;
        this->dir = std::filesystem::temp_directory_path() /
                    ("journal_file_test_" + std::to_string(rd()));
        std::filesystem::create_directories(this->dir);
    }

    void TearDown() override { std::filesystem::remove_all(this->dir); }
};

/// @brief it should append data after the data already in the file.
TEST_F(FileTest, AppendsAndSyncs) {
    const auto path = this->dir / "a";
    {
        const auto file = ASSERT_NIL_P(File::open(path));
        ASSERT_NIL(file->write("ab"));
        ASSERT_NIL(file->sync());
    }
    {
        const auto file = ASSERT_NIL_P(File::open(path));
        ASSERT_NIL(file->write("cd"));
        ASSERT_NIL(file->sync());
    }
    EXPECT_EQ(read_file(path), "abcd");
}

/// @brief it should create a file that holds the data and leave no temp file.
TEST_F(FileTest, CreatesFile) {
    const auto path = this->dir / "a";
    ASSERT_NIL(create(path, "first"));
    EXPECT_EQ(read_file(path), "first");
    EXPECT_EQ(
        std::distance(
            std::filesystem::directory_iterator(this->dir),
            std::filesystem::directory_iterator()
        ),
        1
    );
}

/// @brief it should replace the data of an existing file.
TEST_F(FileTest, CreateReplacesFile) {
    const auto path = this->dir / "a";
    ASSERT_NIL(create(path, "first"));
    ASSERT_NIL(create(path, "second"));
    EXPECT_EQ(read_file(path), "second");
}

/// @brief it should create a file from a path with no directory.
TEST_F(FileTest, CreatesFileInWorkingDirectory) {
    const auto cwd = std::filesystem::current_path();
    std::filesystem::current_path(this->dir);
    const auto err = create("a", "data");
    std::filesystem::current_path(cwd);
    ASSERT_NIL(err);
    EXPECT_EQ(read_file(this->dir / "a"), "data");
}

/// @brief it should cut the file and append after the new end.
TEST_F(FileTest, TruncatesAndAppendsAtNewEnd) {
    const auto path = this->dir / "a";
    const auto file = ASSERT_NIL_P(File::open(path));
    ASSERT_NIL(file->write("abcd"));
    ASSERT_NIL(file->truncate(2));
    ASSERT_NIL(file->write("x"));
    ASSERT_NIL(file->sync());
    EXPECT_EQ(read_file(path), "abx");
}

/// @brief it should rename a file over an existing file.
TEST_F(FileTest, ReplaceOverwritesTarget) {
    ASSERT_NIL(create(this->dir / "a", "a"));
    ASSERT_NIL(create(this->dir / "b", "b"));
    ASSERT_NIL(replace(this->dir / "a", this->dir / "b"));
    EXPECT_FALSE(std::filesystem::exists(this->dir / "a"));
    EXPECT_EQ(read_file(this->dir / "b"), "a");
}

/// @brief it should sync a directory.
TEST_F(FileTest, SyncsDirectory) {
    ASSERT_NIL(sync_dir(this->dir));
}

/// @brief it should refuse a second lock until the first holder closes the file.
TEST_F(FileTest, LockIsExclusive) {
    const auto path = this->dir / "LOCK";
    auto first = ASSERT_NIL_P(File::open(path));
    ASSERT_NIL(first->lock());
    const auto second = ASSERT_NIL_P(File::open(path));
    ASSERT_OCCURRED_AS(second->lock(), LOCKED);
    first.reset();
    ASSERT_NIL(second->lock());
}

/// @brief it should let another handle read a file while it is locked.
TEST_F(FileTest, LockDoesNotBlockReads) {
    const auto path = this->dir / "a";
    const auto file = ASSERT_NIL_P(File::open(path));
    ASSERT_NIL(file->write("data"));
    ASSERT_NIL(file->lock());
    EXPECT_EQ(read_file(path), "data");
}

/// @brief it should remove its temp file when it fails.
TEST_F(FileTest, CreateRemovesTempFileOnFailure) {
    const auto path = this->dir / "a";
    std::filesystem::create_directories(path / "child");
    ASSERT_OCCURRED_AS(create(path, "data"), BASE_ERROR);
    EXPECT_EQ(
        std::distance(
            std::filesystem::directory_iterator(this->dir),
            std::filesystem::directory_iterator()
        ),
        1
    );
}
}
