// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { library } from "@synnaxlabs/client";
import { Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Library } from "@/feature/library";
import { client, createLibrary, ENGINE_CSV } from "@/feature/library/testutil";
import { createResourceTab, primePanel } from "@/platform/panel/testutil";
import {
  createConsoleWrapper,
  fakePickedFile,
  interceptFilePicker,
  renderSuspended,
} from "@/testutil";

const Editor = Library.TABS[library.TYPE_ONTOLOGY_ID.type].Content;

const renderEditor = async (lib: library.Library): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  const { panelKey, tabKey } = await createResourceTab(
    client,
    library.ontologyID(lib.key),
  );
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

const createBinaryField = (name: string, startBit: number): library.Field => ({
  key: uuid.create(),
  encoding: "binary",
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

const createMessage = (fields: library.Field[]): library.Entry => ({
  key: uuid.create(),
  kind: "message",
  name: "Engine",
  format: "binary",
  delimiter: ",",
  fields,
});

describe("Library editor", () => {
  it("should save an edited enum value to the Core", async () => {
    const lib = await createLibrary([
      {
        key: uuid.create(),
        kind: "enum",
        name: "State",
        values: [{ value: 0, name: "Off" }],
      },
    ]);
    await renderEditor(lib);
    await commit("Name", 1, "Idle");
    await save();
    await waitFor(async () => {
      const [entry] = (await client.libraries.retrieve({ key: lib.key })).entries;
      if (entry.kind !== "enum") throw new Error("expected an enum entry");
      expect(entry.values).toEqual([{ value: 0, name: "Idle" }]);
    });
  });

  it("should keep field keys when a field is edited", async () => {
    const fields = [createBinaryField("Rpm", 0), createBinaryField("Temp", 8)];
    const lib = await createLibrary([createMessage(fields)]);
    await renderEditor(lib);
    await commit("Start bit", 2, "16");
    await save();
    await waitFor(async () => {
      const [entry] = (await client.libraries.retrieve({ key: lib.key })).entries;
      if (entry.kind !== "message") throw new Error("expected a message entry");
      expect(entry.fields.map(({ key }) => key)).toEqual(fields.map(({ key }) => key));
      expect(entry.fields[1]).toMatchObject({ name: "Temp", startBit: 16 });
    });
  });

  it("should keep the keys of the fields a removal leaves", async () => {
    const fields = [createBinaryField("Rpm", 0), createBinaryField("Temp", 8)];
    const lib = await createLibrary([createMessage(fields)]);
    await renderEditor(lib);
    await screen.findByLabelText("Name 1");
    const remove = cell("Name", 1)
      .closest("tr")
      ?.querySelector<HTMLButtonElement>("td:last-child button");
    if (remove == null) throw new Error("expected a remove button on row 1");
    fireEvent.click(remove);
    await save();
    await waitFor(async () => {
      const [entry] = (await client.libraries.retrieve({ key: lib.key })).entries;
      if (entry.kind !== "message") throw new Error("expected a message entry");
      expect(entry.fields.map(({ key, name }) => ({ key, name }))).toEqual([
        { key: fields[1].key, name: "Temp" },
      ]);
    });
  });

  it("should show a Core validation error on the enum value it names", async () => {
    const lib = await createLibrary([
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
    await renderEditor(lib);
    await commit("Value", 2, "0");
    await save();
    expect(await screen.findByText(/^Value 2: duplicate value 0/)).toBeTruthy();
  });

  it("should show a Core validation error on the field it names", async () => {
    const lib = await createLibrary([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(lib);
    await commit("Bit length", 2, "65");
    await save();
    expect(
      await screen.findByText(/^Field 2: bit_length must be between 1 and 64/),
    ).toBeTruthy();
  });

  it("should show the entries of an imported file", async () => {
    const picker = interceptFilePicker();
    const lib = await createLibrary();
    await renderEditor(lib);
    fireEvent.click(await screen.findByRole("button", { name: "Import" }));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("engine.csv", ENGINE_CSV)]);
    expect(await screen.findByText("Engine")).toBeTruthy();
    expect(cell("Name", 1).value).toBe("Rpm");
  });
});
