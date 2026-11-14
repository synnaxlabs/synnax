// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <mutex>
#include <string>
#include <vector>

#include "absl/base/log_severity.h"
#include "absl/log/log_entry.h"
#include "absl/log/log_sink.h"
#include "absl/log/log_sink_registry.h"

namespace driver::common {
/// @brief captures the text of every line logged at Severity while in scope.
template<absl::LogSeverity Severity>
class SeveritySink final : public absl::LogSink {
    std::mutex mu;
    std::vector<std::string> lines;

public:
    SeveritySink() { absl::AddLogSink(this); }

    ~SeveritySink() override { absl::RemoveLogSink(this); }

    void Send(const absl::LogEntry &entry) override {
        if (entry.log_severity() != Severity) return;
        std::lock_guard lock(this->mu);
        this->lines.emplace_back(entry.text_message());
    }

    /// @brief returns the number of captured lines that contain text.
    size_t count(const std::string &text) {
        std::lock_guard lock(this->mu);
        return std::ranges::count_if(this->lines, [&](const std::string &line) {
            return line.contains(text);
        });
    }
};

using WarningSink = SeveritySink<absl::LogSeverity::kWarning>;
using ErrorSink = SeveritySink<absl::LogSeverity::kError>;
}
