// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/bus/vendor.h"
#include "driver/mil1553/ballard/api.h"
#include "driver/mil1553/ballard/backend.h"

namespace driver::mil1553::ballard {
namespace {
x::errors::Error load() {
    for (const auto &name: {CARD_LIBRARY_NAME, LIBRARY_NAME})
        if (auto [lib, err] = bus::vendor::load(name, LIBRARY_INFO); err) return err;
    return bus::vendor::unsupported("Ballard MIL-STD-1553");
}
}

std::pair<std::unique_ptr<Channel>, x::errors::Error>
Backend::open(const synnax::mil1553::Properties &) {
    return {nullptr, load()};
}

std::pair<std::vector<Info>, x::errors::Error> Backend::list() {
    return {{}, load()};
}
}
