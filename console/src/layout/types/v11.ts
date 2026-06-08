// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/layout/types/v1";
import * as v10 from "@/layout/types/v10";

export const VERSION = "11.0.0";

export type WindowPanelsState = v10.WindowPanelsState;

// The Redux mosaic is gone: tiling now lives in server-backed panels read through
// Flux. The per-window mosaic state (state.mosaics) is dropped, and tab focus — the
// fullscreen-one-tab session state formerly stored on mosaic.focused — relocates to
// the top-level `focused` map keyed by window key. Focus is session state and is
// never persisted to the panel document.
export const sliceStateZ = v10.sliceStateZ
  .omit({ version: true, mosaics: true })
  .extend({
    version: z.literal(VERSION),
    focused: z.record(z.string(), z.string().nullable()),
  });

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v10.ZERO_SLICE_STATE,
  version: VERSION,
  focused: {},
});

export const sliceMigration: migrate.Migration<v10.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: ({ mosaics: _mosaics, ...rest }) => ({
      ...rest,
      version: VERSION,
      focused: {},
    }),
  });
