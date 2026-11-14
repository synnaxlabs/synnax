// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x";

import { type Lines } from "@/ranger/where.gen";

/** The typed lines of a range Where. */
export const LINES: Lines = {
  overlaps: (range, value) => new TimeRange(value).overlapsWith(range.timeRange),
  labels: (range, keys) => range.labels.some((key) => keys.includes(key)),
  parents: (range, keys) => range.parent != null && keys.includes(range.parent),
  nested: (range, nested) => (range.parent != null) === nested,
  labeled: (range, labeled) => range.labels.length > 0 === labeled,
  // Both read the Core's clock, so only the Core checks them.
  stages: null,
  recent: null,
};
