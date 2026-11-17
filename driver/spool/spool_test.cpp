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
#include <functional>
#include <iterator>
#include <memory>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/journal/journal.h"
#include "driver/spool/spool.h"
#include "driver/spool/spool.pb.h"

namespace driver::spool {
namespace {
const std::filesystem::path TESTDATA = "driver/spool/testdata";

std::string read_file(const std::filesystem::path &path) {
    std::ifstream f(path, std::ios::binary);
    return {std::istreambuf_iterator<char>(f), std::istreambuf_iterator<char>()};
}

void append_file(const std::filesystem::path &path, const std::string &data) {
    std::ofstream f(path, std::ios::binary | std::ios::app);
    f.write(data.data(), static_cast<std::streamsize>(data.size()));
}

/// @brief appends a record that passes its checksum to the segment at path.
void append_record(const std::filesystem::path &path, const journal::Record &record) {
    auto writer = ASSERT_NIL_P(
        journal::Writer::open({.path = path, .magic = SEGMENT_MAGIC})
    );
    ASSERT_NIL(writer->write(record));
    ASSERT_NIL(writer->close());
}

x::uuid::UUID uuid_of(const std::uint8_t first) {
    std::array<std::uint8_t, 16> bytes{};
    for (std::size_t i = 0; i < bytes.size(); i++)
        bytes[i] = static_cast<std::uint8_t>(first + i);
    return x::uuid::UUID(bytes);
}

/// @brief returns a valid header for channels 1 and 2, after edit changes it.
std::string header_payload(const std::function<void(pb::Header &)> &edit) {
    pb::Header pb;
    pb.set_cluster(std::string(16, '\x01'));
    pb.set_task(std::string(16, '\x02'));
    for (const std::uint32_t ch: {1, 2}) {
        pb.add_channels(ch);
        pb.add_indexes(0);
        pb.add_data_types("float32");
    }
    edit(pb);
    return pb.SerializeAsString();
}

x::telem::Frame frame_of(const float value) {
    return {1, x::telem::Series(std::vector<float>{value})};
}

float value_of(const x::telem::Frame &frame) {
    return frame.series->at(0).at<float>(0);
}

Header header_of(const std::uint32_t channel) {
    return Header{
        .cluster = x::uuid::create(),
        .channels = {channel, channel + 1},
        .indexes = {0, 0},
        .data_types = {x::telem::FLOAT32_T, x::telem::FLOAT32_T},
        .subject = {.key = "writer", .name = "Writer", .group = 3},
        .task = x::uuid::create(),
        .rack = 5,
        .created = x::telem::TimeStamp(42),
        .version = "0.62.0",
    };
}

/// @brief the header of the v1 fixture.
Header v1_header() {
    return Header{
        .cluster = uuid_of(0x01),
        .channels = {1, 2, 3},
        .indexes = {1, 1, 1},
        .data_types = {x::telem::TIMESTAMP_T, x::telem::FLOAT32_T, x::telem::STRING_T},
        .subject = {.key = "writer", .name = "Writer", .group = 3},
        .task = uuid_of(0x11),
        .rack = 65537,
        .created = x::telem::TimeStamp(1700000000000000000),
        .version = "0.62.0",
    };
}

/// @brief the frame of the v1 fixture.
x::telem::Frame v1_frame() {
    x::telem::Frame frame(3);
    frame.emplace(
        1,
        x::telem::Series(std::vector<std::int64_t>{10, 20}, x::telem::TIMESTAMP_T)
    );
    auto values = x::telem::Series(std::vector<float>{1.5, 2.5});
    values.alignment = x::telem::Alignment(2, 5);
    values.time_range = x::telem::TimeRange(10, 21);
    frame.emplace(2, std::move(values));
    frame.emplace(3, x::telem::Series(std::vector<std::string>{"a", "bc"}));
    return frame;
}

void expect_header_eq(const Header &actual, const Header &expected) {
    EXPECT_EQ(actual.cluster, expected.cluster);
    EXPECT_EQ(actual.channels, expected.channels);
    EXPECT_EQ(actual.indexes, expected.indexes);
    EXPECT_EQ(actual.data_types, expected.data_types);
    EXPECT_EQ(actual.subject.key, expected.subject.key);
    EXPECT_EQ(actual.subject.name, expected.subject.name);
    EXPECT_EQ(actual.subject.group, expected.subject.group);
    EXPECT_EQ(actual.task, expected.task);
    EXPECT_EQ(actual.rack, expected.rack);
    EXPECT_EQ(actual.created, expected.created);
    EXPECT_EQ(actual.version, expected.version);
}

void expect_v1_frame(const x::telem::Frame &frame) {
    ASSERT_EQ(*frame.channels, (std::vector<std::uint32_t>{1, 2, 3}));
    const auto &times = frame.series->at(0);
    EXPECT_EQ(times.data_type(), x::telem::TIMESTAMP_T);
    EXPECT_EQ(times.values<std::int64_t>(), (std::vector<std::int64_t>{10, 20}));
    const auto &values = frame.series->at(1);
    EXPECT_EQ(values.data_type(), x::telem::FLOAT32_T);
    EXPECT_EQ(values.values<float>(), (std::vector<float>{1.5, 2.5}));
    EXPECT_EQ(values.alignment, x::telem::Alignment(2, 5));
    EXPECT_EQ(values.time_range.start, x::telem::TimeStamp(10));
    EXPECT_EQ(values.time_range.end, x::telem::TimeStamp(21));
    const auto &strings = frame.series->at(2);
    EXPECT_EQ(strings.data_type(), x::telem::STRING_T);
    EXPECT_EQ(strings.strings(), (std::vector<std::string>{"a", "bc"}));
}
}

class SpoolTest : public ::testing::Test {
protected:
    std::filesystem::path dir;

    void SetUp() override {
        this->dir = std::filesystem::temp_directory_path() /
                    ("spool_test_" + x::uuid::create().to_string());
    }

    void TearDown() override { std::filesystem::remove_all(this->dir); }

    [[nodiscard]] std::unique_ptr<Spool>
    open_spool(const std::uint64_t cap = DEFAULT_CAP) const {
        return ASSERT_NIL_P(Spool::open(Config{.dir = this->dir, .cap = cap}));
    }

    void write_frames(Spool &spool, const Header &header, const int count) const {
        const auto writer = spool.open_writer(header);
        for (int i = 0; i < count; i++)
            ASSERT_NIL(writer->write(frame_of(static_cast<float>(i))));
    }

    /// @brief writes a segment with one frame, and returns its path.
    [[nodiscard]] std::filesystem::path write_segment() const {
        const auto spool = this->open_spool();
        this->write_frames(*spool, header_of(1), 1);
        return this->dir / "0.seg";
    }
};

/// @brief it should read a v1 segment, so every later Driver can read it.
TEST_F(SpoolTest, ReadsV1Segment) {
    std::filesystem::create_directories(this->dir);
    std::filesystem::copy_file(TESTDATA / "v1_segment.bin", this->dir / "0.seg");
    const auto spool = this->open_spool();
    ASSERT_EQ(spool->segments(), std::vector<std::uint64_t>{0});
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    expect_header_eq(reader->header, v1_header());
    const auto frame = ASSERT_NIL_P(reader->read());
    expect_v1_frame(frame.value());
    ASSERT_FALSE(ASSERT_NIL_P(reader->read()).has_value());
}

/// @brief it should write the same bytes as the v1 segment.
TEST_F(SpoolTest, WritesV1Segment) {
    {
        const auto spool = this->open_spool();
        ASSERT_NIL(spool->open_writer(v1_header())->write(v1_frame()));
    }
    EXPECT_EQ(read_file(this->dir / "0.seg"), read_file(TESTDATA / "v1_segment.bin"));
}

/// @brief it should read back every header field and the frames after a reopen.
TEST_F(SpoolTest, ReadsBackAfterReopen) {
    const auto header = header_of(1);
    {
        const auto spool = this->open_spool();
        this->write_frames(*spool, header, 3);
    }
    const auto spool = this->open_spool();
    const auto segments = spool->segments();
    ASSERT_EQ(segments.size(), 1);
    const auto reader = ASSERT_NIL_P(spool->open_reader(segments[0]));
    expect_header_eq(reader->header, header);
    for (int i = 0; i < 3; i++) {
        const auto frame = ASSERT_NIL_P(reader->read());
        ASSERT_EQ(value_of(frame.value()), static_cast<float>(i));
    }
    ASSERT_FALSE(ASSERT_NIL_P(reader->read()).has_value());
}

/// @brief it should rebuild each series from the data types of the header.
TEST_F(SpoolTest, RoundTripsFrame) {
    {
        const auto spool = this->open_spool();
        ASSERT_NIL(spool->open_writer(v1_header())->write(v1_frame()));
    }
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    const auto frame = ASSERT_NIL_P(reader->read());
    expect_v1_frame(frame.value());
}

/// @brief it should not write the data type of each series to the segment.
TEST_F(SpoolTest, OmitsDataTypesFromFrames) {
    const auto spool = this->open_spool();
    this->write_frames(*spool, header_of(1), 3);
    const auto segment = read_file(this->dir / "0.seg");
    std::size_t count = 0;
    for (auto i = segment.find("float32"); i != std::string::npos;
         i = segment.find("float32", i + 1))
        count++;
    ASSERT_EQ(count, 2);
}

/// @brief it should reject a frame with a channel that is not in the header.
TEST_F(SpoolTest, RejectsFrameWithUnknownChannel) {
    const auto spool = this->open_spool();
    const auto writer = spool->open_writer(header_of(1));
    ASSERT_MATCHES(
        writer->write({7, x::telem::Series(std::vector<float>{1})}),
        BASE_ERROR
    );
    ASSERT_NIL(writer->write(frame_of(1)));
}

/// @brief it should reject a series with another data type than the header.
TEST_F(SpoolTest, RejectsFrameWithWrongDataType) {
    const auto spool = this->open_spool();
    const auto writer = spool->open_writer(header_of(1));
    ASSERT_MATCHES(
        writer->write({1, x::telem::Series(std::vector<double>{1})}),
        BASE_ERROR
    );
}

/// @brief it should reject a header without a data type for each channel.
TEST_F(SpoolTest, RejectsInvalidHeader) {
    const auto spool = this->open_spool();
    auto header = header_of(1);
    header.data_types.pop_back();
    ASSERT_MATCHES(spool->open_writer(header)->write(frame_of(1)), BASE_ERROR);
    ASSERT_TRUE(spool->segments().empty());
}

/// @brief it should list a segment after close, and refuse later frames.
TEST_F(SpoolTest, ClosesWriter) {
    const auto spool = this->open_spool();
    const auto writer = spool->open_writer(header_of(1));
    ASSERT_NIL(writer->write(frame_of(1)));
    ASSERT_NIL(writer->close());
    ASSERT_EQ(spool->segments().size(), 1);
    ASSERT_MATCHES(writer->write(frame_of(2)), BASE_ERROR);
    ASSERT_NIL(writer->close());
}

/// @brief it should reject a header with a data type it does not know.
TEST_F(SpoolTest, RejectsUnknownDataType) {
    const auto spool = this->open_spool();
    auto header = header_of(1);
    header.data_types[0] = x::telem::DataType("complex128");
    ASSERT_MATCHES(spool->open_writer(header)->write(frame_of(1)), BASE_ERROR);
}

/// @brief it should fail to remove a segment that is not closed.
TEST_F(SpoolTest, RejectsRemoveOfOpenSegment) {
    const auto spool = this->open_spool();
    const auto writer = spool->open_writer(header_of(1));
    ASSERT_NIL(writer->write(frame_of(1)));
    ASSERT_MATCHES(spool->remove(0), BASE_ERROR);
}

/// @brief it should list a segment only after its writer closes.
TEST_F(SpoolTest, ListsClosedSegmentsOnly) {
    const auto spool = this->open_spool();
    auto writer = spool->open_writer(header_of(1));
    ASSERT_NIL(writer->write(frame_of(1)));
    ASSERT_NIL(writer->sync_if_due());
    ASSERT_TRUE(spool->segments().empty());
    writer.reset();
    ASSERT_EQ(spool->segments().size(), 1);
}

/// @brief it should list segments in the order their writers opened.
TEST_F(SpoolTest, ListsSegmentsInOrder) {
    {
        const auto spool = this->open_spool();
        auto first = spool->open_writer(header_of(1));
        auto second = spool->open_writer(header_of(2));
        auto third = spool->open_writer(header_of(3));
        ASSERT_NIL(third->write({3, x::telem::Series(std::vector<float>{3})}));
        ASSERT_NIL(first->write({1, x::telem::Series(std::vector<float>{1})}));
        ASSERT_NIL(second->write({2, x::telem::Series(std::vector<float>{2})}));
    }
    const auto spool = this->open_spool();
    const auto segments = spool->segments();
    ASSERT_EQ(segments.size(), 3);
    for (std::uint32_t i = 0; i < segments.size(); i++) {
        const auto reader = ASSERT_NIL_P(spool->open_reader(segments[i]));
        ASSERT_EQ(reader->header.channels[0], i + 1);
    }
}

/// @brief it should refuse and count every frame after the cap is reached.
TEST_F(SpoolTest, RefusesAndCountsFramesPastCap) {
    const auto spool = this->open_spool(300);
    const auto writer = spool->open_writer(header_of(1));
    ASSERT_NIL(writer->write(frame_of(1)));
    x::errors::Error err;
    while (!err)
        err = writer->write(frame_of(2));
    ASSERT_MATCHES(err, FULL);
    err = writer->write(frame_of(3));
    ASSERT_MATCHES(err, FULL);
    ASSERT_EQ(spool->refused(), 2);
}

/// @brief it should count the bytes of every file it writes against the cap.
TEST_F(SpoolTest, CountsSegmentBytesExactly) {
    std::uint64_t size = 0;
    {
        const auto spool = this->open_spool();
        this->write_frames(*spool, header_of(1), 3);
        size = std::filesystem::file_size(this->dir / "0.seg");
        ASSERT_NIL(spool->remove(0));
    }
    const auto capped = this->open_spool(size);
    this->write_frames(*capped, header_of(1), 3);
    ASSERT_MATCHES(capped->open_writer(header_of(1))->write(frame_of(1)), FULL);
}

/// @brief it should accept frames again after a removed segment frees space.
TEST_F(SpoolTest, AcceptsFramesAfterRemove) {
    const auto spool = this->open_spool(300);
    {
        const auto writer = spool->open_writer(header_of(1));
        while (!writer->write(frame_of(1))) {}
    }
    ASSERT_NIL(spool->remove(spool->segments()[0]));
    ASSERT_TRUE(spool->segments().empty());
    const auto writer = spool->open_writer(header_of(1));
    ASSERT_NIL(writer->write(frame_of(1)));
}

/// @brief it should remove a segment whose first write fails, and fail later writes.
TEST_F(SpoolTest, RemovesSegmentOfFailedWrite) {
    const auto spool = this->open_spool(1000);
    const auto writer = spool->open_writer(header_of(1));
    std::filesystem::create_directory(this->dir / "0.seg");
    auto err = writer->write(frame_of(1));
    ASSERT_MATCHES(err, journal::BASE_ERROR);
    const auto again = writer->write(frame_of(2));
    ASSERT_EQ(again.data, err.data);
    ASSERT_FALSE(std::filesystem::exists(this->dir / "0.seg"));
    ASSERT_NIL(spool->open_writer(header_of(1))->write(frame_of(1)));
    ASSERT_EQ(spool->refused(), 0);
}

/// @brief it should resume a reader from the committed offset after a reopen.
TEST_F(SpoolTest, ResumesFromCommittedOffset) {
    {
        const auto spool = this->open_spool();
        this->write_frames(*spool, header_of(1), 3);
        const auto reader = ASSERT_NIL_P(spool->open_reader(spool->segments()[0]));
        ASSERT_NIL_P(reader->read());
        ASSERT_NIL(reader->commit());
        ASSERT_NIL_P(reader->read());
    }
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(spool->segments()[0]));
    const auto frame = ASSERT_NIL_P(reader->read());
    ASSERT_EQ(value_of(frame.value()), 1);
}

/// @brief it should fail to open a reader with the read offset of another segment.
TEST_F(SpoolTest, RejectsOffsetOfOtherSegment) {
    const auto path = this->write_segment();
    ASSERT_NIL(
        journal::commit_offset(
            this->dir / "0.offset",
            OFFSET_MAGIC,
            {.segment = 1, .pos = std::filesystem::file_size(path)}
        )
    );
    const auto spool = this->open_spool();
    ASSERT_OCCURRED_AS_P(spool->open_reader(0), CORRUPT);
}

/// @brief it should fail to open a reader with a read offset inside the header.
TEST_F(SpoolTest, RejectsOffsetInHeader) {
    static_cast<void>(this->write_segment());
    ASSERT_NIL(
        journal::commit_offset(
            this->dir / "0.offset",
            OFFSET_MAGIC,
            {.segment = 0, .pos = journal::HEADER_SIZE}
        )
    );
    const auto spool = this->open_spool();
    ASSERT_OCCURRED_AS_P(spool->open_reader(0), CORRUPT);
}

/// @brief it should count the cut bytes of a segment once across readers.
TEST_F(SpoolTest, CountsCutOnce) {
    const auto path = this->write_segment();
    append_file(path, std::string("\x40\x00\x00", 3));
    const auto spool = this->open_spool();
    for (int i = 0; i < 2; i++) {
        const auto reader = ASSERT_NIL_P(spool->open_reader(0));
        while (ASSERT_NIL_P(reader->read()).has_value()) {}
    }
    ASSERT_EQ(spool->cut(), 3);
}

/// @brief it should delete a removed segment from disk.
TEST_F(SpoolTest, RemovesSegment) {
    {
        const auto spool = this->open_spool();
        this->write_frames(*spool, header_of(1), 1);
        const auto reader = ASSERT_NIL_P(spool->open_reader(spool->segments()[0]));
        ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
        ASSERT_FALSE(ASSERT_NIL_P(reader->read()).has_value());
        ASSERT_NIL(reader->commit());
        ASSERT_NIL(spool->remove(spool->segments()[0]));
    }
    ASSERT_FALSE(std::filesystem::exists(this->dir / "0.seg"));
    ASSERT_FALSE(std::filesystem::exists(this->dir / "0.offset"));
    ASSERT_TRUE(this->open_spool()->segments().empty());
}

/// @brief it should hold the directory, so a second spool cannot open it.
TEST_F(SpoolTest, LocksDirectory) {
    const auto spool = this->open_spool();
    ASSERT_OCCURRED_AS_P(
        Spool::open(Config{.dir = this->dir, .cap = DEFAULT_CAP}),
        journal::LOCKED
    );
}

/// @brief it should end the segment at a record that a crash cut off.
TEST_F(SpoolTest, EndsAtCutOffRecord) {
    const auto path = this->write_segment();
    append_file(path, std::string("\x40\x00\x00", 3));
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_FALSE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_EQ(spool->cut(), 3);
}

/// @brief it should end the segment at a zero-filled tail that a power loss can
/// leave.
TEST_F(SpoolTest, EndsAtZeroFilledTail) {
    const auto path = this->write_segment();
    append_file(path, std::string(16, '\0'));
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_FALSE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_EQ(spool->cut(), 16);
}

/// @brief it should fail on a record that passes its checksum but is not a frame.
TEST_F(SpoolTest, FailsOnUndecodableFrame) {
    const auto path = this->write_segment();
    append_record(path, {.type = 2, .payload = "\xff\xff"});
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_OCCURRED_AS_P(reader->read(), CORRUPT);
}

/// @brief it should fail on a frame whose series is not whole samples.
TEST_F(SpoolTest, FailsOnPartialSample) {
    const auto path = this->write_segment();
    // A frame of channel 1 with three data bytes: a float32 needs four.
    const std::string payload("\x0a\x01\x01\x12\x05\x0a\x03\x00\x00\x00", 10);
    append_record(path, {.type = 2, .payload = payload});
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_OCCURRED_AS_P(reader->read(), CORRUPT);
}

/// @brief it should fail on a variable-length series with a cut-off sample.
TEST_F(SpoolTest, FailsOnPartialVariableSample) {
    {
        const auto spool = this->open_spool();
        ASSERT_NIL(spool->open_writer(v1_header())->write(v1_frame()));
    }
    // A frame of string channel 3 whose sample says 5 bytes but holds 1.
    const std::string payload(
        "\x0a\x01\x03\x12\x07\x0a\x05\x05\x00\x00\x00"
        "a",
        12
    );
    append_record(this->dir / "0.seg", {.type = 2, .payload = payload});
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_OCCURRED_AS_P(reader->read(), CORRUPT);
}

/// @brief it should fail on a second header in a segment.
TEST_F(SpoolTest, FailsOnSecondHeader) {
    const auto path = this->write_segment();
    append_record(path, {.type = 1, .payload = ""});
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_OCCURRED_AS_P(reader->read(), CORRUPT);
}

/// @brief it should fail on a record type it does not know and cannot skip.
TEST_F(SpoolTest, FailsOnUnknownType) {
    const auto path = this->write_segment();
    append_record(path, {.type = 5, .payload = "new"});
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_OCCURRED_AS_P(reader->read(), journal::UNKNOWN_TYPE);
}

/// @brief it should skip a record type it does not know that has the SKIPPABLE bit.
TEST_F(SpoolTest, SkipsUnknownSkippableType) {
    const auto path = this->write_segment();
    append_record(path, {.type = journal::SKIPPABLE | 5, .payload = "new"});
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_FALSE(ASSERT_NIL_P(reader->read()).has_value());
}

/// @brief it should fail to open on a header that passes its checksum but is not
/// valid.
TEST_F(SpoolTest, FailsOnUndecodableHeader) {
    std::filesystem::create_directories(this->dir);
    append_record(this->dir / "0.seg", {.type = 1, .payload = "\xff\xff"});
    ASSERT_OCCURRED_AS_P(
        Spool::open(Config{.dir = this->dir, .cap = DEFAULT_CAP}),
        CORRUPT
    );
}

/// @brief it should fail to open on a header with a cluster that is not a UUID.
TEST_F(SpoolTest, FailsOnHeaderWithShortCluster) {
    std::filesystem::create_directories(this->dir);
    append_record(
        this->dir / "0.seg",
        {.type = 1,
         .payload = header_payload([](pb::Header &h) { h.set_cluster("short"); })}
    );
    ASSERT_OCCURRED_AS_P(Spool::open(Config{.dir = this->dir}), CORRUPT);
}

/// @brief it should fail to open on a header with a task that is not a UUID.
TEST_F(SpoolTest, FailsOnHeaderWithShortTask) {
    std::filesystem::create_directories(this->dir);
    append_record(
        this->dir / "0.seg",
        {.type = 1, .payload = header_payload([](pb::Header &h) { h.clear_task(); })}
    );
    ASSERT_OCCURRED_AS_P(Spool::open(Config{.dir = this->dir}), CORRUPT);
}

/// @brief it should fail to open on a header without a data type for each channel.
TEST_F(SpoolTest, FailsOnHeaderWithMissingDataType) {
    std::filesystem::create_directories(this->dir);
    append_record(
        this->dir / "0.seg",
        {.type = 1, .payload = header_payload([](pb::Header &h) {
                        h.mutable_data_types()->RemoveLast();
                    })}
    );
    ASSERT_OCCURRED_AS_P(Spool::open(Config{.dir = this->dir}), CORRUPT);
}

/// @brief it should fail on a frame with a channel that is not in the header.
TEST_F(SpoolTest, FailsOnFrameWithUnknownChannel) {
    const auto path = this->write_segment();
    pb::Frame frame;
    frame.add_channels(9);
    frame.add_series()->set_data(std::string(4, '\0'));
    append_record(path, {.type = 2, .payload = frame.SerializeAsString()});
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_OCCURRED_AS_P(reader->read(), CORRUPT);
}

/// @brief it should fail on a frame with more channels than series.
TEST_F(SpoolTest, FailsOnFrameWithMissingSeries) {
    const auto path = this->write_segment();
    pb::Frame frame;
    frame.add_channels(1);
    frame.add_channels(2);
    frame.add_series()->set_data(std::string(4, '\0'));
    append_record(path, {.type = 2, .payload = frame.SerializeAsString()});
    const auto spool = this->open_spool();
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    ASSERT_TRUE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_OCCURRED_AS_P(reader->read(), CORRUPT);
}

/// @brief it should fail to open on a segment file whose name is not a key.
TEST_F(SpoolTest, FailsOnInvalidSegmentName) {
    std::filesystem::create_directories(this->dir);
    append_file(this->dir / "abc.seg", "");
    ASSERT_OCCURRED_AS_P(Spool::open(Config{.dir = this->dir}), CORRUPT);
}

/// @brief it should keep a frame whose sync fails, and fail every later write.
TEST_F(SpoolTest, KeepsFrameWhenSyncFails) {
    auto failing = std::make_shared<bool>(false);
    const auto sync_err = x::errors::Error(journal::BASE_ERROR, "disk failed");
    const auto spool = ASSERT_NIL_P(
        Spool::open(
            Config{
                .dir = this->dir,
                .sync_interval = x::telem::TimeSpan(0),
                .on_sync = [failing, sync_err](const x::errors::Error &err) {
                    return *failing ? sync_err : err;
                },
            }
        )
    );
    {
        const auto writer = spool->open_writer(header_of(1));
        ASSERT_NIL(writer->write(frame_of(0)));
        *failing = true;
        ASSERT_MATCHES(writer->write(frame_of(1)), journal::BASE_ERROR);
        ASSERT_MATCHES(writer->write(frame_of(2)), journal::BASE_ERROR);
    }
    const auto reader = ASSERT_NIL_P(spool->open_reader(0));
    for (int i = 0; i < 2; i++) {
        const auto frame = ASSERT_NIL_P(reader->read());
        ASSERT_EQ(value_of(frame.value()), static_cast<float>(i));
    }
    ASSERT_FALSE(ASSERT_NIL_P(reader->read()).has_value());
    ASSERT_NIL(spool->remove(0));
}

/// @brief it should remove a segment whose first sync fails before a frame lands.
TEST_F(SpoolTest, RemovesSegmentWhenFirstSyncFails) {
    const auto sync_err = x::errors::Error(journal::BASE_ERROR, "disk failed");
    const auto spool = ASSERT_NIL_P(
        Spool::open(
            Config{
                .dir = this->dir,
                .sync_interval = x::telem::TimeSpan(0),
                .on_sync = [sync_err](const x::errors::Error &) { return sync_err; },
            }
        )
    );
    ASSERT_MATCHES(
        spool->open_writer(header_of(1))->write(frame_of(0)),
        journal::BASE_ERROR
    );
    ASSERT_TRUE(spool->segments().empty());
    ASSERT_FALSE(std::filesystem::exists(this->dir / "0.seg"));
}

/// @brief it should fail to open on a segment that starts with a frame.
TEST_F(SpoolTest, FailsOnSegmentWithoutHeader) {
    std::filesystem::create_directories(this->dir);
    append_record(this->dir / "0.seg", {.type = 2, .payload = ""});
    ASSERT_OCCURRED_AS_P(
        Spool::open(Config{.dir = this->dir, .cap = DEFAULT_CAP}),
        CORRUPT
    );
}

/// @brief it should remove a segment that a crash left with no records.
TEST_F(SpoolTest, RemovesEmptySegment) {
    std::filesystem::create_directories(this->dir);
    {
        auto writer = ASSERT_NIL_P(
            journal::Writer::open({.path = this->dir / "4.seg", .magic = SEGMENT_MAGIC})
        );
        ASSERT_NIL(writer->close());
    }
    ASSERT_TRUE(this->open_spool()->segments().empty());
    ASSERT_FALSE(std::filesystem::exists(this->dir / "4.seg"));
}

/// @brief it should remove a segment whose header a power loss left zero-filled.
TEST_F(SpoolTest, RemovesSegmentWithZeroFilledHeader) {
    std::filesystem::create_directories(this->dir);
    {
        auto writer = ASSERT_NIL_P(
            journal::Writer::open({.path = this->dir / "0.seg", .magic = SEGMENT_MAGIC})
        );
        ASSERT_NIL(writer->close());
    }
    append_file(this->dir / "0.seg", std::string(64, '\0'));
    const auto spool = this->open_spool();
    ASSERT_TRUE(spool->segments().empty());
    ASSERT_EQ(spool->cut(), 64);
    ASSERT_FALSE(std::filesystem::exists(this->dir / "0.seg"));
}

/// @brief it should remove a segment that a crash left with a cut-off file start.
TEST_F(SpoolTest, RemovesSegmentWithCutOffFileStart) {
    std::filesystem::create_directories(this->dir);
    append_file(this->dir / "3.seg", std::string(SEGMENT_MAGIC.data(), 5));
    const auto spool = this->open_spool();
    ASSERT_TRUE(spool->segments().empty());
    ASSERT_EQ(spool->cut(), 5);
    ASSERT_FALSE(std::filesystem::exists(this->dir / "3.seg"));
}

/// @brief it should remove a read offset that has no segment.
TEST_F(SpoolTest, RemovesOrphanOffset) {
    std::filesystem::create_directories(this->dir);
    ASSERT_NIL(
        journal::commit_offset(this->dir / "2.offset", OFFSET_MAGIC, {.segment = 2})
    );
    ASSERT_NIL_P(Spool::open(Config{.dir = this->dir, .cap = DEFAULT_CAP}));
    ASSERT_FALSE(std::filesystem::exists(this->dir / "2.offset"));
}

/// @brief it should remove the temp files that a crash can leave.
TEST_F(SpoolTest, RemovesTempFiles) {
    {
        const auto spool = this->open_spool();
        this->write_frames(*spool, header_of(1), 1);
    }
    for (const auto *name: {"0.offset.tmp.12345", "1.seg.tmp.7"})
        append_file(this->dir / name, "partial");
    const auto spool = this->open_spool();
    ASSERT_FALSE(std::filesystem::exists(this->dir / "0.offset.tmp.12345"));
    ASSERT_FALSE(std::filesystem::exists(this->dir / "1.seg.tmp.7"));
    ASSERT_EQ(spool->segments(), std::vector<std::uint64_t>{0});
}

/// @brief it should fail to open a segment with an unknown format version.
TEST_F(SpoolTest, RejectsUnknownVersion) {
    std::filesystem::create_directories(this->dir);
    {
        std::ofstream file(this->dir / "0.seg", std::ios::binary);
        file.write(SEGMENT_MAGIC.data(), SEGMENT_MAGIC.size());
        file.write("\x02\x00\x00\x00", 4);
    }
    ASSERT_OCCURRED_AS_P(
        Spool::open(Config{.dir = this->dir, .cap = DEFAULT_CAP}),
        journal::UNKNOWN_VERSION
    );
}
}
