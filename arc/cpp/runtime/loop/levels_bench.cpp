// Box-only bench: wake-up lateness of the real Arc loop per performance level.
// usage: levels_bench <auto|low|medium|high> <interval_us> <cycles> [hist_file]
// The config goes through the same parse and apply_defaults path as an Arc task, and
// the thread setup comes from an rt::Manager handle, as the Driver gives it.
#ifdef _WIN32
#include <windows.h>
#endif
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <ctime>
#include <sstream>
#include <thread>
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/json/json.h"

#include "x/cpp/thread/rt/rt.h"

#include "arc/cpp/runtime/loop/loop.h"

using namespace arc::runtime::loop;

static int64_t cpu_ns() {
#ifdef _WIN32
    FILETIME c, e, k, u;
    GetThreadTimes(GetCurrentThread(), &c, &e, &k, &u);
    auto t = [](FILETIME f) {
        return (static_cast<int64_t>(f.dwHighDateTime) << 32 | f.dwLowDateTime) * 100;
    };
    return t(k) + t(u);
#else
    timespec ts;
    clock_gettime(CLOCK_THREAD_CPUTIME_ID, &ts);
    return ts.tv_sec * 1000000000LL + ts.tv_nsec;
#endif
}

int main(int argc, char **argv) {
    if (argc < 4) return 2;
    const auto span = atoll(argv[2]) * x::telem::MICROSECOND;
    const int64_t cycles = atoll(argv[3]);
    x::json::Parser parser(nlohmann::json{{"performance", argv[1]}});
    Config config = Config(parser).apply_defaults(span);
    if (!parser.ok()) return 3;
    const char *hist_path = argc > 4 ? argv[4] : nullptr;
    auto manager = std::make_shared<x::thread::rt::Manager>();
    x::thread::rt::Config base_rt;
    base_rt.enabled = true;
    base_rt.priority = DEFAULT_RT_PRIORITY;
    auto handle = std::make_shared<x::thread::rt::Handle>(manager->allocate(base_rt));
    std::thread thread([&] {
        const auto loop = create(config, handle);
        if (const auto err = loop->start()) {
            printf("start failed: %s\n", err.message().c_str());
            return;
        }
        x::breaker::Breaker breaker;
        breaker.start();
        constexpr int BUCKETS = 100000;
        std::vector<uint64_t> hist(BUCKETS + 1);
        int64_t max = 0, sum = 0;
        const auto sw = x::telem::Stopwatch();
        const int64_t cpu0 = cpu_ns();
        auto next = span;
        for (int64_t i = 0; i < cycles; i++, next += span) {
            auto now = sw.elapsed();
            while (now < next) {
                loop->wait(breaker, next - now, span);
                now = sw.elapsed();
            }
            const int64_t late = (now - next).nanoseconds();
            const int64_t us = late / 1000;
            hist[us < BUCKETS ? us : BUCKETS]++;
            sum += late;
            if (late > max) max = late;
        }
        const double cpu = 100.0 * (cpu_ns() - cpu0) / sw.elapsed().nanoseconds();
        auto pct = [&](double p) {
            const auto want = static_cast<uint64_t>(p * cycles);
            uint64_t seen = 0;
            for (int i = 0; i < BUCKETS; i++)
                if ((seen += hist[i]) >= want) return i;
            return BUCKETS;
        };
        std::ostringstream mode;
        mode << config.mode;
        printf(
            "RESULT level=%s mode=%s interval_us=%lld cycles=%lld avg_us=%.2f "
            "p50_us=%d p99_us=%d p999_us=%d p9999_us=%d max_us=%.1f cpu_pct=%.1f "
            "core=%d\n",
            argv[1],
            mode.str().c_str(),
            (long long) span.microseconds(),
            (long long) cycles,
            sum / 1000.0 / cycles,
            pct(0.5),
            pct(0.99),
            pct(0.999),
            pct(0.9999),
            max / 1000.0,
            cpu,
            handle->allocated_core()
        );
        if (hist_path != nullptr) {
            FILE *f = fopen(hist_path, "w");
            for (int i = 0; i <= BUCKETS; i++)
                if (hist[i] > 0) fprintf(f, "%d %llu\n", i, (unsigned long long) hist[i]);
            fclose(f);
        }
        breaker.stop();
    });
    thread.join();
    return 0;
}
