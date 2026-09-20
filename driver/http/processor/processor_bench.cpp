// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <string>

#include "benchmark/benchmark.h"

#include "x/cpp/telem/telem.h"

#include "driver/http/errors/errors.h"
#include "driver/http/mock/server.h"
#include "driver/http/processor/processor.h"

namespace driver::http {
namespace {
mock::Route slow_route(const x::telem::TimeSpan &delay) {
    return {
        .method = Method::GET,
        .path = "/slow",
        .status_code = 200,
        .response_body = "ok",
        .content_type = "text/plain",
        .delay = delay,
    };
}

Request make_request(const std::string &url, const x::telem::TimeSpan &timeout) {
    return Request{.url = url, .method = Method::GET, .timeout = timeout};
}

x::telem::TimeSpan poll_timeout(const benchmark::State &state) {
    return state.range(0) * x::telem::MILLISECOND;
}
}

/// Real time is the server delay. Process CPU time is mostly the io thread polling
/// while the one request waits.
static void BM_InFlightWait(benchmark::State &state) {
    const auto delay = 1 * x::telem::SECOND;
    mock::ServerConfig cfg;
    cfg.routes = {slow_route(delay)};
    mock::Server server(cfg);
    if (const auto err = server.start()) {
        state.SkipWithError(err.message());
        return;
    }
    Processor proc(poll_timeout(state));
    const auto req = make_request(server.base_url() + "/slow", 5 * x::telem::SECOND);
    for (auto _: state) {
        auto [resp, err] = proc.execute(req);
        if (err) {
            state.SkipWithError(err.message());
            break;
        }
    }
}

BENCHMARK(BM_InFlightWait)
    ->Arg(1)
    ->Arg(10)
    ->Arg(100)
    ->Iterations(3)
    ->MeasureProcessCPUTime()
    ->UseRealTime()
    ->Unit(benchmark::kMillisecond);

/// Overshoot is how late a request timeout fires when the server stays silent. It
/// stays near zero when libcurl's own timers cap the poll wait.
static void BM_TimeoutOvershoot(benchmark::State &state) {
    const auto timeout = 50 * x::telem::MILLISECOND;
    mock::ServerConfig cfg;
    cfg.routes = {slow_route(200 * x::telem::MILLISECOND)};
    mock::Server server(cfg);
    if (const auto err = server.start()) {
        state.SkipWithError(err.message());
        return;
    }
    Processor proc(poll_timeout(state));
    const auto req = make_request(server.base_url() + "/slow", timeout);
    double overshoot_ms = 0;
    for (auto _: state) {
        const auto start = std::chrono::steady_clock::now();
        auto [resp, err] = proc.execute(req);
        const auto elapsed = std::chrono::duration<double, std::milli>(
            std::chrono::steady_clock::now() - start
        );
        if (!err.matches(errors::UNREACHABLE_ERROR)) {
            state.SkipWithError("expected a timeout");
            break;
        }
        overshoot_ms += elapsed.count() - timeout.milliseconds();
    }
    state.counters["overshoot_ms"] = benchmark::Counter(
        overshoot_ms,
        benchmark::Counter::kAvgIterations
    );
}

BENCHMARK(BM_TimeoutOvershoot)
    ->Arg(1)
    ->Arg(10)
    ->Arg(100)
    ->Iterations(20)
    ->UseRealTime()
    ->Unit(benchmark::kMillisecond);
}
