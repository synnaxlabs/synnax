// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x";

import { type Where } from "@/ranger/types.gen";
import { type Lines } from "@/ranger/where.gen";

/** @returns the typed lines of a range Where, with labels as its labels line. */
export const createLines = (labels: NonNullable<Lines["labels"]>): Lines => ({
  overlaps: (range, value) => new TimeRange(value).overlapsWith(range.timeRange),
  labels,
});

/**
 * @returns whether adding a label to a range, or removing one when removed is true, can
 * make the range match where. A not inverts the change for the lines under it.
 */
export const labelChangeAdmits = (where: Where, removed: boolean): boolean =>
  (where.labels != null && !removed) ||
  where.and?.some((w) => labelChangeAdmits(w, removed)) === true ||
  where.or?.some((w) => labelChangeAdmits(w, removed)) === true ||
  (where.not != null && labelChangeAdmits(where.not, !removed));
