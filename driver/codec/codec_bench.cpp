// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <functional>
#include <numeric>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "benchmark/benchmark.h"

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/bench/bench.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/binary.h"
#include "driver/codec/json.h"
#include "driver/codec/text.h"

namespace driver::codec { namespace {
namespace icd = synnax::icd;

/// @brief Kind is one field kind of a benchmark: the field at a name, the data type of
/// its channel, the JSON text of a value, and the sample that the text decodes to or
/// encodes from.
struct Kind {
    std::string name;
    std::function<icd::JSONField(const std::string &)> field;
    x::telem::DataType data_type;
    std::string text;
    std::function<x::telem::Series()> sample;
};

/// @brief returns a field of type F named name at /name.
template<typename F>
F base(const std::string &name) {
    F f;
    f.key = x::uuid::create();
    f.name = name;
    f.pointer = "/" + name;
    return f;
}

template<typename F>
std::function<icd::JSONField(const std::string &)> plain() {
    return [](const std::string &name) { return base<F>(name); };
}

std::function<icd::JSONField(const std::string &)> scaled() {
    return [](const std::string &name) {
        auto f = base<icd::FloatJSONField>(name);
        f.scale = 0.5;
        f.offset = 10;
        return f;
    };
}

std::function<icd::JSONField(const std::string &)> bounded() {
    return [](const std::string &name) {
        auto f = base<icd::FloatJSONField>(name);
        f.min = -1e6;
        f.max = 1e6;
        return f;
    };
}

std::function<icd::JSONField(const std::string &)> labeled() {
    return [](const std::string &name) {
        auto f = base<icd::IntegerJSONField>(name);
        f.enum_values = {{.label = "OFF", .value = 0}, {.label = "ON", .value = 1}};
        return f;
    };
}

std::function<icd::JSONField(const std::string &)>
timestamp(const std::string &format) {
    return [format](const std::string &name) {
        auto f = base<icd::TimestampJSONField>(name);
        f.time_format = format;
        return f;
    };
}

template<typename T>
std::function<x::telem::Series()> sample_of(T v, const x::telem::DataType &dt) {
    return [v, dt] { return x::telem::Series(v, dt); };
}

std::function<x::telem::Series()> uuid_sample() {
    return [] {
        const auto id = x::uuid::UUID::parse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
                            .first;
        x::telem::Series s(x::telem::UUID_T, 1);
        s.write(id.data(), 1);
        return s;
    };
}

std::function<x::telem::Series()> bytes_sample() {
    return [] {
        std::string b(16, '\0');
        std::iota(b.begin(), b.end(), 0);
        return x::telem::Series(b, x::telem::BYTES_T);
    };
}

const auto FLOAT_SAMPLE = sample_of(1234.5678, x::telem::FLOAT64_T);
const std::function<x::telem::Series()> STAMP_SAMPLE = [] {
    return x::telem::Series(x::telem::TimeStamp(1767323045123456789));
};
const auto STRING_SAMPLE = sample_of(std::string("pump running"), x::telem::STRING_T);
const auto JSON_SAMPLE = sample_of(
    std::string(R"({"a":1,"b":[true,null]})"),
    x::telem::JSON_T
);
constexpr auto UUID_TEXT = R"("6ba7b810-9dad-11d1-80b4-00c04fd430c8")";
constexpr auto BYTES_TEXT = R"("AAECAwQFBgcICQoLDA0ODw==")";
constexpr auto ISO8601_TEXT = R"("2026-01-02T03:04:05.123456789Z")";

std::vector<Kind> decode_kinds() {
    return {
        {"float",
         plain<icd::FloatJSONField>(),
         x::telem::FLOAT64_T,
         "1234.5678",
         FLOAT_SAMPLE},
        {"float_scaled",
         scaled(),
         x::telem::FLOAT64_T,
         "1234.5678",
         sample_of(1234.5678 * 0.5 + 10, x::telem::FLOAT64_T)},
        {"float_string",
         plain<icd::FloatJSONField>(),
         x::telem::FLOAT64_T,
         R"("1234.5678")",
         FLOAT_SAMPLE},
        {"integer",
         plain<icd::IntegerJSONField>(),
         x::telem::INT64_T,
         "-123456789",
         sample_of(std::int64_t{-123456789}, x::telem::INT64_T)},
        {"integer_label",
         labeled(),
         x::telem::INT64_T,
         R"("ON")",
         sample_of(std::int64_t{1}, x::telem::INT64_T)},
        {"unsigned",
         plain<icd::UnsignedJSONField>(),
         x::telem::UINT32_T,
         "123456",
         sample_of(std::uint32_t{123456}, x::telem::UINT32_T)},
        {"boolean",
         plain<icd::BooleanJSONField>(),
         x::telem::BOOLEAN_T,
         "true",
         sample_of(std::uint8_t{1}, x::telem::BOOLEAN_T)},
        {"string",
         plain<icd::StringJSONField>(),
         x::telem::STRING_T,
         R"("pump running")",
         STRING_SAMPLE},
        {"timestamp_iso8601",
         timestamp("iso8601"),
         x::telem::TIMESTAMP_T,
         ISO8601_TEXT,
         STAMP_SAMPLE},
        {"timestamp_unix_ns",
         timestamp("unix_ns"),
         x::telem::TIMESTAMP_T,
         "1767323045123456789",
         STAMP_SAMPLE},
        {"json",
         plain<icd::JSONJSONField>(),
         x::telem::JSON_T,
         R"({"a":1,"b":[true,null]})",
         JSON_SAMPLE},
        {"uuid",
         plain<icd::UuidJSONField>(),
         x::telem::UUID_T,
         UUID_TEXT,
         uuid_sample()},
        {"bytes",
         plain<icd::BytesJSONField>(),
         x::telem::BYTES_T,
         BYTES_TEXT,
         bytes_sample()},
    };
}

std::vector<Kind> encode_kinds() {
    return {
        {"float",
         plain<icd::FloatJSONField>(),
         x::telem::FLOAT64_T,
         "1234.5678",
         FLOAT_SAMPLE},
        {"float_scaled", scaled(), x::telem::FLOAT64_T, "2449.1356", FLOAT_SAMPLE},
        {"float_bounded", bounded(), x::telem::FLOAT64_T, "1234.5678", FLOAT_SAMPLE},
        {"integer",
         plain<icd::IntegerJSONField>(),
         x::telem::INT64_T,
         "-123456789",
         sample_of(std::int64_t{-123456789}, x::telem::INT64_T)},
        {"integer_from_float",
         plain<icd::IntegerJSONField>(),
         x::telem::FLOAT64_T,
         "1235",
         FLOAT_SAMPLE},
        {"integer_label",
         labeled(),
         x::telem::INT64_T,
         R"("ON")",
         sample_of(std::int64_t{1}, x::telem::INT64_T)},
        {"unsigned",
         plain<icd::UnsignedJSONField>(),
         x::telem::UINT32_T,
         "123456",
         sample_of(std::uint32_t{123456}, x::telem::UINT32_T)},
        {"boolean",
         plain<icd::BooleanJSONField>(),
         x::telem::BOOLEAN_T,
         "true",
         sample_of(std::uint8_t{1}, x::telem::BOOLEAN_T)},
        {"string",
         plain<icd::StringJSONField>(),
         x::telem::STRING_T,
         R"("pump running")",
         STRING_SAMPLE},
        {"timestamp_iso8601",
         timestamp("iso8601"),
         x::telem::TIMESTAMP_T,
         ISO8601_TEXT,
         STAMP_SAMPLE},
        {"timestamp_unix_ns",
         timestamp("unix_ns"),
         x::telem::TIMESTAMP_T,
         "1767323045123456789",
         STAMP_SAMPLE},
        {"json",
         plain<icd::JSONJSONField>(),
         x::telem::JSON_T,
         R"({"a":1,"b":[true,null]})",
         JSON_SAMPLE},
        {"uuid",
         plain<icd::UuidJSONField>(),
         x::telem::UUID_T,
         UUID_TEXT,
         uuid_sample()},
        {"bytes",
         plain<icd::BytesJSONField>(),
         x::telem::BYTES_T,
         BYTES_TEXT,
         bytes_sample()},
    };
}

/// @brief returns true when each series of got holds the bytes of want.
bool each_equal(
    const std::vector<x::telem::Series> &got,
    const x::telem::Series &want
) {
    for (const auto &s: got)
        if (s.byte_size() != want.byte_size() ||
            std::memcmp(s.data(), want.data(), want.byte_size()) != 0)
            return false;
    return true;
}

/// @brief returns true when payload holds the bytes of want.
template<typename T>
bool payload_equal(const std::vector<std::uint8_t> &payload, const T &want) {
    return std::equal(payload.begin(), payload.end(), want.begin(), want.end());
}

const std::vector<std::size_t> SIZES = {1, 10, 100, 1000};

/// @brief KIND_FIELDS is the number of fields of each kind benchmark.
constexpr std::size_t KIND_FIELDS = 10;

/// @brief returns a message of n fields of kind k at /f0 to /f(n-1), each bound to a
/// channel of the data type of k, and a document that holds a value for each.
std::pair<JSONMessage, std::string> json_message(const Kind &k, const std::size_t n) {
    JSONMessage m{.name = k.name};
    std::string doc = "{";
    for (std::size_t i = 0; i < n; i++) {
        const auto name = "f" + std::to_string(i);
        auto f = k.field(name);
        m.bindings.push_back({.field = icd::base(f).key, .data_type = k.data_type});
        m.payload.fields.push_back(std::move(f));
        if (i > 0) doc += ",";
        doc += "\"" + name + "\":" + k.text;
    }
    return {std::move(m), doc + "}"};
}

void json_decode(benchmark::State &state, const Kind &k, const std::size_t n) {
    const auto [m, doc] = json_message(k, n);
    auto [dec, err] = JSON::create({m});
    if (err) {
        state.SkipWithError(err.message());
        return;
    }
    const auto decode = [&] {
        const auto [message, err] = dec.decode(doc);
        if (err || !message.has_value() || !dec.failures(0).empty())
            state.SkipWithError("decode failed");
    };
    decode();
    if (!each_equal(dec.samples(0), k.sample())) state.SkipWithError("wrong samples");
    x::bench::run_with_alloc_tracking(state, [&] {
        decode();
        benchmark::DoNotOptimize(dec.samples(0).data());
    });
    state.SetBytesProcessed(static_cast<std::int64_t>(state.iterations() * doc.size()));
}

void json_encode(benchmark::State &state, const Kind &k, const std::size_t n) {
    const auto [m, doc] = json_message(k, n);
    auto [enc, err] = JSONEncoder::create(m);
    if (err) {
        state.SkipWithError(err.message());
        return;
    }
    std::vector<x::telem::Series> samples;
    for (std::size_t i = 0; i < n; i++)
        samples.push_back(k.sample());
    std::vector<std::uint8_t> payload;
    if (enc.encode(samples, payload) || !payload_equal(payload, doc))
        state.SkipWithError("wrong document");
    x::bench::run_with_alloc_tracking(state, [&] {
        if (enc.encode(samples, payload)) state.SkipWithError("encode failed");
        benchmark::DoNotOptimize(payload);
    });
    state.SetBytesProcessed(
        static_cast<std::int64_t>(state.iterations() * payload.size())
    );
}

/// @brief returns a message of n little-endian 16-bit integer fields, each scaled by
/// 0.1 and bound to a float64 channel, and a frame of it.
std::pair<BinaryMessage, std::vector<std::uint8_t>>
binary_message(const std::size_t n) {
    BinaryMessage m{.name = "frame"};
    for (std::size_t i = 0; i < n; i++) {
        icd::IntegerBinaryField f;
        f.key = x::uuid::create();
        f.name = "f" + std::to_string(i);
        f.start_bit = static_cast<std::uint16_t>(i * 16);
        f.bit_length = 16;
        f.scale = 0.1;
        m.bindings.push_back({.field = f.key, .data_type = x::telem::FLOAT64_T});
        m.payload.fields.emplace_back(std::move(f));
    }
    std::vector<std::uint8_t> frame(n * 2);
    for (std::size_t i = 0; i < frame.size(); i += 2) {
        frame[i] = 0x10;
        frame[i + 1] = 0x27;
    }
    return {std::move(m), std::move(frame)};
}

void binary_decode(benchmark::State &state, const std::size_t n) {
    const auto [m, frame] = binary_message(n);
    auto [dec, err] = Binary::create({m});
    if (err) {
        state.SkipWithError(err.message());
        return;
    }
    const auto decode = [&] {
        const auto [message, err] = dec.decode(frame);
        if (err || !message.has_value()) state.SkipWithError("decode failed");
    };
    decode();
    if (!each_equal(dec.samples(0), x::telem::Series(1000.0, x::telem::FLOAT64_T)))
        state.SkipWithError("wrong samples");
    x::bench::run_with_alloc_tracking(state, [&] {
        decode();
        benchmark::DoNotOptimize(dec.samples(0).data());
    });
}

void binary_encode(benchmark::State &state, const std::size_t n) {
    const auto [m, frame] = binary_message(n);
    auto [enc, err] = BinaryEncoder::create(m);
    if (err) {
        state.SkipWithError(err.message());
        return;
    }
    std::vector<x::telem::Series> samples;
    for (std::size_t i = 0; i < n; i++)
        samples.emplace_back(1000.0, x::telem::FLOAT64_T);
    std::vector<std::uint8_t> payload;
    if (enc.encode(samples, payload) || payload != frame)
        state.SkipWithError("wrong frame");
    x::bench::run_with_alloc_tracking(state, [&] {
        if (enc.encode(samples, payload)) state.SkipWithError("encode failed");
        benchmark::DoNotOptimize(payload);
    });
}

/// @brief returns a message of n delimited float fields bound to float64 channels, and
/// a line of it.
std::pair<TextMessage, std::string> text_message(const std::size_t n) {
    TextMessage m{.name = "line"};
    std::string line;
    for (std::size_t i = 0; i < n; i++) {
        icd::DelimitedTextField f;
        f.key = x::uuid::create();
        f.name = "f" + std::to_string(i);
        f.position = static_cast<std::uint32_t>(i);
        m.bindings.push_back({.field = f.key, .data_type = x::telem::FLOAT64_T});
        m.payload.fields.emplace_back(std::move(f));
        if (i > 0) line += ",";
        line += "1234.5678";
    }
    return {std::move(m), std::move(line)};
}

void text_decode(benchmark::State &state, const std::size_t n) {
    const auto [m, line] = text_message(n);
    auto [dec, err] = Text::create({m});
    if (err) {
        state.SkipWithError(err.message());
        return;
    }
    const auto decode = [&] {
        const auto [message, err] = dec.decode(line);
        if (err || !message.has_value()) state.SkipWithError("decode failed");
    };
    decode();
    if (!each_equal(dec.samples(0), FLOAT_SAMPLE()))
        state.SkipWithError("wrong samples");
    x::bench::run_with_alloc_tracking(state, [&] {
        decode();
        benchmark::DoNotOptimize(dec.samples(0).data());
    });
    state.SetBytesProcessed(
        static_cast<std::int64_t>(state.iterations() * line.size())
    );
}

void text_encode(benchmark::State &state, const std::size_t n) {
    const auto [m, line] = text_message(n);
    auto [enc, err] = TextEncoder::create(m);
    if (err) {
        state.SkipWithError(err.message());
        return;
    }
    std::vector<x::telem::Series> samples;
    for (std::size_t i = 0; i < n; i++)
        samples.emplace_back(1234.5678, x::telem::FLOAT64_T);
    std::vector<std::uint8_t> payload;
    if (enc.encode(samples, payload) || !payload_equal(payload, line))
        state.SkipWithError("wrong line");
    x::bench::run_with_alloc_tracking(state, [&] {
        if (enc.encode(samples, payload)) state.SkipWithError("encode failed");
        benchmark::DoNotOptimize(payload);
    });
}

void register_benchmarks() {
    for (const auto &k: decode_kinds())
        benchmark::RegisterBenchmark(
            "JSONDecode/kind/" + k.name,
            json_decode,
            k,
            KIND_FIELDS
        );
    for (const auto n: SIZES)
        benchmark::RegisterBenchmark(
            "JSONDecode/fields/" + std::to_string(n),
            json_decode,
            decode_kinds()[0],
            n
        );
    for (const auto &k: encode_kinds())
        benchmark::RegisterBenchmark(
            "JSONEncode/kind/" + k.name,
            json_encode,
            k,
            KIND_FIELDS
        );
    for (const auto n: SIZES)
        benchmark::RegisterBenchmark(
            "JSONEncode/fields/" + std::to_string(n),
            json_encode,
            encode_kinds()[0],
            n
        );
    for (const std::size_t n: {1, 4, 32}) {
        const auto fields = std::to_string(n);
        benchmark::RegisterBenchmark("BinaryDecode/fields/" + fields, binary_decode, n);
        benchmark::RegisterBenchmark("BinaryEncode/fields/" + fields, binary_encode, n);
    }
    for (const std::size_t n: {1, 10, 100}) {
        const auto fields = std::to_string(n);
        benchmark::RegisterBenchmark("TextDecode/fields/" + fields, text_decode, n);
        benchmark::RegisterBenchmark("TextEncode/fields/" + fields, text_encode, n);
    }
}
}}

int main(int argc, char **argv) {
    driver::codec::register_benchmarks();
    benchmark::Initialize(&argc, argv);
    if (benchmark::ReportUnrecognizedArguments(argc, argv)) return 1;
    benchmark::RunSpecifiedBenchmarks();
    benchmark::Shutdown();
    return 0;
}
