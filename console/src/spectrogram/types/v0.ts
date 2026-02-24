// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const VERSION = "0.0.0";

export type ToolbarTab = "data" | "display";

export const toolbarStateZ = z.object({
  activeTab: z.enum(["data", "display"]).default("data"),
});
export type ToolbarState = z.infer<typeof toolbarStateZ>;
export const ZERO_TOOLBAR_STATE: ToolbarState = { activeTab: "data" };

export const stateZ = z.object({
  version: z.literal(VERSION).default(VERSION),
  key: z.string(),
  channel: z.number().default(0),
  sampleRate: z.number().default(48000),
  fftSize: z.number().default(2048),
  windowFunction: z.enum(["hann", "blackmanHarris"]).default("hann"),
  overlap: z.number().default(0.5),
  colorMap: z
    .enum(["viridis", "inferno", "magma", "plasma", "jet", "grayscale"])
    .default("viridis"),
  dbMin: z.number().default(-100),
  dbMax: z.number().default(0),
  freqMin: z.number().default(0),
  freqMax: z.number().default(0),
  toolbar: toolbarStateZ.default(ZERO_TOOLBAR_STATE),
});

export type State = z.infer<typeof stateZ>;

export const ZERO_STATE: State = {
  version: VERSION,
  key: "",
  channel: 0,
  sampleRate: 48000,
  fftSize: 2048,
  windowFunction: "hann",
  overlap: 0.5,
  colorMap: "viridis",
  dbMin: -100,
  dbMax: 0,
  freqMin: 0,
  freqMax: 0,
  toolbar: ZERO_TOOLBAR_STATE,
};

export const sliceStateZ = z.object({
  version: z.literal(VERSION).default(VERSION),
  spectrograms: z.record(z.string(), stateZ).default({}),
});

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  spectrograms: {},
};
