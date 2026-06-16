// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate, record } from "@synnaxlabs/x";
import { z } from "zod";

import { type NavState } from "@/layout/types/v0";
import * as v1 from "@/layout/types/v1";
import * as v10 from "@/layout/types/v10";

export const VERSION = "11.0.0";

export const windowPanelsStateZ = z.object({
  active: z.string().nullable(),
  activeTab: z.string().nullable().default(null),
  selected: z.string().array().default([]),
});

export interface WindowPanelsState extends z.infer<typeof windowPanelsStateZ> {}

export const sliceStateZ = v10.sliceStateZ
  .omit({ version: true, mosaics: true, altKeyToKey: true, keyToAltKey: true })
  .extend({
    version: z.literal(VERSION),
    focused: z.record(z.string(), z.string().nullable()),
    windowPanels: z.record(z.string(), windowPanelsStateZ),
    tabUnsavedChanges: z.record(z.string(), z.boolean()),
  });

export interface SliceState extends z.infer<typeof sliceStateZ> {}

const renameItem = (item: string): string => (item === "workspace" ? "project" : item);

const renameWorkspaceNav = (nav: NavState): NavState =>
  record.map(nav, (entry) => ({
    ...entry,
    drawers: record.map(entry.drawers, (drawer) =>
      drawer == null
        ? drawer
        : {
            ...drawer,
            activeItem:
              drawer.activeItem == null ? null : renameItem(drawer.activeItem),
            menuItems: drawer.menuItems.map(renameItem),
          },
    ),
  })) as NavState;

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v10.ZERO_SLICE_STATE,
  version: VERSION,
  nav: renameWorkspaceNav(v10.ZERO_SLICE_STATE.nav),
  focused: {},
  windowPanels: { main: { active: null, activeTab: null, tabHistory: [] } },
  tabUnsavedChanges: {},
});

// MOSAIC_WINDOW_TYPE is the layout type of the legacy detached-mosaic windows.
// Their renderer no longer exists, so the migration drops them along with the
// mosaics themselves.
const MOSAIC_WINDOW_TYPE = "mosaicWindow";

export const sliceMigration: migrate.Migration<v10.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: ({
      mosaics,
      altKeyToKey: _altKeyToKey,
      keyToAltKey: _keyToAltKey,
      layouts,
      ...rest
    }) => ({
      ...rest,
      version: VERSION,
      nav: renameWorkspaceNav(rest.nav),
      layouts: Object.fromEntries(
        Object.entries(layouts).filter(
          ([, layout]) =>
            layout.type !== MOSAIC_WINDOW_TYPE && layout.location !== "mosaic",
        ),
      ),
      focused: {},
      windowPanels: Object.fromEntries(
        Object.keys(mosaics).map((windowKey) => [
          windowKey,
          { active: null, activeTab: null, tabHistory: [] },
        ]),
      ),
      tabUnsavedChanges: {},
    }),
  });
