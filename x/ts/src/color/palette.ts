// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { colorZ } from "@/color/color";

const styleZ = z.object({
  key: z.string(),
  name: z.string(),
  color: colorZ,
});

export type Style = z.infer<typeof styleZ>;

export const paletteZ = z.object({
  key: z.string(),
  name: z.string(),
  swatches: z.array(styleZ),
});

export type Palette = z.infer<typeof paletteZ>;
