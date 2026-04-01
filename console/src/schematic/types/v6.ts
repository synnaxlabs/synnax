// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control } from "@synnaxlabs/pluto";
import { migrate, sticky, xy } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/schematic/types/v0";
import * as v1 from "@/schematic/types/v1";
import * as v5 from "@/schematic/types/v5";

export const VERSION = "6.0.0";

export const toolbarTabZ = v0.toolbarTabZ;
export type ToolbarTab = v0.ToolbarTab;

export const viewportZ = z.object({ position: xy.xyZ, zoom: z.number() });
export interface Viewport extends z.infer<typeof viewportZ> {}

export const legendStateZ = z.object({
  visible: z.boolean(),
  position: sticky.xy,
});
export interface LegendState extends z.infer<typeof legendStateZ> {}

export const stateZ = z.object({
  version: z.literal(VERSION),
  selected: z.array(z.string()).default([]),
  control: control.statusZ,
  legend: legendStateZ,
  activeToolbarTab: toolbarTabZ,
  selectedSymbolGroup: z.string().default("general"),
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  viewport: viewportZ,
});
export interface State extends z.infer<typeof stateZ> {}

export const ZERO_STATE: State = {
  version: VERSION,
  selected: [],
  control: "released",
  legend: { visible: false, position: { x: 0, y: 0 } },
  activeToolbarTab: "symbols",
  selectedSymbolGroup: "general",
  editable: true,
  fitViewOnResize: false,
  viewport: { position: xy.ZERO, zoom: 1 },
};

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  schematics: z.record(z.string(), stateZ),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  schematics: {},
};

export const stateMigration = migrate.createMigration<v5.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    version: VERSION,
    selected: [],
    control: state.control,
    legend: { visible: state.legend.visible, position: state.legend.position },
    activeToolbarTab: state.toolbar.activeTab,
    selectedSymbolGroup: state.toolbar.selectedSymbolGroup,
    editable: state.editable,
    fitViewOnResize: state.fitViewOnResize,
    viewport: state.viewport,
  }),
});

export const sliceMigration = migrate.createMigration<v5.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ schematics }) => ({
    version: VERSION,
    schematics: Object.fromEntries(
      Object.entries(schematics).map(([key, schematic]) => [
        key,
        stateMigration(schematic),
      ]),
    ),
  }),
});
