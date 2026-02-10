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

import { TYPE as ARC_TYPE } from "@/arc/types";
import * as v1 from "@/layout/types/v1";
import * as v8 from "@/layout/types/v8";

export const VERSION = "9.0.0";

export const sliceStateZ = v8.sliceStateZ
  .omit({ version: true })
  .extend({ version: z.literal(VERSION) });

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v8.ZERO_SLICE_STATE,
  version: VERSION,
});

export const sliceMigration: migrate.Migration<v8.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: ({ layouts, ...rest }) => ({
      ...rest,
      version: VERSION,
      layouts: Object.fromEntries(
        Object.entries(layouts).map(([key, layout]) => [
          key,
          { ...layout, type: layout.type === "arc_editor" ? ARC_TYPE : layout.type },
        ]),
      ),
    }),
  });
