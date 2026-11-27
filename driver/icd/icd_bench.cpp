// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <charconv>
#include <cstddef>
#include <cstdint>
#include <map>
#include <span>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

#include "benchmark/benchmark.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/bench/bench.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/icd/reader.h"
#include "driver/icd/writer.h"

namespace driver::icd { namespace {
constexpr synnax::channel::Key INDEX = 1;
constexpr std::int64_t STAMP = 1767323045123456789;

synnax::channel::Channel
channel(const synnax::channel::Key key, const x::telem::DataType &dt) {
    synnax::channel::Channel ch;
    ch.key = key;
    ch.data_type = dt;
    ch.index = dt == x::telem::TIMESTAMP_T ? key : INDEX;
    ch.is_index = dt == x::telem::TIMESTAMP_T;
    return ch;
}

synnax::icd::FloatJSONField float_field(const std::string &name) {
    synnax::icd::FloatJSONField f;
    f.key = x::uuid::create();
    f.name = name;
    f.pointer = "/" + name;
    return f;
}

/// @brief creates a reader of one JSON message with a timestamp field at /t that
/// stamps n float fields at /f0 to /f(n-1), and returns it with a document.
std::pair<Reader, std::string> reader(const std::size_t n) {
    synnax::icd::Message m{.key = x::uuid::create(), .name = "reading"};
    synnax::icd::JSONPayload payload;
    synnax::icd::TimestampJSONField t;
    t.key = x::uuid::create();
    t.name = "t";
    t.pointer = "/t";
    t.time_format = "unix_ns";
    payload.fields.emplace_back(t);
    synnax::icd::ReadMessage rm{.message = m.key, .index = t.key};
    rm.bindings.push_back({.field = t.key, .channel = INDEX});
    std::map<synnax::channel::Key, synnax::channel::Channel> channels{
        {INDEX, channel(INDEX, x::telem::TIMESTAMP_T)},
    };
    std::string doc = R"({"t":)" + std::to_string(STAMP);
    for (std::size_t i = 0; i < n; i++) {
        const auto f = float_field("f" + std::to_string(i));
        const auto key = static_cast<synnax::channel::Key>(i + 2);
        payload.fields.emplace_back(f);
        rm.bindings.push_back({.field = f.key, .channel = key});
        channels.emplace(key, channel(key, x::telem::FLOAT64_T));
        doc += ",\"" + f.name + "\":1234.5678";
    }
    m.payload = payload;
    synnax::icd::ICD ic{.key = x::uuid::create(), .name = "plant", .messages = {m}};
    auto [r, err] = Reader::create({rm}, ic, channels);
    if (err) throw std::runtime_error(err.message());
    return {std::move(r), doc + "}"};
}

/// @brief returns true when fr holds batch stamps of the document at INDEX and batch
/// samples of 1234.5678 for each of fields data channels.
bool read_correct(
    const x::telem::Frame &fr,
    const std::size_t fields,
    const std::size_t batch
) {
    if (fr.size() != fields + 1) return false;
    for (const auto &[key, series]: fr) {
        if (series.size() != batch) return false;
        for (std::size_t i = 0; i < batch; i++) {
            const auto at = static_cast<int>(i);
            if (key == INDEX ? series.at<std::int64_t>(at) != STAMP
                             : series.at<double>(at) != 1234.5678)
                return false;
        }
    }
    return true;
}

/// @brief reads a batch of documents into a new frame, as a read task does.
void read(benchmark::State &state, const std::size_t fields, const std::size_t batch) {
    auto [r, doc] = reader(fields);
    const std::span payload(
        reinterpret_cast<const std::uint8_t *>(doc.data()),
        doc.size()
    );
    const auto read_batch = [&] {
        x::telem::Frame fr;
        for (std::size_t i = 0; i < batch; i++)
            if (const auto [msg, err] = r.read(
                    payload,
                    x::telem::TimeStamp(1),
                    fr,
                    batch
                );
                err || !msg.has_value())
                state.SkipWithError("read failed");
        return fr;
    };
    if (!read_correct(read_batch(), fields, batch)) state.SkipWithError("wrong frame");
    x::bench::run_with_alloc_tracking(state, [&] {
        benchmark::DoNotOptimize(read_batch());
    });
    state.SetItemsProcessed(static_cast<std::int64_t>(state.iterations() * batch));
}

/// @brief creates a writer of one JSON message with a float field for each of n
/// command channels, and returns it with a frame of samples commands to each.
std::pair<Writer, x::telem::Frame>
writer(const std::size_t n, const std::size_t samples) {
    synnax::icd::Message m{.key = x::uuid::create(), .name = "command"};
    synnax::icd::JSONPayload payload;
    synnax::icd::WriteMessage wm{.message = m.key};
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;
    x::telem::Frame fr;
    for (std::size_t i = 0; i < n; i++) {
        const auto f = float_field("f" + std::to_string(i));
        const auto key = static_cast<synnax::channel::Key>(i + 1);
        payload.fields.emplace_back(f);
        synnax::icd::ChannelWriteBinding b;
        b.field = f.key;
        b.channel = key;
        wm.bindings.emplace_back(b);
        auto ch = channel(key, x::telem::FLOAT64_T);
        ch.index = 0;
        channels.emplace(key, ch);
        std::vector<double> values(samples);
        for (std::size_t j = 0; j < samples; j++)
            values[j] = static_cast<double>(i * samples + j) + 0.5;
        fr.emplace(key, x::telem::Series(values));
    }
    m.payload = payload;
    synnax::icd::ICD ic{.key = x::uuid::create(), .name = "plant", .messages = {m}};
    auto [w, err] = Writer::create(wm, ic, channels);
    if (err) throw std::runtime_error(err.message());
    return {std::move(w), std::move(fr)};
}

/// @brief returns the document of row j of the frame of writer: the value of each of n
/// fields at /f0 to /f(n-1).
std::string
document(const std::size_t n, const std::size_t samples, const std::size_t j) {
    std::string doc = "{";
    for (std::size_t i = 0; i < n; i++) {
        if (i > 0) doc += ',';
        std::array<char, 32> buf{};
        const auto v = static_cast<double>(i * samples + j) + 0.5;
        auto *const end = std::to_chars(buf.data(), buf.data() + buf.size(), v).ptr;
        doc += "\"f" + std::to_string(i) + "\":" + std::string(buf.data(), end);
    }
    return doc + "}";
}

void write(
    benchmark::State &state,
    const std::size_t fields,
    const std::size_t samples
) {
    auto [w, fr] = writer(fields, samples);
    std::vector<std::string> docs;
    const auto check = w.write(fr, [&docs](std::span<const std::uint8_t> doc) {
        docs.emplace_back(doc.begin(), doc.end());
        return x::errors::NIL;
    });
    if (check.error || check.refused || docs.size() != samples)
        state.SkipWithError("write failed");
    for (std::size_t j = 0; j < docs.size(); j++)
        if (docs[j] != document(fields, samples, j))
            state.SkipWithError("wrong document");
    std::size_t sent = 0;
    const Writer::Send send = [&sent](std::span<const std::uint8_t>) {
        sent++;
        return x::errors::NIL;
    };
    x::bench::run_with_alloc_tracking(state, [&] {
        if (const auto res = w.write(fr, send); res.error || res.refused)
            state.SkipWithError("write failed");
    });
    if (sent != state.iterations() * samples) state.SkipWithError("missing documents");
    state.SetItemsProcessed(static_cast<std::int64_t>(state.iterations() * samples));
}

void register_benchmarks() {
    for (const std::size_t fields: {1, 10, 100})
        for (const std::size_t batch: {1, 100})
            benchmark::RegisterBenchmark(
                "Read/fields/" + std::to_string(fields) + "/batch/" +
                    std::to_string(batch),
                read,
                fields,
                batch
            );
    for (const auto &[fields, samples]:
         std::vector<std::pair<std::size_t, std::size_t>>{
             {1, 100},
             {10, 10},
             {100, 10}
         })
        benchmark::RegisterBenchmark(
            "Write/fields/" + std::to_string(fields) + "/samples/" +
                std::to_string(samples),
            write,
            fields,
            samples
        );
}
}}

int main(int argc, char **argv) {
    driver::icd::register_benchmarks();
    benchmark::Initialize(&argc, argv);
    if (benchmark::ReportUnrecognizedArguments(argc, argv)) return 1;
    benchmark::RunSpecifiedBenchmarks();
    benchmark::Shutdown();
    return 0;
}
