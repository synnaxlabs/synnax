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
#include <string>

#include "gtest/gtest.h"
#include <unistd.h>

#include "x/cpp/testutil/testutil.h"

#include "driver/daemon/common.h"

namespace fs = std::filesystem;

namespace driver::daemon {
namespace {
std::string read(const fs::path &path) {
    std::ifstream f(path, std::ios::binary);
    return {std::istreambuf_iterator<char>(f), std::istreambuf_iterator<char>()};
}

void write(const fs::path &path, const std::string &data) {
    std::ofstream f(path, std::ios::binary);
    f << data;
}

class InstallBinaryTest : public ::testing::Test {
protected:
    fs::path dir;

    void SetUp() override {
        this->dir = fs::temp_directory_path() /
                    ("install_binary_test_" + std::to_string(getpid()));
        fs::remove_all(this->dir);
        fs::create_directories(this->dir);
    }

    void TearDown() override { fs::remove_all(this->dir); }
};
}

/// @brief it should copy the binary into a new directory and make it executable.
TEST_F(InstallBinaryTest, CopiesBinary) {
    write(this->dir / "src", "binary");
    const auto dst = this->dir / "bin" / "dst";
    ASSERT_NIL(install_binary(this->dir / "src", dst, getuid(), getgid()));
    EXPECT_EQ(read(dst), "binary");
    EXPECT_EQ(fs::status(dst).permissions(), static_cast<fs::perms>(0755));
}

/// @brief it should install a binary over itself.
TEST_F(InstallBinaryTest, InstallsOverItself) {
    const auto dst = this->dir / "dst";
    write(dst, "binary");
    ASSERT_NIL(install_binary(dst, dst, getuid(), getgid()));
    EXPECT_EQ(read(dst), "binary");
}

/// @brief it should not write through a link at the path of its temporary copy.
TEST_F(InstallBinaryTest, DoesNotFollowLink) {
    write(this->dir / "src", "binary");
    write(this->dir / "target", "target");
    fs::create_symlink(
        this->dir / "target",
        this->dir / ("dst." + std::to_string(getpid()) + ".tmp")
    );
    ASSERT_NIL(
        install_binary(this->dir / "src", this->dir / "dst", getuid(), getgid())
    );
    EXPECT_EQ(read(this->dir / "target"), "target");
    EXPECT_EQ(read(this->dir / "dst"), "binary");
}
}
