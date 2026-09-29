// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arinc429, type library, mil1553 } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type record } from "@synnaxlabs/x";
import { fireEvent, screen, within } from "@testing-library/react";
import { type FC } from "react";
import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { ARINC429 } from "@/feature/arinc429";
import {
  type AvionicsLibrary,
  createAvionicsLibrary,
  createBusDevice,
  createBusTask,
  createMessage,
  findMessageError,
  findOpenDialog,
  renderBusTask,
} from "@/feature/bus/testutil";
import { MIL1553 } from "@/feature/mil1553";
import { type Task } from "@/platform/task";
import { clickDeploy, deployAndAwaitTask } from "@/platform/task/testutil";
import { findDialogTriggerByText } from "@/testutil";

const client = createTestClient();

interface Draft {
  Form: FC<Task.FormTabProps>;
  type: string;
  configZ: z.ZodType<record.Unknown>;
  make: string;
  properties: record.Unknown;
  /** The library messages the draft's config holds. */
  messages: (lib: AvionicsLibrary) => library.MessageEntry[];
}

const ARINC429_READ = {
  Form: ARINC429.Task.Read,
  type: ARINC429.Task.READ_TYPE,
  configZ: ARINC429.Task.READ_SCHEMAS.config,
  make: ARINC429.Device.MAKE,
  properties: arinc429.propertiesZ.parse({}),
};

const mil1553Read = (properties: Partial<mil1553.Properties>) => ({
  Form: MIL1553.Task.Read,
  type: MIL1553.Task.READ_TYPE,
  configZ: MIL1553.Task.READ_SCHEMAS.config,
  make: MIL1553.Device.MAKE,
  properties: mil1553.propertiesZ.parse(properties),
});

const mil1553Write = (properties: Partial<mil1553.Properties>) => ({
  ...mil1553Read(properties),
  Form: MIL1553.Task.Write,
  type: MIL1553.Task.WRITE_TYPE,
  configZ: MIL1553.Task.WRITE_SCHEMAS.config,
});

/** Opens a draft of the given task with the picked messages in its form. */
const open = async ({ Form, type, configZ, make, properties, messages }: Draft) => {
  const lib = await createAvionicsLibrary(client);
  const dev = await createBusDevice(client, make, properties);
  const picked = messages(lib);
  const config = configZ.parse({
    library: lib.library.key,
    device: dev.key,
    messages: picked.map(createMessage),
  });
  const draft = await createBusTask(client, type, config);
  const { container } = await renderBusTask(Form, client, draft.key);
  if (picked.length > 0)
    await screen.findByRole("checkbox", { name: picked[0].fields[0].name });
  return { lib, dev, draft, container };
};

/** Opens the add message picker and waits for it to offer the named message. */
const openPicker = async (offered: string): Promise<HTMLElement> => {
  fireEvent.click(await findDialogTriggerByText("Add message"));
  const dialog = await findOpenDialog();
  await within(dialog).findByText(offered);
  return dialog;
};

const ALL_MESSAGES = [
  "Airspeed",
  "Polled",
  "Attitude",
  "Status",
  "Command",
  "Engine",
  "Wheel",
];

/** Expects the picker to offer exactly the named messages of the library. */
const expectOffered = (dialog: HTMLElement, offered: string[]) =>
  expect(
    ALL_MESSAGES.filter((name) => within(dialog).queryByText(name) != null),
  ).toEqual(offered);

/** @returns the device name as channel names hold it, spaces and dashes escaped. */
const escaped = (name: string): string => name.replace(/[ -]/g, "_");

describe("avionics bus tasks", () => {
  describe("message picker", () => {
    it("should offer an ARINC 429 task only its messages with no query", async () => {
      await open({ ...ARINC429_READ, messages: () => [] });
      expectOffered(await openPicker("Airspeed"), ["Airspeed"]);
    });

    it("should offer a MIL-STD-1553 task only MIL-STD-1553 messages", async () => {
      await open({ ...mil1553Read({}), messages: () => [] });
      expectOffered(await openPicker("Attitude"), [
        "Attitude",
        "Status",
        "Command",
        "Engine",
      ]);
    });
  });

  describe("deploy checks", () => {
    it("should show a query on its message and create no channels", async () => {
      const { dev, container } = await open({
        ...ARINC429_READ,
        messages: (lib) => [lib.polled],
      });
      await clickDeploy(container);
      await findMessageError(
        0,
        "Message Polled has a query, which ARINC 429 cannot send",
      );
      expect(await client.channels.retrieve([`${escaped(dev.name)}_raw`])).toHaveLength(
        0,
      );
    });

    it("should show a message of another medium on its message", async () => {
      const { container } = await open({
        ...ARINC429_READ,
        messages: (lib) => [lib.attitude],
      });
      await clickDeploy(container);
      await findMessageError(0, "Message Attitude has no ARINC 429 identifier");
    });

    it.each<[string, Draft, string]>([
      [
        "a bus controller reading a message with no period",
        { ...mil1553Read({ role: "bus_controller" }), messages: (l) => [l.status] },
        "Message Status needs a period for the bus controller to poll it",
      ],
      [
        "a bus controller reading a receive message",
        { ...mil1553Read({ role: "bus_controller" }), messages: (l) => [l.command] },
        "Message Command must be a transmit message for a bus controller to read",
      ],
      [
        "a bus controller sending a transmit message",
        { ...mil1553Write({ role: "bus_controller" }), messages: (l) => [l.attitude] },
        "Message Attitude must be a receive message for a bus controller to send",
      ],
      [
        "a remote terminal reading a transmit message",
        {
          ...mil1553Read({ role: "remote_terminal", terminals: [5] }),
          messages: (l) => [l.attitude],
        },
        "Message Attitude must be a receive message for a remote terminal to read",
      ],
      [
        "a remote terminal reading for a terminal it does not own",
        {
          ...mil1553Read({ role: "remote_terminal", terminals: [5] }),
          messages: (l) => [l.engine],
        },
        "Message Engine is for terminal 7, which the remote terminal does not own",
      ],
      [
        "a remote terminal answering with a receive message",
        {
          ...mil1553Write({ role: "remote_terminal", terminals: [5] }),
          messages: (l) => [l.command],
        },
        "Message Command must be a transmit message for a remote terminal to " +
          "answer with",
      ],
      [
        "a monitor sending",
        { ...mil1553Write({ role: "monitor" }), messages: (l) => [l.command] },
        "Message Command cannot be sent by a monitor",
      ],
    ])("should reject %s", async (_, draft, message) => {
      const { container } = await open(draft);
      await clickDeploy(container);
      await findMessageError(0, message);
    });

    it("should bind a role error to the message that fails it", async () => {
      const { container } = await open({
        ...mil1553Read({ role: "bus_controller" }),
        messages: (l) => [l.attitude, l.status],
      });
      await clickDeploy(container);
      fireEvent.click(await screen.findByText("Status"));
      await findMessageError(
        1,
        "Message Status needs a period for the bus controller to poll it",
      );
      expect(screen.queryByText(/Message Attitude/)).toBeNull();
    });
  });

  describe("channel names", () => {
    it("should name ARINC 429 read channels after the device and message", async () => {
      const { dev, draft, container } = await open({
        ...ARINC429_READ,
        messages: (lib) => [lib.airspeed],
      });
      const { config } = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        ARINC429.Task.READ_SCHEMAS,
      );
      const [msg] = config.messages;
      expect((await client.channels.retrieve(msg.index)).name).toBe(
        `${escaped(dev.name)}_Airspeed_time`,
      );
      expect((await client.channels.retrieve(msg.fields[0].channel)).name).toBe(
        `${escaped(dev.name)}_Airspeed_Speed`,
      );
      expect((await client.channels.retrieve(config.raw)).name).toBe(
        `${escaped(dev.name)}_raw`,
      );
    });

    it("should let a MIL-STD-1553 monitor read any terminal", async () => {
      const { dev, draft, container } = await open({
        ...mil1553Read({ role: "monitor" }),
        messages: (lib) => [lib.engine],
      });
      const { config } = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MIL1553.Task.READ_SCHEMAS,
      );
      expect(
        (await client.channels.retrieve(config.messages[0].fields[0].channel)).name,
      ).toBe(`${escaped(dev.name)}_Engine_Throttle`);
    });

    it("should name MIL-STD-1553 command channels", async () => {
      const { dev, draft, container } = await open({
        ...mil1553Write({ role: "bus_controller" }),
        messages: (lib) => [lib.command],
      });
      const { config } = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MIL1553.Task.WRITE_SCHEMAS,
      );
      const cmd = await client.channels.retrieve(config.messages[0].fields[0].channel);
      expect(cmd.name).toBe(`${escaped(dev.name)}_Command_Setpoint_cmd`);
      expect((await client.channels.retrieve(cmd.index)).name).toBe(
        `${escaped(dev.name)}_Command_cmd_time`,
      );
    });
  });
});
