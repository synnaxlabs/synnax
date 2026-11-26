// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { enumeration } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Enumeration as PEnumeration } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Enumeration } from "@/platform/enumeration";
import { renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

interface Opened {
  /** Settles when the modal closes, with the key of the new enumeration or null. */
  result: Promise<enumeration.Key | null>;
}

/** Opens the prompt that the provider gives. */
const openPrompt = async (params: PEnumeration.CreatePromptParams): Promise<Opened> => {
  const { result } = await renderModalOpener(PEnumeration.useCreatePrompt, [params], {
    client,
    provider: Enumeration.CreatePromptProvider,
  });
  const opened = result();
  if (opened == null) throw new Error("the prompt did not open");
  return { result: opened };
};

const nameField = async (): Promise<HTMLInputElement> =>
  await screen.findByLabelText<HTMLInputElement>("Enumeration name");

// commit types into a cell of the entries table, then blurs it to commit the text.
const commit = (label: string, value: string): void => {
  const cell = screen.getByLabelText(label);
  fireEvent.focus(cell);
  fireEvent.change(cell, { target: { value } });
  fireEvent.blur(cell);
};

const escape = (el: HTMLElement): void => {
  fireEvent.keyDown(el, { key: "Escape", code: "Escape" });
  fireEvent.keyUp(el, { key: "Escape", code: "Escape" });
};

describe("Enumeration.CreatePromptProvider", () => {
  it("should create the enumeration with the typed name and the entries", async () => {
    const entries = [{ key: id.create(), name: "Off", value: 0, abnormal: false }];
    const { result } = await openPrompt({ name: "Pump", entries });
    const field = await nameField();
    expect(field.value).toBe("Pump");
    const name = `Pump ${id.create()}`;
    fireEvent.change(field, { target: { value: name } });
    fireEvent.click(screen.getByRole("button", { name: /^Create/ }));
    const key = await result;
    if (key == null) throw new Error("expected the key of the new enumeration");
    const e = await client.enumerations.retrieve(key);
    expect(e.name).toBe(name);
    expect(e.entries.map((en) => [en.name, en.value])).toEqual([["Off", 0]]);
    await waitFor(() => expect(screen.queryByLabelText("Enumeration name")).toBeNull());
  });

  it("should create the entries typed into the table", async () => {
    const { result } = await openPrompt({
      name: `Pump ${id.create()}`,
      entries: PEnumeration.createEntries(),
    });
    await nameField();
    commit("Name 1", "Off");
    fireEvent.click(screen.getByRole("button", { name: "Add entry" }));
    commit("Name 2", "On");
    fireEvent.click(screen.getByRole("button", { name: /^Create/ }));
    const key = await result;
    if (key == null) throw new Error("expected the key of the new enumeration");
    const e = await client.enumerations.retrieve(key);
    expect(e.entries.map((en) => [en.name, en.value])).toEqual([
      ["Off", 0],
      ["On", 1],
    ]);
  });

  it("should ask before an Escape drops a typed entry", async () => {
    await openPrompt({ name: "Pump", entries: PEnumeration.createEntries() });
    const field = await nameField();
    commit("Name 1", "Off");
    escape(field);
    expect(await screen.findByText("Discard unsaved edits?")).toBeTruthy();
    expect(screen.getByLabelText("Enumeration name")).toBeTruthy();
  });

  it("should close on one Escape from the name field and create nothing", async () => {
    const { result } = await openPrompt({ name: "Pump", entries: [] });
    const field = await nameField();
    escape(field);
    await waitFor(() => expect(screen.queryByLabelText("Enumeration name")).toBeNull());
    expect(await result).toBeNull();
  });

  it("should show an error at the name field on a Create with no name", async () => {
    await openPrompt({ name: "", entries: [] });
    const field = await nameField();
    fireEvent.change(field, { target: { value: " " } });
    fireEvent.click(screen.getByRole("button", { name: /^Create/ }));
    expect(await screen.findByText("Name is required")).toBeTruthy();
    fireEvent.change(field, { target: { value: "Pump" } });
    expect(screen.queryByText("Name is required")).toBeNull();
  });
});
