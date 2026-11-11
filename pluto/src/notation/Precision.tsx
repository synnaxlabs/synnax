// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Input } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";

const BOUNDS = { lower: 0, upper: 17 };

export interface PrecisionProps extends Input.Control<number> {}

/**
 * Precision edits a number of decimal places. It rounds a typed fraction to a whole
 * number and clamps the value to 0 through 17, the most digits a float64 holds.
 */
export const Precision = ({ value, onChange }: PrecisionProps): ReactElement => (
  <Input.Numeric
    value={value}
    onChange={(v) => onChange(Math.round(v))}
    bounds={BOUNDS}
  />
);
