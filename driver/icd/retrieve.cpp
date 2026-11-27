// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>
#include <set>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"

#include "driver/icd/retrieve.h"

namespace driver::icd {
std::pair<Resources, x::errors::Error> retrieve(
    const synnax::Synnax &client,
    const std::set<synnax::icd::Key> &icds,
    const std::set<synnax::channel::Key> &channels
) {
    Resources r;
    auto [found_icds, icd_err] = client.icds.retrieve(
        std::vector(icds.begin(), icds.end())
    );
    if (icd_err) return {Resources{}, icd_err};
    r.icds = std::move(found_icds);
    if (channels.empty()) return {std::move(r), x::errors::NIL};
    // A Where filter skips missing keys, where a retrieve by keys fails on them.
    auto [found, ch_err] = client.channels.retrieve(
        synnax::channel::Where{.key = std::vector(channels.begin(), channels.end())}
    );
    if (ch_err) return {Resources{}, ch_err};
    for (auto &ch: found)
        r.channels.emplace(ch.key, std::move(ch));
    return {std::move(r), x::errors::NIL};
}
}
