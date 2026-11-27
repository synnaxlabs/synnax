#include "benchmark/benchmark.h"

#include "x/cpp/queue/spsc.h"
#include "x/cpp/telem/frame.h"

// Each push signals the notifier: a write syscall per frame, even when the consumer is
// already awake.
static void BM_SPSCPushPop(benchmark::State &st) {
    x::queue::SPSC<x::telem::Frame> q(256);
    for (auto _: st) {
        q.push(x::telem::Frame(1, x::telem::Series(1.0)));
        x::telem::Frame out;
        q.pop(out);
        benchmark::DoNotOptimize(out.size());
    }
}
BENCHMARK(BM_SPSCPushPop);
