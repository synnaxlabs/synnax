// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type library, type Synnax, type task } from "@synnaxlabs/client";
import { id, type record, TimeSpan } from "@synnaxlabs/x";
import { screen } from "@testing-library/react";
import { type FC } from "react";

import { type FormTabProps } from "@/platform/task/Form";
import {
  awaitEditableForm,
  renderTaskFormTab,
  type RenderTaskFormTabResult,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

export interface BusLibrary {
  library: library.Library;
  /** A CAN message with the binary fields Rpm and Temp. */
  engine: library.MessageEntry;
  /** A text message matched by the token S, with the delimited fields Volts and Amps. */
  status: library.MessageEntry;
}

const findMessage = (lib: library.Library, name: string): library.MessageEntry => {
  const entry = lib.entries.find((e) => e.kind === "message" && e.name === name);
  if (entry?.kind !== "message") throw new Error(`message ${name} not found`);
  return entry;
};

/** Creates a library holding one CAN message and one text message. */
export const createBusLibrary = async (client: Synnax): Promise<BusLibrary> => {
  const lib = await client.libraries.create({
    name: uniqueName("bus_library"),
    entries: [
      {
        kind: "message",
        name: "Engine",
        identifier: { type: "can", id: 0x100, extended: false, fd: false },
        length: 8,
        fields: [
          { encoding: "binary", name: "Rpm", startBit: 0, bitLength: 16, units: "rpm" },
          { encoding: "binary", name: "Temp", startBit: 16, bitLength: 8 },
        ],
      },
      {
        kind: "message",
        name: "Status",
        format: "text",
        identifier: { type: "token", prefix: "S" },
        fields: [
          { encoding: "delimited", name: "Volts", position: 1 },
          { encoding: "delimited", name: "Amps", position: 2 },
        ],
      },
    ],
  });
  return {
    library: lib,
    engine: findMessage(lib, "Engine"),
    status: findMessage(lib, "Status"),
  };
};

export interface AvionicsLibrary {
  library: library.Library;
  /** An ARINC 429 message on label 206 with the field Speed. */
  airspeed: library.MessageEntry;
  /** An ARINC 429 message on label 203 that a task polls with a query. */
  polled: library.MessageEntry;
  /** A 1553 transmit message from terminal 5 every 100ms, with Pitch and Roll. */
  attitude: library.MessageEntry;
  /** A 1553 transmit message from terminal 5 with no period, with the field Mode. */
  status: library.MessageEntry;
  /** A 1553 receive message to terminal 5 with the field Setpoint. */
  command: library.MessageEntry;
  /** A 1553 receive message to terminal 7 with the field Throttle. */
  engine: library.MessageEntry;
  /** A CAN message with the field Rpm. */
  can: library.MessageEntry;
}

const ARINC_FIELD = { encoding: "binary", startBit: 10, bitLength: 19 } as const;
const WORD = { encoding: "binary", bitLength: 16 } as const;

const arinc429 = (label: number) => ({ type: "arinc429", label }) as const;

const mil1553 = (rt: number, subaddress: number, direction: library.Direction) =>
  ({ type: "mil1553", rt, subaddress, direction, wordCount: 2 }) as const;

/** Creates a library holding ARINC 429, MIL-STD-1553, and CAN messages. */
export const createAvionicsLibrary = async (
  client: Synnax,
): Promise<AvionicsLibrary> => {
  const lib = await client.libraries.create({
    name: uniqueName("avionics_library"),
    entries: [
      {
        kind: "message",
        name: "Airspeed",
        identifier: arinc429(0o206),
        length: 4,
        fields: [{ ...ARINC_FIELD, name: "Speed", units: "kn" }],
      },
      {
        kind: "message",
        name: "Polled",
        identifier: arinc429(0o203),
        length: 4,
        query: "\x01",
        fields: [{ ...ARINC_FIELD, name: "Altitude" }],
      },
      {
        kind: "message",
        name: "Attitude",
        identifier: mil1553(5, 1, "transmit"),
        length: 4,
        period: TimeSpan.milliseconds(100),
        fields: [
          { ...WORD, name: "Pitch", startBit: 0 },
          { ...WORD, name: "Roll", startBit: 16 },
        ],
      },
      {
        kind: "message",
        name: "Status",
        identifier: mil1553(5, 2, "transmit"),
        length: 4,
        fields: [{ ...WORD, name: "Mode", startBit: 0 }],
      },
      {
        kind: "message",
        name: "Command",
        identifier: mil1553(5, 3, "receive"),
        length: 4,
        fields: [{ ...WORD, name: "Setpoint", startBit: 0 }],
      },
      {
        kind: "message",
        name: "Engine",
        identifier: mil1553(7, 1, "receive"),
        length: 4,
        fields: [{ ...WORD, name: "Throttle", startBit: 0 }],
      },
      {
        kind: "message",
        name: "Wheel",
        identifier: { type: "can", id: 0x200, extended: false, fd: false },
        length: 8,
        fields: [{ ...WORD, name: "Rpm", startBit: 0 }],
      },
    ],
  });
  return {
    library: lib,
    airspeed: findMessage(lib, "Airspeed"),
    polled: findMessage(lib, "Polled"),
    attitude: findMessage(lib, "Attitude"),
    status: findMessage(lib, "Status"),
    command: findMessage(lib, "Command"),
    engine: findMessage(lib, "Engine"),
    can: findMessage(lib, "Wheel"),
  };
};

export interface CreateBusDeviceOptions {
  configured?: boolean;
  model?: string;
}

/** Creates a rack and a device of the given make and properties on the Core. */
export const createBusDevice = async (
  client: Synnax,
  make: string,
  properties: record.Unknown,
  { configured = true, model = make }: CreateBusDeviceOptions = {},
): Promise<device.Device> => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await client.devices.create({
    key: id.create(),
    name: uniqueName(`${make.toLowerCase()}_device`),
    rack: rack.key,
    location: make,
    make,
    model,
    configured,
    properties,
  });
};

/** Creates a task row of the given type and config. */
export const createBusTask = async (
  client: Synnax,
  type: string,
  config: record.Unknown,
): Promise<task.Task> =>
  await client.tasks.create({ name: uniqueName("bus_task"), type, config });

/** Renders a bus task form and waits for it to become editable. */
export const renderBusTask = async (
  Form: FC<FormTabProps>,
  client: Synnax,
  taskKey: task.Key,
): Promise<RenderTaskFormTabResult> => {
  const rendered = await renderTaskFormTab(Form, { client, taskKey });
  await awaitEditableForm();
  return rendered;
};

/** Finds the row of the message field with the given name in a task's field list. */
export const findFieldRow = async (name: string): Promise<HTMLElement> => {
  const checkbox = await screen.findByRole("checkbox", { name });
  const row = checkbox.closest<HTMLElement>(".console-bus-field");
  if (row == null) throw new Error(`no field row holds the ${name} checkbox`);
  return row;
};
