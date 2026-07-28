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

#include "x/cpp/log/log.h"

#include "absl/base/log_severity.h"
#include "absl/log/globals.h"
#include "absl/log/initialize.h"
#include "absl/log/log_entry.h"
#include "absl/log/log_sink.h"
#include "absl/log/log_sink_registry.h"

namespace x::log {
namespace {
std::atomic<bool> color(false);

/// @brief writes log lines to stderr, coloring the whole line by severity:
/// WARNING yellow, ERROR and FATAL red.
class StderrSink final : public absl::LogSink {
public:
    void Send(const absl::LogEntry &entry) override {
        const char *code = nullptr;
        if (color_enabled()) switch (entry.log_severity()) {
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

    void Flush() override { std::fflush(stderr); }
};
}

bool color_enabled() {
    return color.load(std::memory_order_relaxed);
}

void init(const bool enable_color) {
    absl::InitializeLog();
    color.store(enable_color, std::memory_order_relaxed);
    // The default stderr sink would duplicate every line; silence it and let
    // StderrSink own stderr output.
    absl::SetStderrThreshold(absl::LogSeverityAtLeast::kInfinity);
    static StderrSink sink;
    absl::AddLogSink(&sink);
}
}
