// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { json } from "@synnaxlabs/x";

/**
 * @returns the item of items whose shape holds the most documents, the first of a tie,
 * or null for no items.
 */
export const mostCommon = <T>(items: T[], shapeOf: (item: T) => json.Shape): T | null =>
  items.reduce<T | null>(
    (top, item) =>
      top == null || shapeOf(item).count > shapeOf(top).count ? item : top,
    null,
  );
