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

import * as v1 from "@/layout/types/v1";
import * as v9 from "@/layout/types/v9";

export const VERSION = "10.0.0";

const GET_STARTED_TYPE = "getStarted";

export const panelMetaZ = z.object({
  key: z.string(),
  windowKey: z.string(),
  name: z.string(),
  pinned: z.boolean(),
  ephemeral: z.boolean(),
});

export interface PanelMeta extends z.infer<typeof panelMetaZ> {}

export const windowPanelsStateZ = z.object({
  order: z.string().array(),
  active: z.string().nullable(),
});

export interface WindowPanelsState extends z.infer<typeof windowPanelsStateZ> {}

export const sliceStateZ = v9.sliceStateZ.omit({ version: true }).extend({
  version: z.literal(VERSION),
  panels: z.record(z.string(), panelMetaZ),
  windowPanels: z.record(z.string(), windowPanelsStateZ),
});

export interface SliceState extends z.infer<typeof sliceStateZ> {}

const DEFAULT_PANEL_NAME = "Default";

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v9.ZERO_SLICE_STATE,
  version: VERSION,
  panels: {
    main: {
      key: "main",
      windowKey: "main",
      name: DEFAULT_PANEL_NAME,
      pinned: true,
      ephemeral: false,
    },
  },
  windowPanels: { main: { order: ["main"], active: "main" } },
});

export const sliceMigration: migrate.Migration<v9.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: ({ layouts, mosaics, ...rest }) => {
      const nextMosaics = Object.fromEntries(
        Object.entries(mosaics).map(([key, mosaic]) => {
          const [root, next] = Mosaic.removeTab(mosaic.root, GET_STARTED_TYPE);
          const activeTab =
            mosaic.activeTab === GET_STARTED_TYPE ? next : mosaic.activeTab;
          return [key, { ...mosaic, root, activeTab }];
        }),
      );
      const panels: Record<string, PanelMeta> = {};
      const windowPanels: Record<string, WindowPanelsState> = {};
      for (const windowKey of Object.keys(nextMosaics)) {
        panels[windowKey] = {
          key: windowKey,
          windowKey,
          name: DEFAULT_PANEL_NAME,
          pinned: true,
          ephemeral: false,
        };
        windowPanels[windowKey] = { order: [windowKey], active: windowKey };
      }
      return {
        ...rest,
        version: VERSION,
        layouts: Object.fromEntries(
          Object.entries(layouts).filter(
            ([, layout]) => layout.type !== GET_STARTED_TYPE,
          ),
        ),
        mosaics: nextMosaics,
        panels,
        windowPanels,
      };
    },
  });
