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
  lineplot,
  NotFoundError,
  project,
  schematic,
  status,
  type Synnax,
  UnexpectedError,
} from "@synnaxlabs/client";
import { dynamicWindow, TimeSpan } from "@synnaxlabs/x";

const NAME = "Sample project";

// Fixed keys make each new sample overwrite the last one, even after a rename, so only
// one copy of the Arc programs drives the sample channels.
export const KEYS = {
  project: "adacd22e-b3a6-4b20-a587-136795288889",
  schematic: "07ba9c3b-d84e-4e85-934d-e2c750355791",
  linePlot: "7a17bb62-2916-496d-9b4a-55e95c91bca4",
  panel: "6800879c-af6f-4cc7-999d-b356369c47ce",
  simulator: "c9873e9f-69bd-473f-8653-dd4e760b19ec",
  sequence: "c5886fec-bd93-4e34-ab17-b53f342e9729",
} as const;

const INDEX = "sample_time";
const TANK_PT = "sample_tank_pt";
const PRESS_VLV_STATE = "sample_press_vlv_state";
const VENT_VLV_STATE = "sample_vent_vlv_state";
const PRESS_VLV_CMD = "sample_press_vlv_cmd";
const VENT_VLV_CMD = "sample_vent_vlv_cmd";
const START_CMD = "sample_start_cmd";
const ABORT_CMD = "sample_abort_cmd";

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

// Handle offsets below the tops of the valve and vent symbols, measured in the Console.
// A pipe between handles at different heights draws a small jog.
const VALVE_HANDLE_Y = 19.296875;
const VENT_HANDLE_Y = 14.78125;

const VENT_VLV_POSITION = { x: 477, y: -35 };

const createSchematic = (ch: Channels): schematic.New => ({
  key: KEYS.schematic,
  name: "Tank",
  nodes: [
    { key: "press_vlv", position: { x: 155, y: 180 } },
    { key: "tank", position: { x: 300, y: 0 } },
    { key: "vent_vlv", position: VENT_VLV_POSITION },
    {
      key: "vent",
      position: {
        x: 590,
        y: VENT_VLV_POSITION.y + VALVE_HANDLE_Y - VENT_HANDLE_Y,
      },
    },
    { key: "tank_pt", position: { x: 310, y: 244 } },
    { key: "start", position: { x: 270, y: 292 } },
    { key: "abort", position: { x: 394, y: 292 } },
  ],
  edges: [
    {
      key: "press_vlv_to_tank",
      source: { node: "press_vlv", param: "2" },
      target: { node: "tank", param: "5" },
    },
    {
      key: "tank_to_vent_vlv",
      source: { node: "tank", param: "3" },
      target: { node: "vent_vlv", param: "1" },
    },
    {
      key: "vent_vlv_to_vent",
      source: { node: "vent_vlv", param: "2" },
      target: { node: "vent", param: "1" },
    },
  ],
  configs: {
    press_vlv: {
      variant: "valve",
      label: { label: "Press valve" },
      commandChannel: ch[PRESS_VLV_CMD],
      stateChannel: ch[PRESS_VLV_STATE],
    },
    tank: {
      variant: "tank",
      label: { label: "Tank" },
      channel: ch[TANK_PT],
      bounds: { lower: 0, upper: 500 },
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
    press_vlv_to_tank: { variant: "pipe" },
    tank_to_vent_vlv: { variant: "pipe" },
    vent_vlv_to_vent: { variant: "pipe" },
  },
});

const createLinePlot = (ch: Channels): lineplot.New => ({
  key: KEYS.linePlot,
  name: "Tank pressure",
  channels: {
    x1: ch[INDEX],
    y1: [ch[TANK_PT]],
    y2: [ch[PRESS_VLV_STATE], ch[VENT_VLV_STATE]],
  },
  ranges: [{ variant: "window", window: dynamicWindow(TimeSpan.seconds(30)) }],
});

const deploy = async (client: Synnax, key: arc.Key): Promise<void> => {
  const racks = await client.racks.retrieve({
    integration: "arc",
    embedded: true,
    hostIsNode: true,
  });
  if (racks.length === 0) throw new NotFoundError("No rack can run Arc programs");
  const task = await client.arcs.updateTask(key, { rack: racks[0].key });
  if (task == null) throw new UnexpectedError(`Arc ${key} has no task`);
  // A start leaves a running Arc alone, so the stop resets its stage and state. The
  // Driver does not answer a stop for a new task, but it runs commands in order, so the
  // start's status also means the stop ran.
  await task.stop();
  const state = await task.executeCommandSync({ type: "start" });
  if (state.variant === "error") throw status.toError(state);
};

export const exists = async (client: Synnax): Promise<boolean> => {
  const found = await client.projects.retrieve({
    keys: [KEYS.project],
    ignoreNotFoundError: true,
  });
  return found.length > 0;
};

/**
 * The text of the sample's Arc programs, from `sampleSimulator.arc` and
 * `sampleSequence.arc`.
 */
export interface Programs {
  simulator: string;
  sequence: string;
}

/**
 * Creates the sample project and starts its Arc programs, replacing any old sample. The
 * line plot shows the last 30 seconds.
 */
export const create = async (client: Synnax, programs: Programs): Promise<void> => {
  const ch = await createChannels(client);
  await client.projects.create({ key: KEYS.project, name: NAME });
  await client.schematics.create(KEYS.project, createSchematic(ch));
  await client.lineplots.create(KEYS.project, createLinePlot(ch));
  // Overwriting an existing schematic or line plot does not link it to the project,
  // which matters after the user deletes the project.
  await client.ontology.addChildren(
    project.ontologyID(KEYS.project),
    schematic.ontologyID(KEYS.schematic),
    lineplot.ontologyID(KEYS.linePlot),
  );
  await client.arcs.create([
    {
      key: KEYS.simulator,
      name: "Tank simulator",
      mode: "text",
      text: { raw: programs.simulator },
    },
    {
      key: KEYS.sequence,
      name: "Press sequence",
      mode: "text",
      text: { raw: programs.sequence },
    },
  ]);
  await deploy(client, KEYS.simulator);
  await deploy(client, KEYS.sequence);
  await client.panels.create({
    key: KEYS.panel,
    name: NAME,
    parent: project.ontologyID(KEYS.project),
    root: {
      variant: "split",
      direction: "x",
      size: 0.55,
      first: {
        variant: "split",
        direction: "y",
        size: 0.55,
        first: {
          variant: "leaf",
          tabs: [
            { variant: "resource", resource: schematic.ontologyID(KEYS.schematic) },
          ],
        },
        last: {
          variant: "leaf",
          tabs: [{ variant: "resource", resource: lineplot.ontologyID(KEYS.linePlot) }],
        },
      },
      last: {
        variant: "leaf",
        tabs: [
          { variant: "resource", resource: arc.ontologyID(KEYS.sequence) },
          { variant: "resource", resource: arc.ontologyID(KEYS.simulator) },
        ],
      },
    },
  });
};
