// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <cstdio>
#include <mutex>

#ifdef _WIN32
#include <io.h>
#else
#include <unistd.h>
#endif

#include "x/cpp/log/log.h"

#include "absl/base/log_severity.h"
#include "absl/log/globals.h"
#include "absl/log/initialize.h"
#include "absl/log/log_sink_registry.h"

namespace x::log {
namespace {
// Global by necessity: absl's LOG macros and sink registry are process-wide, leaving no
// construction seam to inject this through. Set once by init.
std::atomic<bool> enabled(false);
}

StderrSink::StderrSink(const bool enable_color): color(enable_color) {}

void StderrSink::Send(const absl::LogEntry &entry) {
    const char *code = nullptr;
    if (this->color) switch (entry.log_severity()) {
            case absl::LogSeverity::kWarning:
                code = "\033[0;33m";
                break;
            case absl::LogSeverity::kError:
            case absl::LogSeverity::kFatal:
                code = "\033[0;31m";
                break;
            default:
                break;
        }
    if (code == nullptr) {
        const auto msg = entry.text_message_with_prefix_and_newline();
        std::fwrite(msg.data(), 1, msg.size(), stderr);
        return;
    }
    const auto msg = entry.text_message_with_prefix();
    std::fprintf(
        stderr,
        "%s%.*s\033[m\n",
        code,
        static_cast<int>(msg.size()),
        msg.data()
    );
}

void StderrSink::Flush() {
    std::fflush(stderr);
}

bool color_enabled() {
    return enabled.load(std::memory_order_relaxed);
}

bool stderr_is_terminal() {
#ifdef _WIN32
    return _isatty(_fileno(stderr)) != 0;
#else
    return isatty(fileno(stderr)) != 0;
#endif
}

void init(const bool enable_color) {
    static std::once_flag once;
    std::call_once(once, [enable_color] {
        absl::InitializeLog();
        enabled.store(enable_color, std::memory_order_relaxed);
        // The default stderr sink would duplicate every line; silence it and let
        // StderrSink own stderr output.
        absl::SetStderrThreshold(absl::LogSeverityAtLeast::kInfinity);
        static StderrSink sink(enable_color);
        absl::AddLogSink(&sink);
    });
}
}
