// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RGBATuple } from "./types";

export const hexToRGBA = (hex: string, alpha: number = 1): RGBATuple => [
  p(hex, 1),
  p(hex, 3),
  p(hex, 5),
  alpha,
];

const p = (s: string, n: number): number => parseInt(s.slice(n, n + 2), 16);
