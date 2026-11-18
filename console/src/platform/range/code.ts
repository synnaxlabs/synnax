// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";

/** Python code that retrieves the range. */
export const pythonCode = (key: ranger.Key): string =>
  `rng = client.ranges.retrieve(key="${key}")`;

/** TypeScript code that retrieves the range. */
export const typescriptCode = (key: ranger.Key): string =>
  `const rng = await client.ranges.retrieve("${key}");`;
