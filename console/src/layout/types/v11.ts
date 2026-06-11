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

// WindowPanelsState is the per-window session cursor into the project's panels:
// which panel is active in this window, and which tab within it. The panels
// themselves (tree, name, membership) are document state owned by the server
// and read through Flux — they do not live in Redux.
export const windowPanelsStateZ = z.object({
  active: z.string().nullable(),
  activeTab: z.string().nullable().default(null),
  // tabHistory is the window's most-recently-active tab keys (most recent first).
  // The mosaic adapter uses it to keep a per-leaf selection: each leaf shows the
  // most recently active of its tabs, so selecting a tab in one leaf does not
  // snap sibling leaves back to their first tab.
  tabHistory: z.string().array().default([]),
});

export interface WindowPanelsState extends z.infer<typeof windowPanelsStateZ> {}

// The Redux mosaic is gone: tiling now lives in server-backed panels read through
// Flux. The per-window mosaic state (state.mosaics) is dropped, along with the
// alt-key maps that existed to track mosaic tabs whose layout key changed after a
// server create (task forms now render as panel view tabs with stable tab keys).
// Tab focus — the fullscreen-one-tab session state formerly stored on
// mosaic.focused — relocates to the top-level `focused` map keyed by window key.
// tabUnsavedChanges tracks, keyed by panel tab key, whether a view tab's form has
// unsaved edits. focused and tabUnsavedChanges are session state and are never
// persisted to the panel document (see PERSIST_EXCLUDE).
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

// renameWorkspaceNav replaces the legacy "workspace" nav menu-item key with
// "project" across every drawer, preserving menu order and any user
// customization. The workspace nav toolbar is registered under the "project"
// key after the rename, so persisted menus that still reference "workspace"
// would otherwise stop resolving.
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
