// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type library, type Synnax, type task } from "@synnaxlabs/client";
import { id, type record } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { type FC } from "react";

import { type Task } from "@/platform/task";
import {
  awaitEditableForm,
  commitFieldInput,
  renderTaskFormTab,
  type RenderTaskFormTabResult,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

export interface BusLibrary {
  library: library.Library;
  /** A CAN message with the binary fields Rpm and Temp. */
  engine: library.MessageEntry;
  /** A CAN message with the binary field Pressure. */
  brake: library.MessageEntry;
  /** A text message matched by token S, with the delimited fields Volts and Amps. */
  status: library.MessageEntry;
}

const findMessage = (lib: library.Library, name: string): library.MessageEntry => {
  const entry = lib.entries.find((e) => e.kind === "message" && e.name === name);
  if (entry?.kind !== "message") throw new Error(`message ${name} not found`);
  return entry;
};

/** Creates a library holding two CAN messages and one text message. */
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
        name: "Brake",
        identifier: { type: "can", id: 0x200, extended: false, fd: false },
        length: 8,
        fields: [{ encoding: "binary", name: "Pressure", startBit: 0, bitLength: 16 }],
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
    brake: findMessage(lib, "Brake"),
    status: findMessage(lib, "Status"),
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

/** Creates a task of the given type and config. */
export const createBusTask = async (
  client: Synnax,
  type: string,
  config: record.Unknown,
): Promise<task.Task> =>
  await client.tasks.create({ name: uniqueName("bus_task"), type, config });

/** Renders a bus task form and waits for it to become editable. */
export const renderBusTask = async (
  Form: FC<Task.FormTabProps>,
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

/** Creates the config entry of a task message that binds every field of entry. */
export const createMessage = (entry: library.MessageEntry) => ({
  message: entry.key,
  fields: entry.fields.map((f) => ({ field: f.key })),
});

/** Commits value into the text or numeric input with the given label. */
export const enterField = async (label: string, value: string): Promise<void> =>
  commitFieldInput(await screen.findByLabelText<HTMLInputElement>(label), value);

/** Finds the dialog opened last, such as the picker a click just opened. */
export const findOpenDialog = async (): Promise<HTMLElement> =>
  await waitFor(() => {
    const dialogs = screen.getAllByRole("dialog");
    return dialogs[dialogs.length - 1];
  });

export const BUS_TASK_TYPES = [
  "can_read",
  "can_write",
  "serial_read",
  "serial_write",
  "tcp_read",
  "tcp_write",
  "udp_read",
  "udp_write",
];

export const BUS_CONNECT_COMMANDS = [
  "can_connect_device",
  "serial_connect_device",
  "tcp_connect_device",
  "udp_connect_device",
];

export const BUS_TASK_COMMANDS = BUS_TASK_TYPES.map((type) => {
  const [prefix, kind] = type.split("_");
  return `${prefix}_create_${kind}_task`;
});
