// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@/errors";

const TYPE = "sy.patch.stale";

/**
 * Thrown when an update names a list entry that is missing, or when a guard of an
 * update does not hold in the stored record.
 */
export class StaleError extends errors.createTyped(TYPE) {}

errors.register({
  encode: (err) => (StaleError.matches(err) ? { type: TYPE, data: err.message } : null),
  decode: (payload) => (payload.type === TYPE ? new StaleError(payload.data) : null),
});
