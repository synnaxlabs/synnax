// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
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

const renderEditor = async (ic: icd.ICD): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  const { panelKey, tabKey } = await createResourceTab(client, icd.ontologyID(ic.key));
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

const save = async (): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: "Save" }));
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
  multiplexValues: [],
});

const createJSONMessage = (fields: icd.JSONField[]): icd.Message => ({
  key: uuid.create(),
  name: "Reading",
  payload: { format: "json", fields },
});

const retrieveMessage = async (key: icd.Key): Promise<icd.Message> => {
  const [message] = (await client.icds.retrieve({ key })).messages;
  return message;
};

const retrieveNames = async (key: icd.Key): Promise<string[]> =>
  (await client.icds.retrieve({ key })).messages.map(({ name }) => name);

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
    fireEvent.click(screen.getByRole("button", { name: "Add message" }));
    await findMessageItem("Message 2");
    await save();
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
    await save();
    await waitFor(async () => expect(await retrieveNames(ic.key)).toEqual(["B"]));
  });

  it("should keep field keys when a field is edited", async () => {
    const fields = [createBinaryField("Rpm", 0), createBinaryField("Temp", 8)];
    const ic = await createICD([createMessage(fields)]);
    await renderEditor(ic);
    await commit("Start bit", 2, "16");
    await save();
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
    await save();
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
    await save();
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
    await save();
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

  it("should save the pointer of a JSON field", async () => {
    const ic = await createICD([createJSONMessage([createJSONField("Temp", "/temp")])]);
    await renderEditor(ic);
    await commit("Pointer", 1, "/sensors/0/temp");
    await save();
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload).toMatchObject({
        format: "json",
        fields: [{ name: "Temp", pointer: "/sensors/0/temp" }],
      }),
    );
  });

  it("should drop the scale when a JSON field stops being a number", async () => {
    const ic = await createICD([
      createJSONMessage([{ ...createJSONField("Time", "/ts"), scale: 0.1 }]),
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByText("Number"));
    fireEvent.click(await screen.findByText("Timestamp (ms)"));
    await save();
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { type: "timestamp", format: "unix_ms", scale: 1 },
      ]),
    );
  });

  it("should show a Core validation error on the field it names", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Scale", 2, "0");
    await save();
    expect(
      await screen.findByText(/^Field 2: scale must be finite and non-zero/),
    ).toBeTruthy();
  });

  it("should mark a problem shortly after an edit, with no save", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Scale", 2, "0");
    expect(
      await screen.findByText(/^Field 2: scale must be finite and non-zero/),
    ).toBeTruthy();
  });

  it("should mark a blank name shortly after an edit, with no save", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Name", 2, "");
    expect(
      await screen.findByText(
        /^Field 2: Too small: expected string to have >=1 characters/,
      ),
    ).toBeTruthy();
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

  it("should offer a reload when the ICD changed after it opened", async () => {
    const rpm = createBinaryField("Rpm", 0);
    const ic = await createICD([createMessage([rpm])]);
    await renderEditor(ic);
    await screen.findByLabelText("Name 1");
    const [engine] = ic.messages;
    await client.icds.create({
      ...ic,
      messages: [
        {
          ...engine,
          payload: { format: "binary", fields: [{ ...rpm, name: "Speed" }] },
        },
      ],
    });
    await commit("Name", 1, "Idle");
    await save();
    expect(
      await screen.findByText("This ICD changed after you opened it"),
    ).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Reload" }));
    await waitFor(() => expect(cell("Name", 1).value).toBe("Speed"));
    expect(screen.queryByText("This ICD changed after you opened it")).toBeNull();
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
