// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/opc/errors/errors.h"
#include "driver/opc/telem/telem.h"
#include "driver/opc/util/util.h"

namespace util {
std::pair<size_t, xerrors::Error>
write_to_series(telem::Series &s, const UA_Variant &v) {
    return opc::telem::write_to_series(s, v);
}

xerrors::Error parse_error(const UA_StatusCode &status) {
    return opc::errors::parse(status);
}
}
