// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, color } from "@synnaxlabs/x";
import { z } from "zod";

export const redlineZ = z.object({ bounds: bounds.boundsZ, gradient: color.gradientZ });
export type Redline = z.infer<typeof redlineZ>;
export const ZERO_READLINE: Redline = { bounds: { lower: 0, upper: 1 }, gradient: [] };
