// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  arc,
  type channel,
  DataType,
  DisconnectedError,
  lineplot,
  NotFoundError,
  project,
  schematic,
  type Synnax,
  UnexpectedError,
} from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Session } from "@/session";

const NAME = "Sample project";

const INDEX = "sample_time";
const TANK_PT = "sample_tank_pt";
const PRESS_VLV_STATE = "sample_press_vlv_state";
const VENT_VLV_STATE = "sample_vent_vlv_state";
const PRESS_VLV_CMD = "sample_press_vlv_cmd";
const VENT_VLV_CMD = "sample_vent_vlv_cmd";
const START_CMD = "sample_start_cmd";
const ABORT_CMD = "sample_abort_cmd";

const SIMULATOR = `import time

// Simulates a tank that fills from a 500 psi nitrogen supply through the press valve
// and empties through the vent valve.
func step() {
    pressure f64 $= 0
    if ${PRESS_VLV_CMD} == 1 {
        pressure += (500 - pressure) * 0.02
    }
    if ${VENT_VLV_CMD} == 1 {
        pressure -= pressure * 0.05
    }
    pressure -= pressure * 0.0002
    ${TANK_PT} = pressure
    ${PRESS_VLV_STATE} = ${PRESS_VLV_CMD}
    ${VENT_VLV_STATE} = ${VENT_VLV_CMD}
}

time.interval{period=100ms} -> step{}
`;

const SEQUENCE = `import control
import time

// The sequence runs below the schematic's authority, so an operator can take control
// of the valves at any time. An abort takes absolute authority.
authority 100

sequence main {
    stage idle {
        0 -> ${PRESS_VLV_CMD}
        0 -> ${VENT_VLV_CMD}
        ${START_CMD} => pressurize
    }
    stage pressurize {
        control.set_authority{value=100}
        1 -> ${PRESS_VLV_CMD}
        0 -> ${VENT_VLV_CMD}
        ${TANK_PT} > 450 => abort
        ${ABORT_CMD} => abort
        ${TANK_PT} > 300 => hold
    }
    stage hold {
        0 -> ${PRESS_VLV_CMD}
        ${ABORT_CMD} => abort
        time.wait{duration=10s} => vent
    }
    stage vent {
        1 -> ${VENT_VLV_CMD}
        ${TANK_PT} < 5 => idle
    }
    stage abort {
        control.set_authority{value=255}
        0 -> ${PRESS_VLV_CMD}
        1 -> ${VENT_VLV_CMD}
        ${START_CMD} => pressurize
    }
}

1 => main
`;

/** The schematic control authority that lets an operator override the sequence. */
export const SAMPLE_SCHEMATIC_AUTHORITY = 200;

type Channels = Record<
  | typeof INDEX
  | typeof TANK_PT
  | typeof PRESS_VLV_STATE
  | typeof VENT_VLV_STATE
  | typeof PRESS_VLV_CMD
  | typeof VENT_VLV_CMD
  | typeof START_CMD
  | typeof ABORT_CMD,
  channel.Key
>;

const createChannels = async (client: Synnax): Promise<Channels> => {
  const opts = { retrieveIfNameExists: true };
  const index = await client.channels.create(
    { name: INDEX, dataType: DataType.TIMESTAMP, isIndex: true },
    opts,
  );
  const created = await client.channels.create(
    [
      { name: TANK_PT, dataType: DataType.FLOAT64, index: index.key },
      { name: PRESS_VLV_STATE, dataType: DataType.UINT8, index: index.key },
      { name: VENT_VLV_STATE, dataType: DataType.UINT8, index: index.key },
      { name: PRESS_VLV_CMD, dataType: DataType.UINT8, virtual: true },
      { name: VENT_VLV_CMD, dataType: DataType.UINT8, virtual: true },
      { name: START_CMD, dataType: DataType.UINT8, virtual: true },
      { name: ABORT_CMD, dataType: DataType.UINT8, virtual: true },
    ],
    opts,
  );
  const keys = Object.fromEntries(created.map((ch) => [ch.name, ch.key]));
  return { ...keys, [INDEX]: index.key } as Channels;
};

const createSchematic = (ch: Channels): schematic.New => ({
  name: "Tank",
  nodes: [
    { key: "supply", position: { x: 0, y: 28 } },
    { key: "press_vlv", position: { x: 140, y: 80 } },
    { key: "tank", position: { x: 300, y: 0 } },
    { key: "vent_vlv", position: { x: 500, y: 80 } },
    { key: "vent", position: { x: 660, y: 84 } },
    { key: "tank_pt", position: { x: 318, y: 240 } },
    { key: "start", position: { x: 250, y: 320 } },
    { key: "abort", position: { x: 370, y: 320 } },
  ],
  edges: [
    {
      key: "supply_to_press_vlv",
      source: { node: "supply", param: "4" },
      target: { node: "press_vlv", param: "1" },
    },
    {
      key: "press_vlv_to_tank",
      source: { node: "press_vlv", param: "2" },
      target: { node: "tank", param: "7" },
    },
    {
      key: "tank_to_vent_vlv",
      source: { node: "tank", param: "8" },
      target: { node: "vent_vlv", param: "1" },
    },
    {
      key: "vent_vlv_to_vent",
      source: { node: "vent_vlv", param: "2" },
      target: { node: "vent", param: "1" },
    },
  ],
  configs: {
    supply: { variant: "cylinder", label: { label: "Nitrogen supply" } },
    press_vlv: {
      variant: "valve",
      label: { label: "Press valve" },
      commandChannel: ch[PRESS_VLV_CMD],
      stateChannel: ch[PRESS_VLV_STATE],
    },
    tank: {
      variant: "tank",
      label: { label: "Tank" },
      fill: { channel: ch[TANK_PT], bounds: { lower: 0, upper: 500 } },
    },
    vent_vlv: {
      variant: "valve",
      label: { label: "Vent valve" },
      commandChannel: ch[VENT_VLV_CMD],
      stateChannel: ch[VENT_VLV_STATE],
    },
    vent: { variant: "vent" },
    tank_pt: {
      variant: "value",
      label: { label: "Tank pressure" },
      channel: ch[TANK_PT],
      units: "psi",
      precision: 1,
    },
    start: {
      variant: "button",
      label: { label: "Start" },
      commandChannel: ch[START_CMD],
    },
    abort: {
      variant: "button",
      label: { label: "Abort" },
      commandChannel: ch[ABORT_CMD],
    },
    supply_to_press_vlv: { variant: "pipe" },
    press_vlv_to_tank: { variant: "pipe" },
    tank_to_vent_vlv: { variant: "pipe" },
    vent_vlv_to_vent: { variant: "pipe" },
  },
});

const createLinePlot = (ch: Channels): lineplot.New => ({
  name: "Tank pressure",
  channels: {
    x1: ch[INDEX],
    y1: [ch[TANK_PT]],
    y2: [ch[PRESS_VLV_STATE], ch[VENT_VLV_STATE]],
  },
  ranges: { x1: [Session.Range.RECENT_KEY] },
});

const deploy = async (client: Synnax, key: arc.Key): Promise<void> => {
  const racks = await client.racks.retrieve({
    integration: "arc",
    embedded: true,
    hostIsNode: true,
  });
  if (racks.length === 0) throw new NotFoundError("No rack can run Arc programs");
  const task = await client.arcs.setRack(key, racks[0].key);
  if (task == null) throw new UnexpectedError(`Arc ${key} has no task`);
  await task.start();
};

export interface Sample {
  project: project.Key;
  schematic: schematic.Key;
}

/**
 * Creates the sample tank pressurization project: its channels, a schematic, a line
 * plot, and two running Arc programs, one that simulates the tank and one that
 * pressurizes it. Each call creates a new project and new Arc programs, and reuses the
 * sample channels when they exist.
 */
export const createSample = async (client: Synnax): Promise<Sample> => {
  const ch = await createChannels(client);
  const proj = await client.projects.create({ name: NAME, layout: {} });
  const sch = await client.schematics.create(proj.key, createSchematic(ch));
  const plot = await client.lineplots.create(proj.key, createLinePlot(ch));
  const [simulator, sequence] = await client.arcs.create([
    { name: "Tank simulator", mode: "text", text: { raw: SIMULATOR } },
    { name: "Press sequence", mode: "text", text: { raw: SEQUENCE } },
  ]);
  await deploy(client, simulator.key);
  await deploy(client, sequence.key);
  await client.panels.create({
    name: NAME,
    parent: project.ontologyID(proj.key),
    root: {
      variant: "split",
      direction: "x",
      size: 0.55,
      first: {
        variant: "leaf",
        tabs: [{ variant: "resource", resource: schematic.ontologyID(sch.key) }],
      },
      last: {
        variant: "split",
        direction: "y",
        size: 0.5,
        first: {
          variant: "leaf",
          tabs: [{ variant: "resource", resource: lineplot.ontologyID(plot.key) }],
        },
        last: {
          variant: "leaf",
          tabs: [
            { variant: "resource", resource: arc.ontologyID(sequence.key) },
            { variant: "resource", resource: arc.ontologyID(simulator.key) },
          ],
        },
      },
    },
  });
  return { project: proj.key, schematic: sch.key };
};

/** Returns a callback that creates the sample project and opens it. */
export const useOpenSample = (): (() => void) => {
  const client = PSynnax.use();
  const dispatch = Session.useDispatch();
  const handleError = Status.useErrorHandler();
  return useCallback(
    () =>
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const sample = await createSample(client);
        dispatch(
          Session.Schematic.create({
            key: sample.schematic,
            editable: false,
            control: { authority: SAMPLE_SCHEMATIC_AUTHORITY },
          }),
        );
        dispatch(Session.Project.select(sample.project));
      }, "Failed to open the sample project"),
    [client, dispatch, handleError],
  );
};
