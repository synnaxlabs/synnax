// Copyright 2025 Synnax Labs, Inc.
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

/**
 * For optional arrays: normalizes null to undefined, preserves [].
 * Use when you need to distinguish "didn't ask" (undefined) from "asked, found nothing" ([]).
 *
 * - null → undefined (Go nil becomes JS undefined)
 * - undefined → undefined (didn't ask)
 * - [] → [] (asked, found nothing)
 * - [items] → [items]
 */
export const nullToUndefined = <Z extends z.ZodType>(item: Z) =>
  z
    .union([z.null().transform(() => undefined), z.undefined(), item.array()])
    .optional();
