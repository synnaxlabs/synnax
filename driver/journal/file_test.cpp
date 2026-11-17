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

TEST_F(FileTest, CreatesFile) {
    const auto path = this->dir / "a";
    ASSERT_NIL(create(path, "first"));
    EXPECT_EQ(read_file(path), "first");
    EXPECT_FALSE(std::filesystem::exists(this->dir / "a.tmp"));
}

TEST_F(FileTest, CreateReplacesFile) {
    const auto path = this->dir / "a";
    ASSERT_NIL(create(path, "first"));
    ASSERT_NIL(create(path, "second"));
    EXPECT_EQ(read_file(path), "second");
}

TEST_F(FileTest, CreateDropsStaleTempFile) {
    const auto path = this->dir / "a";
    std::ofstream(this->dir / "a.tmp") << "stale";
    ASSERT_NIL(create(path, "new"));
    EXPECT_EQ(read_file(path), "new");
}

TEST_F(FileTest, ReplaceOverwritesTarget) {
    ASSERT_NIL(create(this->dir / "a", "a"));
    ASSERT_NIL(create(this->dir / "b", "b"));
    ASSERT_NIL(replace(this->dir / "a", this->dir / "b"));
    EXPECT_FALSE(std::filesystem::exists(this->dir / "a"));
    EXPECT_EQ(read_file(this->dir / "b"), "a");
}

TEST_F(FileTest, SyncsDirectory) {
    ASSERT_NIL(sync_dir(this->dir));
}

TEST_F(FileTest, LockIsExclusive) {
    const auto path = this->dir / "LOCK";
    auto first = ASSERT_NIL_P(File::open(path));
    ASSERT_NIL(first->lock());
    const auto second = ASSERT_NIL_P(File::open(path));
    ASSERT_OCCURRED_AS(second->lock(), LOCKED);
    first.reset();
    ASSERT_NIL(second->lock());
}
}
