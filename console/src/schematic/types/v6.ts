// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control, Diagram, Viewport } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/schematic/types/v0";
import * as v1 from "@/schematic/types/v1";
import type * as v5 from "@/schematic/types/v5";

export const VERSION = "6.0.0";

export const stateZ = z.object({
  version: z.literal(VERSION),
  selected: z.string().array(),
  viewport: Diagram.viewportZ,
  control: control.statusZ,
  controlAcquireTrigger: z.number(),
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  mode: Viewport.modeZ,
  toolbar: v0.toolbarStateZ,
});

export interface State extends z.infer<typeof stateZ> {}

export const ZERO_STATE: State = {
  version: VERSION,
  selected: [],
  viewport: { position: { x: 0, y: 0 }, zoom: 1 },
  control: "released",
  controlAcquireTrigger: 0,
  editable: true,
  fitViewOnResize: false,
  mode: "select",
  toolbar: v0.ZERO_TOOLBAR_STATE,
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
  migrate: (state) => {
    const selectedNodes = state.nodes
      .filter((node) => node.selected)
      .map((node) => node.key);
    const selectedEdges = state.edges
      .filter((edge) => edge.selected)
      .map((edge) => edge.key);
    return {
      version: VERSION,
      selected: [...selectedNodes, ...selectedEdges],
      viewport: state.viewport,
      control: state.control,
      controlAcquireTrigger: state.controlAcquireTrigger,
      editable: state.editable,
      fitViewOnResize: state.fitViewOnResize,
      mode: state.mode,
      toolbar: state.toolbar,
    };
  },
});

export const sliceMigration = migrate.createMigration<v5.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: (state) => ({
    version: VERSION,
    schematics: Object.fromEntries(
      Object.entries(state.schematics).map(([key, schematic]) => [
        key,
        stateMigration(schematic),
      ]),
    ),
  }),
});
