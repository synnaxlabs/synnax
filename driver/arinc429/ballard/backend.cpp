// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/arinc429/ballard/api.h"
#include "driver/arinc429/ballard/backend.h"
#include "driver/arinc429/vendor.h"

namespace driver::arinc429::ballard {
namespace {
x::errors::Error load() {
    for (const auto &name: {CARD_LIBRARY_NAME, LIBRARY_NAME})
        if (auto [lib, err] = vendor::load(name, LIBRARY_INFO); err) return err;
    return vendor::unsupported("Ballard ARINC 429");
}
}

std::pair<std::unique_ptr<Channel>, x::errors::Error>
Backend::open(const synnax::arinc429::Properties &, Direction) {
    return {nullptr, load()};
}

std::pair<std::vector<Info>, x::errors::Error> Backend::list() {
    return {{}, load()};
}
}
