// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Mosaic } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/service/project/layoutMigrations/v1";
import * as v9 from "@/service/project/layoutMigrations/v9";

export const VERSION = "10.0.0";

const GET_STARTED_TYPE = "getStarted";

export const sliceStateZ = v9.sliceStateZ
  .omit({ version: true })
  .extend({ version: z.literal(VERSION) });

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v9.ZERO_SLICE_STATE,
  version: VERSION,
});

export const sliceMigration: migrate.Migration<v9.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: ({ layouts, mosaics, ...rest }) => ({
      ...rest,
      version: VERSION,
      layouts: Object.fromEntries(
        Object.entries(layouts).filter(
          ([, layout]) => layout.type !== GET_STARTED_TYPE,
        ),
      ),
      mosaics: Object.fromEntries(
        Object.entries(mosaics).map(([key, mosaic]) => {
          const [root, next] = Mosaic.removeTab(mosaic.root, GET_STARTED_TYPE);
          const activeTab =
            mosaic.activeTab === GET_STARTED_TYPE ? next : mosaic.activeTab;
          return [key, { ...mosaic, root, activeTab }];
        }),
      ),
    }),
  });
