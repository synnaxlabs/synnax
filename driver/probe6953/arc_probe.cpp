#include "benchmark/benchmark.h"

#include "arc/cpp/stl/channels/state.h"

// A WASM read_f64 of a channel calls read_value, which deep-copies every buffered
// series of the channel to return the last sample.
static void BM_ArcReadValue(benchmark::State &st) {
    const size_t n = st.range(0);
    std::vector<arc::stl::channels::Digest> d;
    for (uint32_t c = 1; c <= 10; c++) d.push_back({c, x::telem::FLOAT64_T, 0});
    arc::stl::channels::State s(d);
    x::telem::Frame fr(10);
    for (uint32_t c = 1; c <= 10; c++) {
        x::telem::Series ser(x::telem::FLOAT64_T, n);
        for (size_t i = 0; i < n; i++) ser.write(1.0 * i);
        fr.emplace(c, std::move(ser));
    }
    s.ingest(fr);
    for (auto _: st) {
        for (uint32_t c = 1; c <= 10; c++) {
            auto [ms, ok] = s.read_value(c);
            benchmark::DoNotOptimize(ms.series.back().at<double>(-1));
        }
    }
}
BENCHMARK(BM_ArcReadValue)->Arg(1)->Arg(400);

static void BM_ArcIngestFlush(benchmark::State &st) {
    const size_t n = st.range(0);
    std::vector<arc::stl::channels::Digest> d;
    for (uint32_t c = 1; c <= 10; c++) d.push_back({c, x::telem::FLOAT64_T, 0});
    arc::stl::channels::State s(d);
    for (auto _: st) {
        x::telem::Frame fr(10);
        for (uint32_t c = 1; c <= 10; c++)
            fr.emplace(c, x::telem::Series(x::telem::FLOAT64_T, n));
        s.ingest(fr);
        x::telem::Frame out;
        s.flush_into(out, x::telem::TimeStamp(1));
        benchmark::DoNotOptimize(out.size());
    }
}
BENCHMARK(BM_ArcIngestFlush)->Arg(400);
