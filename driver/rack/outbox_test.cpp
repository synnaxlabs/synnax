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

#ifndef _WIN32
#include <unistd.h>
#endif

#include "driver/journal/journal.h"
#include "driver/outbox/outbox.h"
#include "driver/rack/rack.h"

namespace driver::rack {
class OutboxFailureTest : public ::testing::Test {
protected:
    std::filesystem::path root;
    std::filesystem::path dir;

    void SetUp() override {
        this->root = std::filesystem::temp_directory_path() /
                     ("rack_outbox_" + x::uuid::create().to_string());
        this->dir = this->root / OUTBOX_DIR;
        std::filesystem::create_directories(this->dir);
    }

    void TearDown() override {
        std::error_code ec;
        std::filesystem::permissions(this->root, std::filesystem::perms::owner_all, ec);
        std::filesystem::remove_all(this->root, ec);
    }

    /// @returns the error of opening the outbox in dir.
    [[nodiscard]] x::errors::Error open_err() const {
        return outbox::Outbox::open({.dir = this->dir}).second;
    }
};

/// @brief it should tell the user to update the Driver when a newer Driver wrote the
/// outbox, and leave its files in place.
TEST_F(OutboxFailureTest, UnknownVersion) {
    {
        std::ofstream file(this->dir / "0.calls", std::ios::binary);
        file.write(outbox::SEGMENT_MAGIC.data(), outbox::SEGMENT_MAGIC.size());
        file.write("\x02\x00\x00\x00", 4);
    }
    const auto err = this->open_err();
    ASSERT_OCCURRED_AS(err, journal::UNKNOWN_VERSION);
    const auto failure = outbox_failure(this->dir, err);
    ASSERT_OCCURRED_AS(failure, journal::UNKNOWN_VERSION);
    ASSERT_EQ(
        failure.data,
        "The Driver cannot open its outbox in " + this->dir.string() + ": " + err.data +
            ". A newer Driver wrote it. Update the Driver, or move " +
            this->dir.string() +
            " aside to discard the changes it holds for the Core. Then restart the "
            "Driver."
    );
    ASSERT_EQ(std::filesystem::file_size(this->dir / "0.calls"), 12);
}

/// @brief it should tell the user to move a damaged outbox aside.
TEST_F(OutboxFailureTest, Corrupt) {
    ASSERT_NIL(outbox::Outbox::open({.dir = this->dir}).second);
    ASSERT_NIL(
        journal::commit_offset(
            this->dir / "offset",
            outbox::OFFSET_MAGIC,
            {.segment = 0, .pos = 1 << 30}
        )
    );
    const auto err = this->open_err();
    ASSERT_OCCURRED_AS(err, journal::CORRUPT);
    const auto failure = outbox_failure(this->dir, err);
    ASSERT_OCCURRED_AS(failure, journal::CORRUPT);
    ASSERT_EQ(
        failure.data,
        "The Driver cannot open its outbox in " + this->dir.string() + ": " + err.data +
            ". Its files are damaged. Move " + this->dir.string() +
            " aside to discard the changes it holds for the Core. Then restart the "
            "Driver."
    );
}

/// @brief it should tell the user to stop the other Driver that uses the outbox.
TEST_F(OutboxFailureTest, Locked) {
    const auto [held, held_err] = outbox::Outbox::open({.dir = this->dir});
    ASSERT_NIL(held_err);
    const auto err = this->open_err();
    ASSERT_OCCURRED_AS(err, journal::LOCKED);
    const auto failure = outbox_failure(this->dir, err);
    ASSERT_OCCURRED_AS(failure, journal::LOCKED);
    ASSERT_EQ(
        failure.data,
        "The Driver cannot open its outbox in " + this->dir.string() + ": " + err.data +
            ". Another Driver uses it. Stop the other Driver, or give each Driver "
            "its own state directory. Then restart the Driver."
    );
}

/// @brief it should tell the user to let the Driver write the outbox directory.
TEST_F(OutboxFailureTest, UnwritableDirectory) {
#ifdef _WIN32
    GTEST_SKIP() << "POSIX modes do not apply on Windows";
#else
    if (geteuid() == 0) GTEST_SKIP() << "root writes any directory";
    std::filesystem::remove_all(this->dir);
    std::filesystem::permissions(
        this->root,
        std::filesystem::perms::owner_read | std::filesystem::perms::owner_exec
    );
    const auto err = this->open_err();
    ASSERT_TRUE(err);
    ASSERT_EQ(
        outbox_failure(this->dir, err).data,
        "The Driver cannot open its outbox in " + this->dir.string() + ": " + err.data +
            ". Make sure that the Driver can write to " + this->dir.string() +
            " and that the disk has free space. On Linux, run 'sudo synnax-driver "
            "install' to update the service. Then restart the Driver."
    );
#endif
}
}
