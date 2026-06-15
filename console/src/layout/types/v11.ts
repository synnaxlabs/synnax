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

export const sliceStateZ = v10.sliceStateZ
  .omit({ version: true })
  .extend({ version: z.literal(VERSION) });

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
});

export const sliceMigration: migrate.Migration<v10.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: (s) => ({ ...s, version: VERSION, nav: renameWorkspaceNav(s.nav) }),
  });
