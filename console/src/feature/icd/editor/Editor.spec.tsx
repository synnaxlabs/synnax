// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, type Synnax } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
  spyOnSend,
} from "@synnaxlabs/client/testutil";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Editor } from "@/feature/icd/editor/Editor";
import { client, createICD, ENGINE_CSV } from "@/feature/icd/testutil";
import { createResourceTab, primePanel } from "@/platform/panel/testutil";
import {
  createConsoleWrapper,
  fakePickedFile,
  interceptFilePicker,
  renderSuspended,
} from "@/testutil";

const renderEditor = async (ic: icd.ICD, as: Synnax = client): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  const { panelKey, tabKey } = await createResourceTab(client, icd.resourceID(ic.key));
  await primePanel(wrapper, panelKey);
  await renderSuspended(
    <PlutoPanel.Scope.Provider value={panelKey}>
      <PlutoPanel.TabScope.Provider value={tabKey}>
        <Editor />
      </PlutoPanel.TabScope.Provider>
    </PlutoPanel.Scope.Provider>,
    { wrapper },
  );
};

const cell = (column: string, row: number): HTMLInputElement =>
  screen.getByLabelText(`${column} ${row}`);

const commit = async (column: string, row: number, value: string): Promise<void> => {
  await screen.findByLabelText(`${column} ${row}`);
  const input = cell(column, row);
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

const createBinaryField = (name: string, startBit: number): icd.BinaryField => ({
  key: uuid.create(),
  name,
  startBit,
  bitLength: 8,
  byteOrder: "little_endian",
  signed: false,
  float: false,
  scale: 1,
  offset: 0,
  units: "",
  multiplexValues: [],
});

const createMessage = (
  fields: icd.BinaryField[],
  name: string = "Engine",
): icd.Message => ({
  key: uuid.create(),
  name,
  payload: { format: "binary", fields },
});

const createJSONField = (name: string, pointer: string): icd.JSONField => ({
  key: uuid.create(),
  name,
  type: "number",
  pointer,
  scale: 1,
  offset: 0,
  units: "",
});

const createJSONMessage = (fields: icd.JSONField[]): icd.Message => ({
  key: uuid.create(),
  name: "Reading",
  payload: { format: "json", fields },
});

const createTextMessage = (): icd.Message => ({
  key: uuid.create(),
  name: "Line",
  payload: { format: "text", delimiter: ",", prefix: "", fields: [] },
});

const retrieveMessage = async (key: icd.Key): Promise<icd.Message> => {
  const [message] = (await client.icds.retrieve({ key })).messages;
  return message;
};

const retrieveNames = async (key: icd.Key): Promise<string[]> =>
  (await client.icds.retrieve({ key })).messages.map(({ name }) => name);

/** Edits the ICD through a client that the editor does not use. */
const dispatchElsewhere = async (key: icd.Key, action: icd.Action): Promise<void> => {
  const other = createTestClient();
  await other.icds.retrieve({ key });
  expect(await other.icds.dispatch(key, action)).toBe(true);
};

const findMessageItem = async (name: string): Promise<HTMLElement> => {
  const texts = await screen.findAllByText(name);
  const item = texts
    .map((t) => t.closest<HTMLElement>(`.${Menu.CONTEXT_TARGET}`))
    .find((el) => el != null);
  if (item == null) throw new Error(`no message in the list is named ${name}`);
  return item;
};

const removeFromContextMenu = async (item: HTMLElement): Promise<void> => {
  await act(async () => {
    fireEvent.contextMenu(item);
  });
  fireEvent.click(await screen.findByText("Remove"));
};

describe("ICD editor", () => {
  it("should add a message with a unique name", async () => {
    const ic = await createICD([createMessage([], "Message 1")]);
    await renderEditor(ic);
    await findMessageItem("Message 1");
    fireEvent.click(await screen.findByRole("button", { name: "Add message" }));
    await findMessageItem("Message 2");
    await waitFor(async () =>
      expect(await retrieveNames(ic.key)).toEqual(["Message 1", "Message 2"]),
    );
  });

  it("should remove the selected messages from the context menu", async () => {
    const ic = await createICD([
      createMessage([], "A"),
      createMessage([], "B"),
      createMessage([], "C"),
    ]);
    await renderEditor(ic);
    await findMessageItem("A");
    const c = await findMessageItem("C");
    fireEvent.keyDown(document.body, { code: "Control" });
    fireEvent.click(c);
    fireEvent.keyUp(document.body, { code: "Control" });
    await removeFromContextMenu(c);
    await waitFor(() => expect(screen.queryByText("A")).toBeNull());
    expect(screen.queryByText("C")).toBeNull();
    await waitFor(async () => expect(await retrieveNames(ic.key)).toEqual(["B"]));
  });

  it("should keep field keys when a field is edited", async () => {
    const fields = [createBinaryField("Rpm", 0), createBinaryField("Temp", 8)];
    const ic = await createICD([createMessage(fields)]);
    await renderEditor(ic);
    await commit("Start bit", 2, "16");
    await waitFor(async () => {
      const { payload } = await retrieveMessage(ic.key);
      expect(payload.fields.map(({ key }) => key)).toEqual(
        fields.map(({ key }) => key),
      );
      expect(payload.fields[1]).toMatchObject({ name: "Temp", startBit: 16 });
    });
  });

  it("should keep the keys of the fields a removal leaves", async () => {
    const fields = [createBinaryField("Rpm", 0), createBinaryField("Temp", 8)];
    const ic = await createICD([createMessage(fields)]);
    await renderEditor(ic);
    await screen.findByLabelText("Name 1");
    const remove = cell("Name", 1)
      .closest("tr")
      ?.querySelector<HTMLButtonElement>("td:last-child button");
    if (remove == null) throw new Error("expected a remove button on row 1");
    fireEvent.click(remove);
    await waitFor(async () => {
      const { payload } = await retrieveMessage(ic.key);
      expect(payload.fields.map(({ key, name }) => ({ key, name }))).toEqual([
        { key: fields[1].key, name: "Temp" },
      ]);
    });
  });

  it("should clear the bitrate switch when CAN FD turns off", async () => {
    const ic = await createICD([
      {
        ...createMessage([]),
        payload: {
          format: "binary",
          fields: [createBinaryField("Rpm", 0)],
          identifier: {
            type: "can",
            id: 0x100,
            extended: false,
            fd: true,
            bitrateSwitched: true,
          },
        },
      },
    ]);
    await renderEditor(ic);
    const brs = await screen.findByLabelText<HTMLInputElement>("Bitrate switch");
    expect(brs.checked).toBe(true);
    fireEvent.click(screen.getByLabelText("CAN FD"));
    await waitFor(() => expect(brs.checked).toBe(false));
    expect(brs.disabled).toBe(true);
    await waitFor(async () => {
      expect((await retrieveMessage(ic.key)).payload).toMatchObject({
        identifier: { fd: false, bitrateSwitched: false },
      });
    });
  });

  it("should replace the payload when the format changes", async () => {
    const ic = await createICD([createMessage([createBinaryField("Rpm", 0)])]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: "Text" }));
    const delimiter = await screen.findByLabelText(/^Delimiter/);
    fireEvent.change(delimiter, { target: { value: ";" } });
    await waitFor(async () => {
      const { payload } = await retrieveMessage(ic.key);
      expect(payload).toEqual({
        format: "text",
        delimiter: ";",
        prefix: "",
        fields: [],
      });
    });
  });

  it("should store the pointer of a JSON field", async () => {
    const ic = await createICD([createJSONMessage([createJSONField("Temp", "/temp")])]);
    await renderEditor(ic);
    await commit("Pointer", 1, "/sensors/0/temp");
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload).toMatchObject({
        format: "json",
        fields: [{ name: "Temp", pointer: "/sensors/0/temp" }],
      }),
    );
  });

  it("should add the JSON fields a paste adds as numbers", async () => {
    const ic = await createICD([createJSONMessage([createJSONField("Temp", "/temp")])]);
    await renderEditor(ic);
    const input = await screen.findByLabelText("Name 1");
    fireEvent.focus(input);
    fireEvent.paste(input, {
      clipboardData: { getData: () => "Temp\t/temp\nSpeed\t/speed" },
    });
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { name: "Temp", pointer: "/temp", type: "number" },
        { name: "Speed", pointer: "/speed", type: "number", scale: 1 },
      ]),
    );
  });

  it("should give each field a paste adds its own name", async () => {
    const ic = await createICD([createJSONMessage([createJSONField("Temp", "/temp")])]);
    await renderEditor(ic);
    const input = await screen.findByLabelText("Pointer 1");
    fireEvent.focus(input);
    fireEvent.paste(input, { clipboardData: { getData: () => "/a\n/b\n/c" } });
    await waitFor(async () =>
      expect(
        (await retrieveMessage(ic.key)).payload.fields.map(({ name }) => name),
      ).toEqual(["Temp", "Field 2", "Field 3"]),
    );
  });

  it("should drop the scale when a JSON field stops being a number", async () => {
    const ic = await createICD([
      createJSONMessage([{ ...createJSONField("Time", "/ts"), scale: 0.1 }]),
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByText("Number"));
    fireEvent.click(await screen.findByText("Timestamp (ms)"));
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { type: "timestamp", format: "unix_ms", scale: 1 },
      ]),
    );
  });

  it("should store an edit that leaves a problem", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Scale", 2, "0");
    expect(
      await screen.findByText(/^Field 2: scale must be finite and non-zero/),
    ).toBeTruthy();
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields[1].scale).toBe(0),
    );
  });

  it("should mark a problem shortly after an edit", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Scale", 2, "0");
    expect(
      await screen.findByText(/^Field 2: scale must be finite and non-zero/),
    ).toBeTruthy();
  });

  it("should mark a blank name shortly after an edit", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Name", 2, "");
    expect(await screen.findByText(/^Field 2: name is required/)).toBeTruthy();
  });

  it("should clear the mark when an edit fixes the problem", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Scale", 2, "0");
    await screen.findByText(/^Field 2: scale must be finite and non-zero/);
    await commit("Scale", 2, "2");
    await waitFor(() =>
      expect(screen.queryByText(/^Field 2: scale must be finite/)).toBeNull(),
    );
  });

  it("should show an edit made elsewhere", async () => {
    const rpm = createBinaryField("Rpm", 0);
    const ic = await createICD([createMessage([rpm])]);
    await renderEditor(ic);
    await screen.findByLabelText("Name 1");
    const [engine] = ic.messages;
    await dispatchElsewhere(
      ic.key,
      icd.set({
        path: ["messages", engine.key, "payload", "fields", rpm.key, "name"],
        value: "Speed",
      }),
    );
    await waitFor(() => expect(cell("Name", 1).value).toBe("Speed"));
  });

  it("should send a cell edit as an edit to its field", async () => {
    const fields = [createBinaryField("Rpm", 0), createBinaryField("Temp", 8)];
    const ic = await createICD([createMessage(fields)]);
    await renderEditor(ic);
    const [engine] = ic.messages;
    await screen.findByLabelText("Start bit 2");
    const send = spyOnSend(client);
    await commit("Start bit", 2, "16");
    await waitFor(() =>
      expect(
        send.mock.calls.find(([target]) => target === "/icd/dispatch")?.[1],
      ).toMatchObject({
        actions: [
          {
            type: "set",
            set: {
              path: [
                "messages",
                engine.key,
                "payload",
                "fields",
                fields[1].key,
                "start_bit",
              ],
              value: 16,
            },
          },
        ],
      }),
    );
  });

  it("should show a viewer the ICD without the controls that edit it", async () => {
    const ic = await createICD([createMessage([createBinaryField("Mode", 0)])]);
    await renderEditor(ic, await createTestClientWithRole(client, "Viewer"));
    expect(await screen.findByText("Mode")).toBeTruthy();
    expect(screen.getByText("Binary")).toBeTruthy();
    expect(screen.queryByText("JSON")).toBeNull();
    expect(screen.queryByRole("button", { name: "Add row" })).toBeNull();
    expect(screen.queryByText("Add message")).toBeNull();
    expect(screen.queryByRole("button", { name: "Import" })).toBeNull();
  });

  it("should add a tagged field before it has a tag", async () => {
    const ic = await createICD([createTextMessage()]);
    await renderEditor(ic);
    const [, addTagged] = await screen.findAllByRole("button", { name: "Add row" });
    fireEvent.click(addTagged);
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { encoding: "tagged", name: "Field 1", tag: "" },
      ]),
    );
    expect(await screen.findByText(/^Field 1: tag is required/)).toBeTruthy();
  });

  it("should offer a field identifier only when the message has a field", async () => {
    const ic = await createICD([createMessage([])]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: /^Identifier/ }));
    await screen.findByText("CAN");
    expect(screen.queryByText("Field value")).toBeNull();
  });

  it("should offer a field identifier when the message has a field", async () => {
    const ic = await createICD([createMessage([createBinaryField("Mode", 0)])]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: /^Identifier/ }));
    expect(await screen.findByText("Field value")).toBeTruthy();
  });

  it("should show the messages of an imported file", async () => {
    const picker = interceptFilePicker();
    const ic = await createICD();
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: "Import" }));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("engine.csv", ENGINE_CSV)]);
    await findMessageItem("Engine");
    expect(cell("Name", 1).value).toBe("Rpm");
  });
});
