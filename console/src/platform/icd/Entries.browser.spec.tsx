// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, serial, type Synnax, type task } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { ICD } from "@/platform/icd";
import {
  createEntry,
  expectRefused,
  pickNewMessage,
  typeText,
} from "@/platform/icd/testutil";
import { Task } from "@/platform/task";
import {
  awaitEditableForm,
  findDetailsPane,
  findListButton,
  renderTaskFormTab,
} from "@/platform/task/testutil";

// The Core keeps the config of a known task type only, so the entries are serial ones.
const schemas = {
  type: z.literal("serial_read"),
  config: serial.readConfigZ,
  statusData: z.object({ running: z.boolean() }).nullish(),
};

const client = createTestClient();

interface Rendered {
  key: task.Key;
  /** The name of the device of the entries, unique to the test. */
  device: string;
}

interface RenderEntriesOptions {
  /** The client of the user that renders the form, when not the one that saves it. */
  as?: Synnax;
  entries?: icd.ReadEntry[];
  device?: string;
}

/** Renders the entries of a new draft task. */
const renderEntries = async ({
  as,
  entries = [],
  device = uniqueName("port"),
}: RenderEntriesOptions = {}): Promise<Rendered> => {
  const Form = () => <ICD.Entries device={{ name: device }} />;
  const Renderer = Task.wrapForm<typeof schemas>({
    Form,
    schemas,
    deployConfigZ: schemas.config,
    type: "serial_read",
    onConfigure: async (_client, config) => [config, 0],
  });
  const draft = await client.tasks.create(
    {
      name: uniqueName("task"),
      type: "serial_read",
      config: schemas.config.parse({ entries }),
    },
    schemas,
  );
  await renderTaskFormTab(Renderer, { client, as, taskKey: draft.key });
  // A preview never turns editable.
  if (as == null) await awaitEditableForm();
  return { key: draft.key, device };
};

const retrieveEntries = async (key: task.Key): Promise<icd.ReadEntry[]> =>
  (await client.tasks.retrieve({ key, schemas })).config.entries;

/** @returns the entries of the task, once it holds count of them. */
const awaitEntries = async (key: task.Key, count: number): Promise<icd.ReadEntry[]> =>
  await waitFor(async () => {
    const entries = await retrieveEntries(key);
    expect(entries).toHaveLength(count);
    return entries;
  });

/** Adds an entry, and waits for the tree to show it under the name of its ICD. */
const addEntry = async (icdName: string): Promise<HTMLElement> => {
  const count = screen.queryAllByRole("treeitem", { name: new RegExp(icdName) }).length;
  fireEvent.click(await findListButton("Add entry"));
  return await waitFor(() => {
    const items = screen.getAllByRole("treeitem", { name: new RegExp(icdName) });
    expect(items).toHaveLength(count + 1);
    return items[count];
  });
};

const QUERY_ERROR = "Query must be pairs of hex digits, such as 01 03";

describe("ICD.Entries", () => {
  it("should add an entry that reads a new binary message of an ICD named after the device", async () => {
    const { key, device } = await renderEntries();
    const item = await addEntry(device);
    within(item).getByText("BINARY");
    const [entry] = await awaitEntries(key, 1);
    const { name, messages } = await client.icds.retrieve(entry.icd);
    expect(name).toBe(device);
    expect(messages.map((m) => [m.name, m.payload.format])).toEqual([
      ["Message 1", "binary"],
    ]);
    expect(entry.messages).toEqual([{ message: messages[0].key, bindings: [] }]);
  });

  it("should add the message of a later entry to the ICD of the first", async () => {
    const { key, device } = await renderEntries();
    await addEntry(device);
    await addEntry(device);
    const [first, second] = await awaitEntries(key, 2);
    expect(second.icd).toBe(first.icd);
    await waitFor(async () =>
      expect(
        (await client.icds.retrieve(first.icd)).messages.map(({ name }) => name),
      ).toEqual(["Message 1", "Message 2"]),
    );
  });

  it("should add a message to an entry from its details", async () => {
    const { key, device } = await renderEntries();
    await addEntry(device);
    fireEvent.click(await screen.findByRole("button", { name: "Add message" }));
    await waitFor(async () => {
      const [entry] = await retrieveEntries(key);
      const { messages } = await client.icds.retrieve(entry.icd);
      expect(messages.map(({ name, payload }) => [name, payload.format])).toEqual([
        ["Message 1", "binary"],
        ["Message 2", "binary"],
      ]);
      expect(entry.messages.map(({ message }) => message)).toEqual(
        messages.map(({ key }) => key),
      );
    });
  });

  it("should make an entry read a new message of another format", async () => {
    const { key, device } = await renderEntries();
    const item = await addEntry(device);
    await pickNewMessage("Message 1", "New text message");
    await within(item).findByText("TEXT");
    await waitFor(async () => {
      const [entry] = await retrieveEntries(key);
      // The form made the binary message and it has no fields, so it leaves the ICD.
      const { messages } = await client.icds.retrieve(entry.icd);
      expect(messages.map(({ payload }) => payload.format)).toEqual(["text"]);
      expect(entry.messages).toMatchObject([{ message: messages[0].key }]);
    });
  });

  it("should add an entry that reads a message of the format the other entries read", async () => {
    const { key, device } = await renderEntries();
    await addEntry(device);
    await pickNewMessage("Message 1", "New text message");
    await waitFor(async () => {
      const [entry] = await retrieveEntries(key);
      const { messages } = await client.icds.retrieve(entry.icd);
      expect(messages.map(({ payload }) => payload.format)).toEqual(["text"]);
    });
    const item = await addEntry(device);
    await within(item).findByText("TEXT");
    const [, entry] = await awaitEntries(key, 2);
    const { messages } = await client.icds.retrieve(entry.icd);
    expect(
      messages.find(({ key }) => key === entry.messages[0].message)?.payload.format,
    ).toBe("text");
  });

  it("should add an entry in the format of the enabled entries before a disabled one", async () => {
    const text = icd.textPayloadZ.parse({ format: "text" });
    const binary = icd.binaryPayloadZ.parse({ format: "binary" });
    const { key } = await renderEntries({
      entries: [
        await createEntry(client, text, { disabled: true }),
        await createEntry(client, binary),
      ],
    });
    fireEvent.click(await findListButton("Add entry"));
    const [, , entry] = await awaitEntries(key, 3);
    const { messages } = await client.icds.retrieve(entry.icd);
    expect(
      messages.find(({ key }) => key === entry.messages[0].message)?.payload.format,
    ).toBe("binary");
  });

  it("should add a message in the format of the other entries to an entry that reads no message its ICD holds", async () => {
    const reading = await createEntry(
      client,
      icd.binaryPayloadZ.parse({ format: "binary" }),
    );
    const gone = icd.readEntryZ.parse({ icd: reading.icd, messages: [] });
    const { key } = await renderEntries({ entries: [reading, gone] });
    const { name } = await client.icds.retrieve(reading.icd);
    const items = await screen.findAllByRole("treeitem", { name: new RegExp(name) });
    fireEvent.click(items[items.length - 1]);
    fireEvent.click(await screen.findByRole("button", { name: "Add message" }));
    await waitFor(async () => {
      const [, entry] = await retrieveEntries(key);
      expect(entry.messages).toHaveLength(1);
      const { messages } = await client.icds.retrieve(entry.icd);
      const added = messages.find(({ key }) => key === entry.messages[0].message);
      expect(added?.payload.format).toBe("binary");
    });
  });

  it("should switch an entry off", async () => {
    const { key, device } = await renderEntries();
    const item = await addEntry(device);
    fireEvent.click(within(item).getByRole("checkbox", { name: "Enabled" }));
    await waitFor(async () =>
      expect((await retrieveEntries(key))[0].disabled).toBe(true),
    );
  });

  it("should offer no add in a preview", async () => {
    await renderEntries({ as: await createTestClientWithRole(client, "Viewer") });
    const details = await findDetailsPane();
    await within(details).findByText("No entry selected");
    expect(within(details).queryByRole("button", { name: "Add entry" })).toBeNull();
  });

  describe("poll", () => {
    /** Renders a task with one entry, selected. */
    const renderEntry = async (): Promise<task.Key> => {
      const { key, device } = await renderEntries();
      await addEntry(device);
      await awaitEntries(key, 1);
      return key;
    };

    const retrievePoll = async (key: task.Key): Promise<icd.Poll | undefined> =>
      (await retrieveEntries(key))[0].poll;

    it("should poll with a query written in hexadecimal, at the default rate", async () => {
      const key = await renderEntry();
      expect(screen.queryByLabelText(/^Rate/)).toBeNull();
      const query = await typeText(/^Query/, "0x01 0302");
      await waitFor(async () =>
        expect(await retrievePoll(key)).toEqual({
          query: new Uint8Array([0x01, 0x03, 0x02]),
          rate: 1,
        }),
      );
      expect(query.value).toBe("01 03 02");
    });

    it("should keep the rate of a poll whose query changes", async () => {
      const key = await renderEntry();
      await typeText(/^Query/, "01");
      await typeText(/^Rate/, "4");
      await waitFor(async () => expect((await retrievePoll(key))?.rate).toBe(4));
      await typeText(/^Query/, "02");
      await waitFor(async () =>
        expect(await retrievePoll(key)).toEqual({
          query: new Uint8Array([0x02]),
          rate: 4,
        }),
      );
    });

    it("should keep query text that does not parse, with its error", async () => {
      const key = await renderEntry();
      await expectRefused(await typeText(/^Query/, "01 0"), "01 0", QUERY_ERROR);
      expect(await retrievePoll(key)).toBeUndefined();
    });

    it("should stop polling when the query is cleared", async () => {
      const key = await renderEntry();
      await typeText(/^Query/, "01");
      await waitFor(async () => expect(await retrievePoll(key)).toBeDefined());
      await typeText(/^Query/, "");
      await waitFor(async () => expect(await retrievePoll(key)).toBeUndefined());
      expect(screen.queryByLabelText(/^Rate/)).toBeNull();
    });
  });
});
