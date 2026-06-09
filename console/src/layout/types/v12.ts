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
import * as v11 from "@/layout/types/v11";

export const VERSION = "12.0.0";

export type WindowPanelsState = v11.WindowPanelsState;

// tabUnsavedChanges tracks, keyed by panel tab key, whether a view tab's form has
// unsaved edits. It is session state — this operator's local draft state — and is
// never persisted to disk nor synced to the panel document (see PERSIST_EXCLUDE).
export const sliceStateZ = v11.sliceStateZ.omit({ version: true }).extend({
  version: z.literal(VERSION),
  tabUnsavedChanges: z.record(z.string(), z.boolean()),
});

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v11.ZERO_SLICE_STATE,
  version: VERSION,
  tabUnsavedChanges: {},
});

export const sliceMigration: migrate.Migration<v11.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: (s) => ({ ...s, version: VERSION, tabUnsavedChanges: {} }),
  });
