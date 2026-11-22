// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/soem/filter.h"

namespace driver::ethercat::soem {
x::errors::Error disable_dewesoft_filter(const std::vector<std::string> &) {
    return x::errors::Error(
        errors::FILTER_ERROR,
        "Only a Driver on Windows can turn off the DewesoftX filter."
    );
}
}
