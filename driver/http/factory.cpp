// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/http/http.h"

namespace driver::http {
std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    return {nullptr, false};
}

std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::rack::Rack &rack
) {
    return {};
}
}
