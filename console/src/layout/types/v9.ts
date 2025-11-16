// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate, record } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/layout/types/v0";
import * as v1 from "@/layout/types/v1";
import * as v8 from "@/layout/types/v8";

export const VERSION = "9.0.0";

export const navDrawerEntryStateZ = v0.navDrawerEntryStateZ.omit({
  menuItems: true,
});

export type NavDrawerEntryState = z.infer<typeof navDrawerEntryStateZ>;

const navDrawerStateZ = z.object({
  left: navDrawerEntryStateZ,
  right: navDrawerEntryStateZ,
  bottom: navDrawerEntryStateZ,
});

const mainNavStateZ = z.object({ drawers: navDrawerStateZ });

const MAIN_LAYOUT_KEY = "main";

const partialNavStateZ = z.object({ drawers: navDrawerStateZ.partial() });

const navStateZ = z
  .record(z.string(), partialNavStateZ)
  .and(z.object({ [MAIN_LAYOUT_KEY]: mainNavStateZ }));

type NavState = z.infer<typeof navStateZ>;

const ZERO_NAV_STATE: NavState = {
  main: {
    drawers: {
      left: { activeItem: null },
      right: { activeItem: null },
      bottom: { activeItem: null },
    },
  },
};

export const sliceStateZ = v8.sliceStateZ
  .omit({ version: true, nav: true })
  .extend({ version: z.literal(VERSION), nav: navStateZ });

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v8.ZERO_SLICE_STATE,
  version: VERSION,
  nav: ZERO_NAV_STATE,
});

const migrateDrawers = (drawers: Record<string, v0.NavDrawerEntryState | undefined>) =>
  Object.fromEntries(
    Object.entries(drawers).map(([location, drawer]) => {
      if (drawer == null) return [location, drawer];
      return [location, record.omit(drawer, "menuItems")];
    }),
  );

export const sliceMigration: migrate.Migration<v8.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: (s): SliceState => {
      const nav = Object.fromEntries(
        Object.entries(s.nav).map(([layoutKey, layoutNav]) => {
          if (layoutNav.drawers == null) return [layoutKey, layoutNav];
          return [layoutKey, { drawers: migrateDrawers(layoutNav.drawers) }];
        }),
      ) as NavState;

      return { ...s, version: VERSION, nav };
    },
  });
