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

import * as v1 from "@/project/services/layoutMigrations/v1";
import * as v10 from "@/project/services/layoutMigrations/v10";

export const VERSION = "11.0.0";

// The nav drawer state has moved out of the layout slice into its own slice; the field
// is dropped from the schema here so persisted layouts shed it on load.
export const sliceStateZ = v10.sliceStateZ
  .omit({ version: true, nav: true })
  .extend({ version: z.literal(VERSION) });

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v10.ZERO_SLICE_STATE,
  version: VERSION,
});

export const sliceMigration: migrate.Migration<v10.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: ({ nav: _nav, ...rest }) => ({ ...rest, version: VERSION }),
  });
