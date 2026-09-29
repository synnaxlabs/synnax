// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <utility>
#include <vector>

#include "driver/arinc429/backend.h"

namespace driver::arinc429::ballard {
/// @brief Backend is the Ballard BTIDriver backend. The Driver cannot drive Ballard
/// cards yet: open and list return the missing library error when the BTIDriver
/// libraries are absent, and an unsupported error when they are present.
class Backend final : public arinc429::Backend {
public:
    std::pair<std::unique_ptr<Channel>, x::errors::Error>
    open(const synnax::arinc429::Properties &props, Direction direction) override;

    std::pair<std::vector<Info>, x::errors::Error> list() override;
};
}
