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

import * as v0 from "@/layout/types/v0";

export const VERSION = "0.1.0";

export const sliceStateZ = v0.sliceStateZ
  .omit({ version: true })
  .extend({ version: z.literal(VERSION) });

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  ...v0.ZERO_SLICE_STATE,
  version: VERSION,
};

export const SLICE_MIGRATION_NAME = "layout.slice";

export const sliceMigration: migrate.Migration<v0.SliceState, SliceState> =
  migrate.createMigration({
    name: SLICE_MIGRATION_NAME,
    migrate: (s) => ({ ...s, version: VERSION }),
  });
