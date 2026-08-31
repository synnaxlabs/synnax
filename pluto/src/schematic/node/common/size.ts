// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type text } from "@synnaxlabs/x";

import { type Size } from "@/component/size";

/** The text level a symbol renders at for each size rung. */
export const SIZE_LEVELS: Record<Size, text.Level> = {
  tiny: "small",
  small: "h5",
  medium: "h4",
  large: "h3",
  huge: "h2",
};

/** The rung for each text level. Unoffered levels map to medium. */
export const LEVEL_SIZES: Record<text.Level, Size> = {
  h1: "medium",
  h2: "huge",
  h3: "large",
  h4: "medium",
  h5: "small",
  p: "medium",
  small: "tiny",
};
