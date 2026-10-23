// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
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

const createMessage = (fields: icd.BinaryField[]): icd.MessageEntry => ({
  key: uuid.create(),
  kind: "message",
  name: "Engine",
  payload: { format: "binary", fields },
});

const createStateEnum = (): icd.EnumEntry => ({
  key: uuid.create(),
  kind: "enum",
  name: "State",
  values: [
    { value: 0, name: "Off" },
    { value: 1, name: "On" },
  ],
});

const retrieveMessage = async (key: icd.Key): Promise<icd.MessageEntry> => {
  const [entry] = (await client.icds.retrieve({ key })).entries;
  if (entry.kind !== "message") throw new Error("expected a message entry");
  return entry;
};

describe("ICD editor", () => {
  it("should save an edited enum value to the Core", async () => {
    const ic = await createICD([
      {
        key: uuid.create(),
        kind: "enum",
        name: "State",
        values: [{ value: 0, name: "Off" }],
      },
    ]);
    await renderEditor(ic);
    await commit("Name", 1, "Idle");
    await save();
    await waitFor(async () => {
      const [entry] = (await client.icds.retrieve({ key: ic.key })).entries;
      if (entry.kind !== "enum") throw new Error("expected an enum entry");
      expect(entry.values).toEqual([{ value: 0, name: "Idle" }]);
    });
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
    await waitFor(() => expect(screen.queryByLabelText("Bitrate switch")).toBeNull());
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
    const delimiter = await screen.findByLabelText("Delimiter");
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

  it("should show a Core validation error on the enum value it names", async () => {
    const ic = await createICD([
      {
        key: uuid.create(),
        kind: "enum",
        name: "State",
        values: [
          { value: 0, name: "Off" },
          { value: 1, name: "On" },
        ],
      },
    ]);
    await renderEditor(ic);
    await commit("Value", 2, "0");
    await save();
    expect(await screen.findByText(/^Value 2: duplicate value 0/)).toBeTruthy();
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
    const ic = await createICD([createStateEnum()]);
    await renderEditor(ic);
    await commit("Value", 2, "0");
    expect(await screen.findByText(/^Value 2: duplicate value 0/)).toBeTruthy();
  });

  it("should clear the mark when an edit fixes the problem", async () => {
    const ic = await createICD([createStateEnum()]);
    await renderEditor(ic);
    await commit("Value", 2, "0");
    await screen.findByText(/^Value 2: duplicate value 0/);
    await commit("Value", 2, "2");
    await waitFor(() =>
      expect(screen.queryByText(/^Value 2: duplicate value 0/)).toBeNull(),
    );
  });

  it("should offer a reload when the ICD changed after it opened", async () => {
    const ic = await createICD([createStateEnum()]);
    await renderEditor(ic);
    await screen.findByLabelText("Name 1");
    const [state] = ic.entries;
    if (state.kind !== "enum") throw new Error("expected an enum entry");
    await client.icds.create({
      ...ic,
      entries: [{ ...state, values: [{ value: 0, name: "Stopped" }] }],
    });
    await commit("Name", 1, "Idle");
    await save();
    expect(
      await screen.findByText("This ICD changed after you opened it"),
    ).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Reload" }));
    await waitFor(() => expect(cell("Name", 1).value).toBe("Stopped"));
    expect(screen.queryByText("This ICD changed after you opened it")).toBeNull();
  });

  it("should show the entries of an imported file", async () => {
    const picker = interceptFilePicker();
    const ic = await createICD();
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: "Import" }));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("engine.csv", ENGINE_CSV)]);
    expect(await screen.findByText("Engine")).toBeTruthy();
    expect(cell("Name", 1).value).toBe("Rpm");
  });
});
