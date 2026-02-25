// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

/**
 * For required arrays: coerces null/undefined to empty array [].
 * Use when the array must always be present and iterable.
 *
 * - null → []
 * - undefined → []
 * - [] → []
 * - [items] → [items]
 */
export const nullishToEmpty = <Z extends z.ZodType>(item: Z) =>
  z.union([
    z.union([z.null(), z.undefined()]).transform<z.infer<Z>[]>(() => []),
    item.array(),
  ]);
