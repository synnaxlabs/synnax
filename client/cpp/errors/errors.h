// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "freighter/cpp/freighter.h"

namespace synnax {
const freighter::Error BASE_ERROR("sy");
const freighter::Error VALIDATION_ERROR = BASE_ERROR.sub("validation");
const freighter::Error QUERY_ERROR = BASE_ERROR.sub("query");
const freighter::Error MULTIPLE_RESULTS = QUERY_ERROR.sub("multiple_results");
const freighter::Error NOT_FOUND = QUERY_ERROR.sub("not_found");
const freighter::Error AUTH_ERROR = BASE_ERROR.sub("auth");
const freighter::Error INVALID_TOKEN = AUTH_ERROR.sub("invalid-token");
const freighter::Error INVALID_CREDENTIALS = AUTH_ERROR.sub("invalid-credentials");
const freighter::Error INTERNAL_ERROR = BASE_ERROR.sub("internal");
const freighter::Error UNEXPECTED_ERROR = BASE_ERROR.sub("unexpected");
const freighter::Error CONTROL_ERROR = BASE_ERROR.sub("control");
const freighter::Error UNAUTHORIZED_ERROR = CONTROL_ERROR.sub("unauthorized");
}
