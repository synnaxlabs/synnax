// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "arc/cpp/runtime/loop/loop_windows.h"

namespace arc::runtime::loop {

std::unique_ptr<Loop>
create(const Config &cfg, std::shared_ptr<x::thread::rt::Handle> rt_handle) {
    return std::make_unique<Windows<>>(cfg, std::move(rt_handle));
}
}
