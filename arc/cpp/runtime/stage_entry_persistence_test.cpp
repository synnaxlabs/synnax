// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/testutil/compile.h"
#include "arc/cpp/runtime/testutil/harness.h"
#include "arc/cpp/types/types.h"

/// A persistence check armed on stage entry acts only on samples taken inside the
/// stage (SY-5456) and measures its hold in sample time (SY-5457). Mirrors
/// arc/go/stage_entry_persistence_test.go.
///
/// Timeline, with T the stamp at which the run stage enters:
///
///   T-34 .. T-14 ms  samples below 300 are taken, during the previous stage
///   T                run stage enters, check armed, step=700 written
///   T+2              that batch is delivered and dropped
///   T+22             a batch taken inside the stage is delivered
///
/// A func with a flow-fed input runs once per sample with the sample's stamp. An
/// inline expression such as `sensor < 300.0` compiles to a function with no inputs
/// that reads the channel's latest value, runs once per batch, and stamps the cycle
/// clock, so stable.for behind it measures arrival to arrival.
namespace arc::runtime {
namespace {
const x::telem::TimeStamp BASE(10 * x::telem::SECOND.nanoseconds());
const x::telem::TimeSpan STARTUP_WAIT = 250 * x::telem::MILLISECOND;

/// @brief wraps one run-stage check in the sequence shape under test.
std::string program(const std::string &check) {
    std::string imports = "import time\n";
    if (check.contains("stable.")) imports += "import stable\n";
    return imports + R"(
func low_for_50() {v f64} bool {
    count i64 $= 0
    if v < 300.0 {
        count = count + 1
    } else {
        count = 0
    }
    return count >= 50
}
func is_low() {v f64} bool {
    return v < 300.0
}
sequence main {
    stage startup {
        600 -> %step%
        time.wait(250ms) => next
    }
    stage run {
        700 -> %step%
        %sensor% -> %sensor_seen%
        )" +
           check +
           R"(
    }
    stage shutdown {
        900 -> %step%
    }
}
1 => main)";
}

/// @brief compiles the program for one check and runs it under a fake wall clock.
class Sequence {
    synnax::Synnax client = new_test_client();
    testutil::Channels channels;
    x::telem::TimeStamp now = BASE;
    testutil::Harness harness;

public:
    explicit Sequence(const std::string &check):
        channels(
            this->client,
            {{"step_idx", x::telem::TIMESTAMP_T, "", true},
             {"step", x::telem::INT64_T, "step_idx"},
             {"sensor_idx", x::telem::TIMESTAMP_T, "", true},
             {"sensor", x::telem::FLOAT64_T, "sensor_idx"},
             {"sensor_seen", x::telem::FLOAT64_T}}
        ),
        harness(this->client, this->channels.substitute(program(check)), [this] {
            return this->now;
        }) {}

    [[nodiscard]] types::ChannelKey key(const std::string &name) const {
        return this->channels.key(name);
    }

    /// @brief runs one scheduler cycle at wall clock at and returns the writes.
    x::telem::Frame cycle(const x::telem::TimeStamp at) {
        this->now = at;
        this->harness.tick(at - BASE);
        return this->harness.flush();
    }

    void ingest_sensor(
        const std::vector<x::telem::TimeStamp> &stamps,
        const std::vector<double> &values
    ) {
        this->harness.ingest_indexed(
            this->key("sensor_idx"),
            x::telem::Series(stamps),
            this->key("sensor"),
            x::telem::Series(values, x::telem::FLOAT64_T)
        );
    }

    /// @brief ingests 100 samples at 5 kHz, every one below 300, starting at from.
    void ingest_low_batch(const x::telem::TimeStamp from) {
        std::vector<x::telem::TimeStamp> stamps;
        std::vector<double> values;
        for (int i = 0; i < 100; i++) {
            stamps.push_back(
                from + static_cast<std::int64_t>(200 * i) * x::telem::MICROSECOND
            );
            values.push_back(255 + i * 0.35);
        }
        this->ingest_sensor(stamps, values);
    }

    /// @brief ingests the batch taken 34 to 14 ms before entry: the batch the stage
    /// sees first, one 16 ms delivery latency late.
    void ingest_stale_batch(const x::telem::TimeStamp entry) {
        this->ingest_low_batch(entry - 34 * x::telem::MILLISECOND);
    }

    /// @brief every (value, stamp) pair written to step in the frame.
    [[nodiscard]] std::pair<std::vector<std::int64_t>, std::vector<x::telem::TimeStamp>>
    step_writes(const x::telem::Frame &fr) const {
        std::vector<std::int64_t> vals;
        std::vector<x::telem::TimeStamp> stamps;
        for (const auto &[ch, series]: fr) {
            if (ch == this->key("step"))
                for (const auto v: series.values<std::int64_t>())
                    vals.push_back(v);
            if (ch == this->key("step_idx"))
                for (const auto v: series.values<std::int64_t>())
                    stamps.emplace_back(v);
        }
        return {vals, stamps};
    }

    [[nodiscard]] size_t seen_count(const x::telem::Frame &fr) const {
        size_t n = 0;
        for (const auto &[ch, series]: fr)
            if (ch == this->key("sensor_seen")) n += series.size();
        return n;
    }

    /// @brief drives the sequence through startup into run and returns the entry
    /// stamp T, the stamp on the step=700 write.
    x::telem::TimeStamp enter_run() {
        auto [vals, stamps] = this->step_writes(this->cycle(BASE));
        EXPECT_EQ(vals, std::vector<std::int64_t>{600});
        auto out = this->cycle(BASE + STARTUP_WAIT);
        auto [run_vals, run_stamps] = this->step_writes(out);
        EXPECT_EQ(run_vals, std::vector<std::int64_t>{700})
            << "run must enter after the wait";
        if (run_stamps.empty()) return x::telem::TimeStamp(0);
        return run_stamps[0];
    }

    void expect_abort_at(
        const x::telem::Frame &out,
        const x::telem::TimeStamp entry,
        const x::telem::TimeStamp at
    ) const {
        auto [vals, stamps] = this->step_writes(out);
        ASSERT_EQ(vals, std::vector<std::int64_t>{900}) << "shutdown must have entered";
        ASSERT_EQ(stamps.size(), 1);
        EXPECT_EQ((stamps[0] - entry).nanoseconds(), (at - entry).nanoseconds());
    }

    void expect_no_abort(const x::telem::Frame &out) const {
        auto [vals, stamps] = this->step_writes(out);
        EXPECT_TRUE(vals.empty()) << "no abort expected";
    }
};

const std::string COUNTER_CHECK = "%sensor% -> low_for_50() => shutdown";
const std::string STABLE_PER_SAMPLE_CHECK =
    "%sensor% -> is_low() -> stable.for(duration=10ms) => shutdown";
const std::string
    STABLE_INLINE_CHECK = "%sensor% < 300.0 -> stable.for(duration=10ms) => shutdown";
}

// A sample counter fed per sample.
TEST(StageEntryPersistenceTest, CounterIgnoresTheStaleBatchAndTripsOnInStageSamples) {
    Sequence h(COUNTER_CHECK);
    const auto entry = h.enter_run();
    h.ingest_stale_batch(entry);
    auto out = h.cycle(entry + 2 * x::telem::MILLISECOND);
    EXPECT_EQ(h.seen_count(out), 0) << "pre-entry samples are dropped";
    h.expect_no_abort(out);

    h.ingest_low_batch(entry);
    out = h.cycle(entry + 22 * x::telem::MILLISECOND);
    EXPECT_EQ(h.seen_count(out), 100);
    h.expect_abort_at(out, entry, entry + 22 * x::telem::MILLISECOND);
}

// stable.for fed per sample. 20 ms of low samples taken inside the stage prove a
// 10 ms hold.
TEST(StageEntryPersistenceTest, StableForIgnoresTheStaleBatchAndTripsOnInStageSamples) {
    Sequence h(STABLE_PER_SAMPLE_CHECK);
    const auto entry = h.enter_run();
    h.ingest_stale_batch(entry);
    auto out = h.cycle(entry + 2 * x::telem::MILLISECOND);
    EXPECT_EQ(h.seen_count(out), 0) << "pre-entry samples are dropped";
    h.expect_no_abort(out);

    h.ingest_low_batch(entry);
    out = h.cycle(entry + 22 * x::telem::MILLISECOND);
    EXPECT_EQ(h.seen_count(out), 100);
    h.expect_abort_at(out, entry, entry + 22 * x::telem::MILLISECOND);
}

// The data shows 2 ms below 300. The 16 ms on the wall clock since the first sample
// does not count.
TEST(
    StageEntryPersistenceTest,
    StableForDoesNotTripOn2msOfLowSamplesDelivered16msLate
) {
    Sequence h(STABLE_PER_SAMPLE_CHECK);
    const auto entry = h.enter_run();
    h.ingest_sensor(
        {entry, entry + 1 * x::telem::MILLISECOND, entry + 2 * x::telem::MILLISECOND},
        {290, 292, 294}
    );
    const auto out = h.cycle(entry + 18 * x::telem::MILLISECOND);
    EXPECT_EQ(h.seen_count(out), 3);
    h.expect_no_abort(out);
}

TEST(StageEntryPersistenceTest, StableForDoesNotTripOnABatchAboveTheThreshold) {
    Sequence h(STABLE_PER_SAMPLE_CHECK);
    const auto entry = h.enter_run();
    h.ingest_sensor(
        {entry,
         entry + 2 * x::telem::MILLISECOND,
         entry + 4 * x::telem::MILLISECOND,
         entry + 6 * x::telem::MILLISECOND},
        {313, 316, 320, 323}
    );
    const auto out = h.cycle(entry + 22 * x::telem::MILLISECOND);
    EXPECT_EQ(h.seen_count(out), 4);
    h.expect_no_abort(out);
}

TEST(StageEntryPersistenceTest, InlineExpressionSeesOneCycleStampedValuePerBatch) {
    Sequence h(STABLE_INLINE_CHECK);
    const auto entry = h.enter_run();
    h.ingest_sensor({entry, entry + 1 * x::telem::MILLISECOND}, {290, 292});
    auto out = h.cycle(entry + 2 * x::telem::MILLISECOND);
    EXPECT_EQ(h.seen_count(out), 2);
    h.expect_no_abort(out);

    h.ingest_sensor(
        {entry + 8 * x::telem::MILLISECOND, entry + 10 * x::telem::MILLISECOND},
        {294, 296}
    );
    out = h.cycle(entry + 12 * x::telem::MILLISECOND);
    h.expect_abort_at(out, entry, entry + 12 * x::telem::MILLISECOND);
}
}
