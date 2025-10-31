// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import { type NavState } from "@/layout/types/v0";
import * as v1 from "@/layout/types/v1";
import * as v5 from "@/layout/types/v5";

const VERSION = "7.0.0";

const ZERO_NAV_STATE: NavState = {
  main: {
    drawers: {
      left: {
        activeItem: null,
        menuItems: ["channel", "range", "workspace", "device", "task", "user", "arc"],
      },
      right: { activeItem: null, menuItems: [] },
      bottom: { activeItem: null, menuItems: ["visualization"] },
    },
  },
};

export const sliceStateZ = v5.sliceStateZ
  .omit({ version: true })
  .extend({ version: z.literal(VERSION) });

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v5.ZERO_SLICE_STATE,
  version: VERSION,
  nav: ZERO_NAV_STATE,
});

export const sliceMigration: migrate.Migration<v5.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: (s) => ({ ...s, version: VERSION, nav: ZERO_NAV_STATE }),
  });
