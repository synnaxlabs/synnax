// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "driver/http/errors/errors.h"

namespace driver::http::errors {
x::errors::Error classify_status(const int status_code) {
    if (status_code >= 200 && status_code < 300) return x::errors::NIL;
    if (status_code == 408 || status_code == 429 ||
        (status_code >= 500 && status_code < 600))
        return TEMPORARY_ERROR.sub(std::to_string(status_code));
    return CRITICAL_ERROR.sub(std::to_string(status_code));
}
}
