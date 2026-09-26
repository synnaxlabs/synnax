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

/** Default typography level for each component size. */
export const SIZE_TEXT_LEVELS: Record<Size, text.Level> = {
  tiny: "small",
  small: "small",
  medium: "p",
  large: "h5",
  huge: "h2",
};

/** Default component size for each typography level. */
export const TEXT_LEVEL_SIZES: Record<text.Level, Size> = {
  h1: "huge",
  h2: "huge",
  h3: "huge",
  h4: "large",
  h5: "small",
  p: "medium",
  small: "small",
};
