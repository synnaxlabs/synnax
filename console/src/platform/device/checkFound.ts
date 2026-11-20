// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { strings } from "@synnaxlabs/x";

/**
 * Checks that a retrieve by key found a device for every key. A key filter skips a
 * missing key instead of failing.
 * @throws {NotFoundError} if a key has no device in devices.
 */
export const checkFound = (keys: string[], devices: Array<{ key: string }>): void => {
  const found = new Set(devices.map((d) => d.key));
  const missing = keys.filter((k) => !found.has(k));
  if (missing.length > 0)
    throw new NotFoundError(
      `Found no device with key ${strings.naturalLanguageJoin(missing)}`,
    );
};
