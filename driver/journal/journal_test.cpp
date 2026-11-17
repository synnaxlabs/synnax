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
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/journal/journal.h"

namespace driver::journal {
namespace {
const Magic SEGMENT = {'S', 'Y', 'T', 'E', 'S', 'T', 'S', 'G'};
const Magic OTHER = {'S', 'Y', 'T', 'E', 'S', 'T', 'O', 'F'};
const std::filesystem::path TESTDATA = "driver/journal/testdata";

std::string read_file(const std::filesystem::path &path) {
    std::ifstream f(path, std::ios::binary);
    return {std::istreambuf_iterator<char>(f), std::istreambuf_iterator<char>()};
}

void write_file(const std::filesystem::path &path, const std::string &data) {
    std::ofstream f(path, std::ios::binary | std::ios::trunc);
    f.write(data.data(), static_cast<std::streamsize>(data.size()));
}

void append_file(const std::filesystem::path &path, const std::string &data) {
    std::ofstream f(path, std::ios::binary | std::ios::app);
    f.write(data.data(), static_cast<std::streamsize>(data.size()));
}

std::vector<Record> read_all(Reader &reader) {
    std::vector<Record> records;
    while (true) {
        auto record = ASSERT_NIL_P(reader.next());
        if (!record) return records;
        records.push_back(std::move(*record));
    }
}
}

class JournalTest : public ::testing::Test {
protected:
    std::filesystem::path dir;
    std::filesystem::path path;

    void SetUp() override {
        std::random_device rd;
        this->dir = std::filesystem::temp_directory_path() /
                    ("journal_test_" + std::to_string(rd()));
        std::filesystem::create_directories(this->dir);
        this->path = this->dir / "0.seg";
    }

    void TearDown() override { std::filesystem::remove_all(this->dir); }

    void write(const std::vector<Record> &records) const {
        auto writer = ASSERT_NIL_P(
            Writer::open({.path = this->path, .magic = SEGMENT})
        );
        for (const auto &record: records)
            ASSERT_NIL(writer->write(record));
        ASSERT_NIL(writer->close());
    }

    [[nodiscard]] std::unique_ptr<Reader>
    open_reader(std::set<std::uint8_t> types = {1}, std::uint64_t pos = 0) const {
        return ASSERT_NIL_P(
            Reader::open(
                {.path = this->path,
                 .magic = SEGMENT,
                 .types = std::move(types),
                 .pos = pos}
            )
        );
    }
};

/// @brief v1 files must stay readable by every later Driver.
TEST_F(JournalTest, ReadsV1Segment) {
    auto reader = ASSERT_NIL_P(
        Reader::open(
            {.path = TESTDATA / "v1_segment.bin", .magic = SEGMENT, .types = {1, 3}}
        )
    );
    EXPECT_EQ(reader->header.version, 1);
    EXPECT_EQ(reader->header.flags, 0x0102);
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 3);
    EXPECT_EQ(records[0].type, 1);
    EXPECT_EQ(records[0].payload, "hello");
    EXPECT_EQ(records[1].type, 1);
    EXPECT_EQ(records[1].payload, "");
    EXPECT_EQ(records[2].type, 3);
    EXPECT_EQ(records[2].payload, std::string("\x00\x01\x02", 3));
    EXPECT_FALSE(reader->torn());
    EXPECT_EQ(reader->pos(), 60);
}

TEST_F(JournalTest, WritesV1Segment) {
    auto writer = ASSERT_NIL_P(
        Writer::open({.path = this->path, .magic = SEGMENT, .flags = 0x0102})
    );
    ASSERT_NIL(writer->write({.type = 1, .payload = "hello"}));
    ASSERT_NIL(writer->write({.type = 0x82, .payload = "skip"}));
    ASSERT_NIL(writer->write({.type = 1, .payload = ""}));
    ASSERT_NIL(writer->write({.type = 3, .payload = std::string("\x00\x01\x02", 3)}));
    EXPECT_EQ(writer->pos(), HEADER_SIZE + 4 * RECORD_PREFIX_SIZE + 12);
    ASSERT_NIL(writer->close());
    EXPECT_EQ(read_file(this->path), read_file(TESTDATA / "v1_segment.bin"));
}

TEST_F(JournalTest, AppendsAfterReopen) {
    this->write({{.type = 1, .payload = "a"}});
    this->write({{.type = 1, .payload = "b"}});
    const auto reader = this->open_reader();
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 2);
    EXPECT_EQ(records[1].payload, "b");
}

TEST_F(JournalTest, ReadsFromPosition) {
    this->write({{.type = 1, .payload = "a"}, {.type = 1, .payload = "b"}});
    auto reader = this->open_reader();
    ASSERT_NIL_P(reader->next());
    const auto pos = reader->pos();
    reader = this->open_reader({1}, pos);
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 1);
    EXPECT_EQ(records[0].payload, "b");
}

TEST_F(JournalTest, PositionOutsideFileIsCorrupt) {
    this->write({{.type = 1, .payload = "a"}});
    ASSERT_OCCURRED_AS_P(
        Reader::open({.path = this->path, .magic = SEGMENT, .pos = 1000}),
        CORRUPT
    );
    ASSERT_OCCURRED_AS_P(
        Reader::open({.path = this->path, .magic = SEGMENT, .pos = 4}),
        CORRUPT
    );
}

TEST_F(JournalTest, RecoversCutOffTail) {
    this->write({{.type = 1, .payload = "a"}, {.type = 1, .payload = "b"}});
    const auto size = std::filesystem::file_size(this->path);
    append_file(this->path, std::string("\x10\x00\x00\x00\x01\x02", 6));
    auto reader = this->open_reader();
    EXPECT_EQ(read_all(*reader).size(), 2);
    EXPECT_TRUE(reader->torn());
    EXPECT_EQ(reader->pos(), size);
    reader.reset();
    this->write({{.type = 1, .payload = "c"}});
    reader = this->open_reader();
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 3);
    EXPECT_EQ(records[2].payload, "c");
    EXPECT_FALSE(reader->torn());
}

TEST_F(JournalTest, RecoversZeroFilledTail) {
    this->write({{.type = 1, .payload = "a"}});
    const auto size = std::filesystem::file_size(this->path);
    append_file(this->path, std::string(4096, '\0'));
    auto reader = this->open_reader();
    EXPECT_EQ(read_all(*reader).size(), 1);
    EXPECT_TRUE(reader->torn());
    reader.reset();
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    EXPECT_EQ(writer->pos(), size);
    ASSERT_NIL(writer->close());
    EXPECT_EQ(std::filesystem::file_size(this->path), size);
}

TEST_F(JournalTest, BadChecksumBeforeTailIsCorrupt) {
    this->write({{.type = 1, .payload = "abc"}, {.type = 1, .payload = "def"}});
    auto data = read_file(this->path);
    data[HEADER_SIZE + 9] = 'x';
    write_file(this->path, data);
    const auto reader = this->open_reader();
    ASSERT_OCCURRED_AS_P(reader->next(), CORRUPT);
    ASSERT_OCCURRED_AS_P(Writer::open({.path = this->path, .magic = SEGMENT}), CORRUPT);
}

TEST_F(JournalTest, SkipsUnknownSkippableType) {
    this->write(
        {{.type = 2 | SKIPPABLE, .payload = "new"}, {.type = 1, .payload = "old"}}
    );
    const auto reader = this->open_reader();
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 1);
    EXPECT_EQ(records[0].payload, "old");
}

TEST_F(JournalTest, UnknownTypeIsError) {
    this->write({{.type = 2, .payload = "new"}});
    const auto reader = this->open_reader();
    ASSERT_OCCURRED_AS_P(reader->next(), UNKNOWN_TYPE);
}

TEST_F(JournalTest, WriterRejectsTypeZero) {
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    ASSERT_OCCURRED_AS(writer->write({.type = SKIPPABLE}), ERROR);
    ASSERT_NIL(writer->write({.type = 1}));
}

TEST_F(JournalTest, BadMagicIsUnknownMagic) {
    this->write({});
    ASSERT_OCCURRED_AS_P(
        Reader::open({.path = this->path, .magic = OTHER}),
        UNKNOWN_MAGIC
    );
    ASSERT_OCCURRED_AS_P(
        Writer::open({.path = this->path, .magic = OTHER}),
        UNKNOWN_MAGIC
    );
}

TEST_F(JournalTest, NewerVersionIsUnknownVersion) {
    this->write({});
    auto data = read_file(this->path);
    data[8] = 2;
    write_file(this->path, data);
    ASSERT_OCCURRED_AS_P(
        Reader::open({.path = this->path, .magic = SEGMENT}),
        UNKNOWN_VERSION
    );
    ASSERT_OCCURRED_AS_P(
        Writer::open({.path = this->path, .magic = SEGMENT}),
        UNKNOWN_VERSION
    );
}

TEST_F(JournalTest, CutOffFileStartIsCorrupt) {
    write_file(this->path, "SYTE");
    ASSERT_OCCURRED_AS_P(Reader::open({.path = this->path, .magic = SEGMENT}), CORRUPT);
}

TEST_F(JournalTest, SyncsByTimeAndBytes) {
    auto now = x::telem::TimeStamp(0);
    int syncs = 0;
    auto writer = ASSERT_NIL_P(
        Writer::open({
            .path = this->path,
            .magic = SEGMENT,
            .sync_bytes = 100,
            .now = [&] { return now; },
            .on_sync =
                [&](const x::errors::Error &err) {
                    syncs++;
                    return err;
                },
        })
    );
    ASSERT_NIL(writer->write({.type = 1, .payload = "a"}));
    EXPECT_EQ(syncs, 0);
    now = now + x::telem::SECOND;
    ASSERT_NIL(writer->write({.type = 1, .payload = "b"}));
    EXPECT_EQ(syncs, 1);
    ASSERT_NIL(writer->write({.type = 1, .payload = std::string(100, 'c')}));
    EXPECT_EQ(syncs, 2);
    ASSERT_NIL(writer->write({.type = 1, .payload = "d"}));
    ASSERT_NIL(writer->sync_if_due());
    EXPECT_EQ(syncs, 2);
    now = now + x::telem::SECOND;
    ASSERT_NIL(writer->sync_if_due());
    EXPECT_EQ(syncs, 3);
    ASSERT_NIL(writer->sync_if_due());
    EXPECT_EQ(syncs, 3);
    ASSERT_NIL(writer->write({.type = 1, .payload = "e"}));
    ASSERT_NIL(writer->close());
    EXPECT_EQ(syncs, 4);
}

TEST_F(JournalTest, SyncErrorIsFatal) {
    const auto injected = x::errors::Error(ERROR, "disk failed");
    int syncs = 0;
    auto writer = ASSERT_NIL_P(
        Writer::open({
            .path = this->path,
            .magic = SEGMENT,
            .sync_bytes = 1,
            .on_sync = [&](const x::errors::Error &) {
                syncs++;
                return injected;
            },
        })
    );
    ASSERT_OCCURRED_AS(writer->write({.type = 1, .payload = "a"}), ERROR);
    ASSERT_OCCURRED_AS(writer->write({.type = 1, .payload = "b"}), ERROR);
    ASSERT_OCCURRED_AS(writer->close(), ERROR);
    EXPECT_EQ(syncs, 1);
}
}
