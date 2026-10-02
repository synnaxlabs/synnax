// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "x/cpp/loop/sleeper.h"

namespace x::loop {
struct Sleeper::Impl {
    static void sleep(const telem::TimeSpan &dur) {
        std::this_thread::sleep_for(dur.chrono());
    }
};

Sleeper::Sleeper(): impl_(std::make_unique<Impl>()) {}

Sleeper::~Sleeper() = default;

Sleeper::Sleeper(Sleeper &&other) noexcept = default;

Sleeper &Sleeper::operator=(Sleeper &&other) noexcept = default;

void Sleeper::sleep(const telem::TimeSpan &dur) {
    this->impl_->sleep(dur);
}
}
