// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x";

import { type label } from "@/label";
import { type Payload, type Where } from "@/ranger/types.gen";
import { type Lines } from "@/ranger/where.gen";

/** @returns the typed lines of a range Where, reading labels through labelsOf. */
export const createLines = (labelsOf: (range: Payload) => label.Label[]): Lines => ({
  overlaps: (range, value) => new TimeRange(value).overlapsWith(range.timeRange),
  labels: (range, keys) => {
    const labels = labelsOf(range);
    return keys.some((key) => labels.some((l) => l.key === key));
  },
});

/** @returns whether where has a labels line at any depth. */
export const hasLabels = (where: Where): boolean =>
  where.labels != null ||
  where.and?.some(hasLabels) === true ||
  where.or?.some(hasLabels) === true ||
  (where.not != null && hasLabels(where.not));
