// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <iterator>
#include <random>
#include <string>
#include <thread>
#include <vector>

#include "absl/crc/crc32c.h"
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

/// @brief returns the little-endian bytes of value.
std::string le32(const std::uint32_t value) {
    std::string buf;
    for (int i = 0; i < 4; i++)
        buf.push_back(static_cast<char>(value >> (8 * i)));
    return buf;
}

/// @brief returns the checksum salt of the file at path.
std::uint32_t salt_of(const std::filesystem::path &path) {
    const auto data = read_file(path);
    std::uint32_t salt = 0;
    for (int i = 0; i < 4; i++)
        salt |= static_cast<std::uint32_t>(static_cast<std::uint8_t>(data[12 + i]))
             << (8 * i);
    return salt;
}

/// @brief encodes a record with a valid checksum for salt, which the writer would
/// refuse.
std::string
encode(const std::uint32_t salt, const std::uint8_t type, const std::string &payload) {
    std::string len(4, '\0');
    len[0] = static_cast<char>(payload.size());
    const auto crc = static_cast<std::uint32_t>(absl::ExtendCrc32c(
        absl::crc32c_t{salt},
        len + static_cast<char>(type) + payload
    ));
    return len + le32(crc) + static_cast<char>(type) + payload;
}

/// @brief writes the file start of the file at path with version and flags, and a
/// valid checksum.
void set_header(
    const std::filesystem::path &path,
    const std::uint16_t version,
    const std::uint16_t flags
) {
    auto data = read_file(path);
    data[8] = static_cast<char>(version);
    data[9] = static_cast<char>(version >> 8);
    data[10] = static_cast<char>(flags);
    data[11] = static_cast<char>(flags >> 8);
    const auto crc = static_cast<std::uint32_t>(
        absl::ComputeCrc32c(data.substr(0, 16))
    );
    data.replace(16, 4, le32(crc));
    write_file(path, data);
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

/// @brief it should read a v1 segment, which every later Driver must read.
TEST_F(JournalTest, ReadsV1Segment) {
    auto reader = ASSERT_NIL_P(
        Reader::open(
            {.path = TESTDATA / "v1_segment.bin",
             .magic = SEGMENT,
             .types = {1, 3},
             .flags = 0x0100}
        )
    );
    EXPECT_EQ(reader->header.version, 1);
    EXPECT_EQ(reader->header.flags, 0x0102);
    EXPECT_EQ(reader->header.salt, 0x5A17C0DE);
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 3);
    EXPECT_EQ(records[0].type, 1);
    EXPECT_EQ(records[0].payload, "hello");
    EXPECT_EQ(records[1].type, 1);
    EXPECT_EQ(records[1].payload, "");
    EXPECT_EQ(records[2].type, 3);
    EXPECT_EQ(records[2].payload, std::string("\x00\x01\x02", 3));
    EXPECT_EQ(reader->cut(), 0);
    EXPECT_EQ(reader->pos(), 68);
}

/// @brief it should write the same bytes as the v1 segment fixture.
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
    // Each file has its own salt, so only the bytes before it match the fixture.
    const auto written = read_file(this->path);
    const auto fixture = read_file(TESTDATA / "v1_segment.bin");
    EXPECT_EQ(written.size(), fixture.size());
    EXPECT_EQ(written.substr(0, 12), fixture.substr(0, 12));
    const auto reader = ASSERT_NIL_P(
        Reader::open(
            {.path = this->path, .magic = SEGMENT, .types = {1, 3}, .flags = 0x0100}
        )
    );
    EXPECT_EQ(read_all(*reader).size(), 3);
}

/// @brief it should fail the checksum of a record copied from another file.
TEST_F(JournalTest, RecordFromAnotherFileFailsChecksum) {
    this->write({{.type = 1, .payload = "a"}});
    const auto other = this->dir / "1.seg";
    {
        auto writer = ASSERT_NIL_P(Writer::open({.path = other, .magic = SEGMENT}));
        ASSERT_NIL(writer->write({.type = 1, .payload = "copied"}));
        ASSERT_NIL(writer->close());
    }
    append_file(this->path, read_file(other).substr(HEADER_SIZE));
    const auto reader = this->open_reader();
    EXPECT_EQ(read_all(*reader).size(), 1);
    EXPECT_GT(reader->cut(), 0);
}

/// @brief it should return CORRUPT for a file start with a flipped bit.
TEST_F(JournalTest, DamagedFileStartIsCorrupt) {
    this->write({});
    auto data = read_file(this->path);
    data[10] = static_cast<char>(data[10] ^ 0x01);
    write_file(this->path, data);
    ASSERT_OCCURRED_AS_P(Reader::open({.path = this->path, .magic = SEGMENT}), CORRUPT);
    ASSERT_OCCURRED_AS_P(Writer::open({.path = this->path, .magic = SEGMENT}), CORRUPT);
}

/// @brief it should refuse an unknown required flag and ignore an unknown other flag.
TEST_F(JournalTest, RefusesUnknownRequiredFlag) {
    this->write({{.type = 1, .payload = "a"}});
    set_header(this->path, VERSION, 0x0001);
    EXPECT_EQ(read_all(*this->open_reader()).size(), 1);
    set_header(this->path, VERSION, 0x0200);
    ASSERT_OCCURRED_AS_P(
        Reader::open({.path = this->path, .magic = SEGMENT}),
        UNKNOWN_FLAG
    );
    ASSERT_OCCURRED_AS_P(
        Writer::open({.path = this->path, .magic = SEGMENT}),
        UNKNOWN_FLAG
    );
    ASSERT_NIL_P(Reader::open({.path = this->path, .magic = SEGMENT, .flags = 0x0200}));
}

/// @brief it should append after the records of an existing file.
TEST_F(JournalTest, AppendsAfterReopen) {
    this->write({{.type = 1, .payload = "a"}});
    this->write({{.type = 1, .payload = "b"}});
    const auto reader = this->open_reader();
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 2);
    EXPECT_EQ(records[1].payload, "b");
}

/// @brief it should start reading at a position from an earlier reader.
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

/// @brief it should refuse a position outside the records of the file.
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

/// @brief it should cut a record that is cut off at the end of the file.
TEST_F(JournalTest, RecoversCutOffTail) {
    this->write({{.type = 1, .payload = "a"}, {.type = 1, .payload = "b"}});
    const auto size = std::filesystem::file_size(this->path);
    append_file(this->path, std::string("\x10\x00\x00\x00\x01\x02", 6));
    auto reader = this->open_reader();
    EXPECT_EQ(read_all(*reader).size(), 2);
    EXPECT_EQ(reader->cut(), 6);
    EXPECT_EQ(reader->pos(), size);
    reader.reset();
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    EXPECT_EQ(writer->cut(), 6);
    ASSERT_NIL(writer->write({.type = 1, .payload = "c"}));
    ASSERT_NIL(writer->close());
    reader = this->open_reader();
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 3);
    EXPECT_EQ(records[2].payload, "c");
    EXPECT_EQ(reader->cut(), 0);
}

/// @brief it should cut a zero-filled tail.
TEST_F(JournalTest, RecoversZeroFilledTail) {
    this->write({{.type = 1, .payload = "a"}});
    const auto size = std::filesystem::file_size(this->path);
    append_file(this->path, std::string(4096, '\0'));
    auto reader = this->open_reader();
    EXPECT_EQ(read_all(*reader).size(), 1);
    EXPECT_EQ(reader->cut(), 4096);
    reader.reset();
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    EXPECT_EQ(writer->pos(), size);
    EXPECT_EQ(writer->cut(), 4096);
    ASSERT_NIL(writer->close());
    EXPECT_EQ(std::filesystem::file_size(this->path), size);
}

/// @brief it should cut at a damaged record in the bytes a crash can leave unsynced,
/// with the valid records after it, as out of order writeback leaves them.
TEST_F(JournalTest, CutsAtHoleBeforeUnsyncedRecords) {
    this->write(
        {{.type = 1, .payload = "abc"},
         {.type = 1, .payload = "def"},
         {.type = 1, .payload = "ghi"}}
    );
    const auto size = std::filesystem::file_size(this->path);
    const auto second = HEADER_SIZE + RECORD_PREFIX_SIZE + 3;
    auto data = read_file(this->path);
    data[second + RECORD_PREFIX_SIZE] = 'x';
    write_file(this->path, data);
    auto reader = this->open_reader();
    EXPECT_EQ(read_all(*reader).size(), 1);
    EXPECT_EQ(reader->cut(), size - second);
    reader.reset();
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    EXPECT_EQ(writer->cut(), size - second);
    EXPECT_EQ(writer->pos(), second);
    ASSERT_NIL(writer->close());
    EXPECT_EQ(std::filesystem::file_size(this->path), second);
}

/// @brief it should return CORRUPT for a damaged record before the bytes a crash can
/// leave unsynced, and leave the file unchanged.
TEST_F(JournalTest, DamageBeforeUnsyncedBytesIsCorrupt) {
    this->write({});
    auto bad = encode(salt_of(this->path), 1, "abc");
    bad[RECORD_PREFIX_SIZE] = 'x';
    append_file(this->path, bad);
    const auto size = HEADER_SIZE + bad.size() + RECORD_PREFIX_SIZE + MAX_PAYLOAD + 1;
    // A sparse tail puts the damage before the unsynced bytes without writing them.
    std::filesystem::resize_file(this->path, size);
    const auto reader = ASSERT_NIL_P(
        Reader::open(
            {.path = this->path, .magic = SEGMENT, .types = {1}, .sync_bytes = 1}
        )
    );
    ASSERT_OCCURRED_AS_P(reader->next(), CORRUPT);
    ASSERT_OCCURRED_AS_P(
        Writer::open({.path = this->path, .magic = SEGMENT, .sync_bytes = 1}),
        CORRUPT
    );
    EXPECT_EQ(std::filesystem::file_size(this->path), size);
}

/// @brief it should cut a damaged last record that has only zeros after it.
TEST_F(JournalTest, RecoversBadChecksumAtEnd) {
    this->write({{.type = 1, .payload = "abc"}, {.type = 1, .payload = "def"}});
    const auto second = HEADER_SIZE + RECORD_PREFIX_SIZE + 3;
    auto data = read_file(this->path);
    data[second + RECORD_PREFIX_SIZE] = 'x';
    write_file(this->path, data + std::string(50, '\0'));
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    EXPECT_EQ(writer->pos(), second);
    EXPECT_EQ(writer->cut(), data.size() + 50 - second);
}

/// @brief it should read a position at a record that a crash cut off as the end of
/// the file.
TEST_F(JournalTest, PositionAtCutOffRecordIsEnd) {
    this->write({{.type = 1, .payload = "a"}});
    const auto end = std::filesystem::file_size(this->path);
    append_file(
        this->path,
        encode(salt_of(this->path), 1, "bcd").substr(0, RECORD_PREFIX_SIZE + 1)
    );
    const auto reader = this->open_reader({1}, end);
    EXPECT_FALSE(ASSERT_NIL_P(reader->next()).has_value());
    EXPECT_EQ(reader->cut(), RECORD_PREFIX_SIZE + 1);
}

/// @brief it should return an error when the file is shorter than at open.
TEST_F(JournalTest, ShortReadIsError) {
    this->write({{.type = 1, .payload = std::string(std::size_t{100} * 1024, 'a')}});
    const auto reader = this->open_reader();
    std::filesystem::resize_file(this->path, HEADER_SIZE + 20);
    ASSERT_OCCURRED_AS_P(reader->next(), BASE_ERROR);
    EXPECT_EQ(reader->cut(), 0);
}

/// @brief it should read only up to the end of the config.
TEST_F(JournalTest, ReadsUpToEnd) {
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    ASSERT_NIL(writer->write({.type = 1, .payload = "a"}));
    ASSERT_NIL(writer->sync());
    const auto synced = writer->synced_pos();
    EXPECT_EQ(synced, writer->pos());
    ASSERT_NIL(writer->write({.type = 1, .payload = "b"}));
    EXPECT_EQ(writer->synced_pos(), synced);
    auto reader = ASSERT_NIL_P(
        Reader::open(
            {.path = this->path, .magic = SEGMENT, .types = {1}, .end = synced}
        )
    );
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 1);
    EXPECT_EQ(records[0].payload, "a");
    EXPECT_EQ(reader->cut(), 0);
    ASSERT_OCCURRED_AS_P(
        Reader::open({.path = this->path, .magic = SEGMENT, .end = 1000}),
        CORRUPT
    );
}

/// @brief it should sync on one thread while another thread writes.
TEST_F(JournalTest, SyncsWhileWriting) {
    auto writer = ASSERT_NIL_P(
        Writer::open(
            {.path = this->path,
             .magic = SEGMENT,
             .sync_interval = x::telem::HOUR,
             .sync_bytes = UINT64_MAX}
        )
    );
    std::atomic<bool> done = false;
    std::thread syncer([&] {
        while (!done)
            EXPECT_NIL(writer->sync());
    });
    for (int i = 0; i < 1000; i++)
        ASSERT_NIL(writer->write({.type = 1, .payload = "a"}));
    done = true;
    syncer.join();
    ASSERT_NIL(writer->sync());
    EXPECT_EQ(writer->synced_pos(), writer->pos());
    EXPECT_EQ(read_all(*this->open_reader()).size(), 1000);
}

/// @brief it should read records larger than its read buffer.
TEST_F(JournalTest, ReadsLargeRecords) {
    const auto large = std::string(std::size_t{200} * 1024, 'l');
    this->write(
        {{.type = 1, .payload = "a"},
         {.type = 1, .payload = large},
         {.type = 1, .payload = "b"}}
    );
    const auto reader = this->open_reader();
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 3);
    EXPECT_EQ(records[1].payload, large);
    EXPECT_EQ(records[2].payload, "b");
}

/// @brief it should end the log at a record whose length passes the end of the file.
TEST_F(JournalTest, LengthPastEndEndsLog) {
    this->write({{.type = 1, .payload = "abc"}, {.type = 1, .payload = "def"}});
    auto data = read_file(this->path);
    data[HEADER_SIZE + 3] = 0x01;
    write_file(this->path, data);
    const auto reader = this->open_reader();
    EXPECT_EQ(read_all(*reader).size(), 0);
    EXPECT_EQ(reader->cut(), data.size() - HEADER_SIZE);
}

/// @brief it should return CORRUPT for a valid record with type 0.
TEST_F(JournalTest, TypeZeroIsCorrupt) {
    this->write({});
    append_file(this->path, encode(salt_of(this->path), 0, "a"));
    const auto reader = this->open_reader();
    ASSERT_OCCURRED_AS_P(reader->next(), CORRUPT);
    ASSERT_OCCURRED_AS_P(Writer::open({.path = this->path, .magic = SEGMENT}), CORRUPT);
}

/// @brief it should return CORRUPT for a valid record with only the SKIPPABLE bit.
TEST_F(JournalTest, SkippableTypeZeroIsCorrupt) {
    this->write({});
    append_file(this->path, encode(salt_of(this->path), SKIPPABLE, "a"));
    const auto reader = this->open_reader();
    ASSERT_OCCURRED_AS_P(reader->next(), CORRUPT);
    ASSERT_OCCURRED_AS_P(Writer::open({.path = this->path, .magic = SEGMENT}), CORRUPT);
}

/// @brief it should skip an unknown type that has the SKIPPABLE bit.
TEST_F(JournalTest, SkipsUnknownSkippableType) {
    this->write(
        {{.type = 2 | SKIPPABLE, .payload = "new"}, {.type = 1, .payload = "old"}}
    );
    const auto reader = this->open_reader();
    const auto records = read_all(*reader);
    ASSERT_EQ(records.size(), 1);
    EXPECT_EQ(records[0].payload, "old");
}

/// @brief it should return UNKNOWN_TYPE for an unknown type without the SKIPPABLE bit.
TEST_F(JournalTest, UnknownTypeIsError) {
    this->write({{.type = 2, .payload = "new"}, {.type = 1, .payload = "old"}});
    const auto reader = this->open_reader();
    ASSERT_OCCURRED_AS_P(reader->next(), UNKNOWN_TYPE);
    EXPECT_EQ(reader->pos(), HEADER_SIZE);
    ASSERT_OCCURRED_AS_P(reader->next(), UNKNOWN_TYPE);
}

/// @brief it should refuse a record with type 0 and keep writing.
TEST_F(JournalTest, WriterRejectsTypeZero) {
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    ASSERT_OCCURRED_AS(writer->write({.type = SKIPPABLE}), BASE_ERROR);
    ASSERT_NIL(writer->write({.type = 1}));
}

/// @brief it should refuse a payload over MAX_PAYLOAD and keep writing.
TEST_F(JournalTest, WriterRejectsTooLargePayload) {
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    ASSERT_OCCURRED_AS(
        writer->write({.type = 1, .payload = std::string(MAX_PAYLOAD + 1, 'a')}),
        BASE_ERROR
    );
    ASSERT_NIL(writer->write({.type = 1}));
}

/// @brief it should refuse a write after close.
TEST_F(JournalTest, WriteAfterCloseIsError) {
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    ASSERT_NIL(writer->close());
    ASSERT_OCCURRED_AS(writer->write({.type = 1}), BASE_ERROR);
}

/// @brief it should refuse a second writer on a file until the first closes.
TEST_F(JournalTest, SecondWriterIsLocked) {
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    ASSERT_OCCURRED_AS_P(Writer::open({.path = this->path, .magic = SEGMENT}), LOCKED);
    ASSERT_NIL(writer->close());
    writer.reset();
    ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
}

/// @brief it should return UNKNOWN_MAGIC for the magic of another kind of file.
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

/// @brief it should return UNKNOWN_VERSION for a newer format version.
TEST_F(JournalTest, NewerVersionIsUnknownVersion) {
    this->write({});
    set_header(this->path, 2, 0);
    ASSERT_OCCURRED_AS_P(
        Reader::open({.path = this->path, .magic = SEGMENT}),
        UNKNOWN_VERSION
    );
    ASSERT_OCCURRED_AS_P(
        Writer::open({.path = this->path, .magic = SEGMENT}),
        UNKNOWN_VERSION
    );
}

/// @brief it should return CORRUPT for a format version of 0.
TEST_F(JournalTest, VersionZeroIsCorrupt) {
    this->write({});
    set_header(this->path, 0, 0);
    ASSERT_OCCURRED_AS_P(Reader::open({.path = this->path, .magic = SEGMENT}), CORRUPT);
}

/// @brief it should refuse a magic of all zeros.
TEST_F(JournalTest, RefusesZeroMagic) {
    ASSERT_OCCURRED_AS_P(Writer::open({.path = this->path}), BASE_ERROR);
    EXPECT_FALSE(std::filesystem::exists(this->path));
    write_file(this->path, std::string(HEADER_SIZE, '\0'));
    ASSERT_OCCURRED_AS_P(Reader::open({.path = this->path}), BASE_ERROR);
}

/// @brief it should create a file whole and remove the temp files of a crash.
TEST_F(JournalTest, CreatesFileWhole) {
    write_file(this->dir / "0.seg.tmp.7", "SYTE");
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    EXPECT_EQ(writer->synced_pos(), HEADER_SIZE);
    EXPECT_FALSE(std::filesystem::exists(this->dir / "0.seg.tmp.7"));
    EXPECT_EQ(std::filesystem::file_size(this->path), HEADER_SIZE);
}

/// @brief it should let other handles replace and remove a file while it reads.
TEST_F(JournalTest, ReaderDoesNotBlockRemove) {
    this->write({{.type = 1, .payload = "a"}});
    const auto reader = this->open_reader();
    std::filesystem::remove(this->path);
    EXPECT_FALSE(std::filesystem::exists(this->path));
    EXPECT_EQ(ASSERT_NIL_P(reader->next()).value().payload, "a");
}

/// @brief it should write a new file start over a file shorter than one.
TEST_F(JournalTest, WriterReplacesCutOffFileStart) {
    write_file(this->path, "SYTE");
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    EXPECT_EQ(writer->pos(), HEADER_SIZE);
    EXPECT_EQ(writer->cut(), 4);
    ASSERT_NIL(writer->write({.type = 1, .payload = "a"}));
    ASSERT_NIL(writer->close());
    const auto reader = this->open_reader();
    EXPECT_EQ(read_all(*reader).size(), 1);
}

/// @brief it should cut a last record that fails its checksum, with zeros after it.
TEST_F(JournalTest, RecoversBadChecksumAtTail) {
    this->write({{.type = 1, .payload = "a"}});
    const auto size = std::filesystem::file_size(this->path);
    auto bad = encode(salt_of(this->path), 1, "bc");
    bad[4] = static_cast<char>(bad[4] ^ 0xFF);
    append_file(this->path, bad + std::string(100, '\0'));
    auto writer = ASSERT_NIL_P(Writer::open({.path = this->path, .magic = SEGMENT}));
    EXPECT_EQ(writer->cut(), bad.size() + 100);
    EXPECT_EQ(writer->pos(), size);
}

/// @brief it should return an error for a file that does not exist.
TEST_F(JournalTest, ReaderMissingFileIsError) {
    ASSERT_OCCURRED_AS_P(
        Reader::open({.path = this->dir / "missing", .magic = SEGMENT}),
        BASE_ERROR
    );
}

/// @brief it should return an error for a file in a directory that does not exist.
TEST_F(JournalTest, WriterMissingDirectoryIsError) {
    ASSERT_OCCURRED_AS_P(
        Writer::open({.path = this->dir / "missing" / "0.seg", .magic = SEGMENT}),
        BASE_ERROR
    );
}

/// @brief it should read a file start that is cut off as a log with no records.
TEST_F(JournalTest, CutOffFileStartIsEmptyLog) {
    write_file(this->path, "SYTE");
    const auto reader = this->open_reader();
    EXPECT_FALSE(ASSERT_NIL_P(reader->next()).has_value());
    EXPECT_EQ(reader->cut(), 4);
    EXPECT_EQ(reader->pos(), 0);
}

/// @brief it should sync when sync_interval passes or sync_bytes are written.
TEST_F(JournalTest, SyncsByTimeAndBytes) {
    this->write({});
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
    EXPECT_EQ(syncs, 1);
    ASSERT_NIL(writer->write({.type = 1, .payload = "a"}));
    EXPECT_EQ(syncs, 1);
    now = now + x::telem::SECOND;
    ASSERT_NIL(writer->write({.type = 1, .payload = "b"}));
    EXPECT_EQ(syncs, 2);
    ASSERT_NIL(writer->write({.type = 1, .payload = std::string(100, 'c')}));
    EXPECT_EQ(syncs, 3);
    ASSERT_NIL(writer->write({.type = 1, .payload = "d"}));
    ASSERT_NIL(writer->sync_if_due());
    EXPECT_EQ(syncs, 3);
    now = now + x::telem::SECOND;
    ASSERT_NIL(writer->sync_if_due());
    EXPECT_EQ(syncs, 4);
    ASSERT_NIL(writer->sync_if_due());
    EXPECT_EQ(syncs, 4);
    ASSERT_NIL(writer->write({.type = 1, .payload = "e"}));
    ASSERT_NIL(writer->close());
    EXPECT_EQ(syncs, 5);
}

/// @brief it should refuse all later writes and syncs after a sync error.
TEST_F(JournalTest, SyncErrorIsFatal) {
    this->write({});
    const auto injected = x::errors::Error(BASE_ERROR, "disk failed");
    int syncs = 0;
    auto writer = ASSERT_NIL_P(
        Writer::open({
            .path = this->path,
            .magic = SEGMENT,
            .sync_bytes = 1,
            .on_sync = [&](const x::errors::Error &err) {
                return syncs++ == 0 ? err : injected;
            },
        })
    );
    const auto synced = writer->synced_pos();
    ASSERT_OCCURRED_AS(writer->write({.type = 1, .payload = "a"}), BASE_ERROR);
    ASSERT_OCCURRED_AS(writer->write({.type = 1, .payload = "b"}), BASE_ERROR);
    ASSERT_OCCURRED_AS(writer->close(), BASE_ERROR);
    EXPECT_EQ(syncs, 2);
    EXPECT_EQ(writer->synced_pos(), synced);
}
}
