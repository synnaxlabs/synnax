// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <filesystem>
#include <fstream>
#include <string>

#include "absl/crc/crc32c.h"
#include "gtest/gtest.h"

#include "freighter/cpp/freighter.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#ifndef _WIN32
#include <unistd.h>
#endif

#include "driver/journal/journal.h"
#include "driver/outbox/outbox.h"
#include "driver/rack/rack.h"

namespace driver::rack {
/// @brief writes a segment file start at path with the format version and flags, which
/// a newer Driver writes.
void write_future_segment(
    const std::filesystem::path &path,
    const std::uint16_t version = 2,
    const std::uint16_t flags = 0
) {
    std::string header(outbox::SEGMENT_MAGIC.data(), outbox::SEGMENT_MAGIC.size());
    for (const auto v: {version, flags}) {
        header.push_back(static_cast<char>(v & 0xFF));
        header.push_back(static_cast<char>(v >> 8));
    }
    header += std::string(journal::HEADER_SIZE - 16, '\0');
    const auto crc = static_cast<std::uint32_t>(absl::ComputeCrc32c(header));
    for (int i = 0; i < 4; i++)
        header.push_back(static_cast<char>(crc >> (8 * i)));
    std::ofstream(path, std::ios::binary) << header;
}

/// @returns the text of err as a sentence, as outbox_failure gives it.
std::string sentence(const x::errors::Error &err) {
    return err.data.ends_with('.') ? err.data : err.data + ".";
}

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
    write_future_segment(this->dir / "0.calls");
    const auto err = this->open_err();
    ASSERT_OCCURRED_AS(err, journal::UNKNOWN_VERSION);
    const auto failure = outbox_failure(this->dir, err);
    ASSERT_OCCURRED_AS(failure, journal::UNKNOWN_VERSION);
    ASSERT_EQ(
        failure.data,
        "The Driver cannot open its outbox in " + this->dir.string() + ": " +
            sentence(err) + " A newer Driver wrote it. Update the Driver, or move " +
            this->dir.string() +
            " aside to discard the changes it holds for the Core. Then restart the "
            "Driver."
    );
    ASSERT_EQ(std::filesystem::file_size(this->dir / "0.calls"), journal::HEADER_SIZE);
}

/// @brief it should tell the user to update the Driver when a newer Driver set a flag
/// that this one must know to read the outbox.
TEST_F(OutboxFailureTest, UnknownFlag) {
    write_future_segment(this->dir / "0.calls", journal::VERSION, 0x0100);
    const auto err = this->open_err();
    ASSERT_OCCURRED_AS(err, journal::UNKNOWN_FLAG);
    const auto text = outbox_failure(this->dir, err).data;
    EXPECT_NE(text.find("A newer Driver wrote it."), std::string::npos);
    EXPECT_EQ(text.find(".."), std::string::npos) << text;
}

/// @brief it should tell the user to move a damaged outbox aside.
TEST_F(OutboxFailureTest, Corrupt) {
    ASSERT_NIL_P(outbox::Outbox::open({.dir = this->dir}));
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
        "The Driver cannot open its outbox in " + this->dir.string() + ": " +
            sentence(err) + " Its files are damaged. Move " + this->dir.string() +
            " aside to discard the changes it holds for the Core. Then restart the "
            "Driver."
    );
}

/// @brief it should tell the user to stop the other Driver that uses the outbox.
TEST_F(OutboxFailureTest, Locked) {
    const auto held = ASSERT_NIL_P(outbox::Outbox::open({.dir = this->dir}));
    const auto err = this->open_err();
    ASSERT_OCCURRED_AS(err, x::fs::LOCKED);
    const auto failure = outbox_failure(this->dir, err);
    ASSERT_OCCURRED_AS(failure, x::fs::LOCKED);
    ASSERT_EQ(
        failure.data,
        "The Driver cannot open its outbox in " + this->dir.string() + ": " +
            sentence(err) +
            " Another Driver uses it. Stop the other Driver, or give each Driver "
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
    ASSERT_OCCURRED_AS(err, outbox::BASE_ERROR);
    ASSERT_EQ(
        outbox_failure(this->dir, err).data,
        "The Driver cannot open its outbox in " + this->dir.string() + ": " +
            sentence(err) + " Make sure that the Driver can write to " +
            this->dir.string() +
            " and that the disk has free space. On Linux, run 'sudo synnax-driver "
            "install' to update the service. Then restart the Driver."
    );
#endif
}

/// @brief it should add no second full stop after a cause that ends in one.
TEST(OutboxStuckTest, KeepsOneFullStop) {
    const auto health = outbox_stuck(
        "/data/outbox",
        x::telem::MINUTE,
        x::errors::Error(freighter::UNREACHABLE, "connection refused.")
    );
    EXPECT_FALSE(health.description.contains("..")) << health.description;
}

/// @brief it should warn that the outbox is stuck, with the error and the fix.
TEST(OutboxStuckTest, Warns) {
    const auto health = outbox_stuck(
        "/data/outbox",
        5 * x::telem::MINUTE,
        x::errors::Error(freighter::UNREACHABLE, "connection refused")
    );
    EXPECT_EQ(health.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(health.message, OUTBOX_STUCK_MSG);
    EXPECT_EQ(
        health.description,
        "The Core has not taken the first Arc status set or range change for " +
            (5 * x::telem::MINUTE).to_string() +
            ": connection refused. Check that the Driver can reach the Core and that "
            "the Core is running. The Driver keeps them in /data/outbox and sends them "
            "once the Core takes them."
    );
}
}
