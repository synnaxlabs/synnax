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

#include "driver/mil1553/backend.h"

namespace driver::mil1553::ddc {
/// @brief Backend is the DDC AceXtreme backend. The Driver cannot drive these cards
/// yet: open and list return the missing library error when the vendor library is
/// absent, and an unsupported error when it is present.
class Backend final : public mil1553::Backend {
public:
    std::pair<std::unique_ptr<Channel>, x::errors::Error>
    open(const synnax::mil1553::Properties &props) override;

    std::pair<std::vector<Info>, x::errors::Error> list() override;
};
}
