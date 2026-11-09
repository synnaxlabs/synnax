// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/tz/zoneinfo.h"

namespace x::tz::zoneinfo {
namespace {
#include "x/cpp/tz/zoneinfo.inc"
}

std::string_view data() {
    return {reinterpret_cast<const char *>(DATA), sizeof(DATA)};
}

std::span<const Entry> entries() {
    return ENTRIES;
}

}
