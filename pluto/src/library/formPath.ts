// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, deep } from "@synnaxlabs/x";

/**
 * Rewrites a dotted path from the Core, such as "entries.2.bit_length", into the form
 * path of the value it names: camel case parts, and element keys in place of array
 * indexes.
 */
export const toFormPath = (path: string, value: unknown): string =>
  deep.resolvePath(caseconv.snakeToCamel(path.split(".")).join("."), value);
