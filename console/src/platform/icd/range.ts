// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import type { bounds } from "@synnaxlabs/x";

/**
 * @returns the raw values of a field, before its scale and offset, that decode inside
 * its min and max. An absent limit is infinite.
 */
export const rawBounds = ({
  scale,
  offset,
  min = -Infinity,
  max = Infinity,
}: Pick<icd.BaseField, "scale" | "offset" | "min" | "max">): bounds.Bounds => {
  const lower = (min - offset) / scale;
  const upper = (max - offset) / scale;
  return scale < 0 ? { lower: upper, upper: lower } : { lower, upper };
};
