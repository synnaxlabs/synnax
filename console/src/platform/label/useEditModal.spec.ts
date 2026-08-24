// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { mockBoundingClientRect } from "@synnaxlabs/pluto/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Label } from "@/platform/label";
import { openModal } from "@/platform/modals/testutil";
import {
  createTestClientWithGrants,
  getBySelector,
  getIconButton,
  type Grants,
  queryIconButton,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

// The add button renders only once the create-permission query resolves.
const findAddButton = async (): Promise<HTMLButtonElement> =>
  await waitFor(() =>
    getBySelector<HTMLButtonElement>(document.body, ".console-label__create"),
  );

const queryAddButton = (): HTMLButtonElement | null =>
  document.body.querySelector<HTMLButtonElement>(".console-label__create");

const getCreateItem = (): HTMLElement => {
  const item = document.querySelector<HTMLElement>(
    ".console-label__list-item.console--create",
  );
  if (item == null) throw new Error("create item not found");
  return item;
};

const openEditModal = async (as: Client = client) =>
  await openModal(Label.useEditModal, { client: as });

// Narrows the paginated list down to the one label the test created and returns its
// row, the only handle on that label's own delete button.
const findLabelRow = async (name: string): Promise<HTMLElement> => {
  const search = await screen.findByPlaceholderText("Search labels...");
  fireEvent.change(search, { target: { value: name } });
  // The search input carries the term as its own value, so the row must be picked
  // out of every input displaying the name.
  return await waitFor(() => {
    const row = screen
      .getAllByDisplayValue(name)
      .map((el) => el.closest<HTMLElement>(".console-label__list-item"))
      .find((el) => el != null);
    if (row == null) throw new Error(`no list item for label ${name}`);
    return row;
  });
};

describe("Label.useEditModal", () => {
  it("should reveal the create form when the add button is clicked", async () => {
    await openEditModal();
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Search labels...")).toBeTruthy(),
    );
    expect(screen.getByText("Edit")).toBeTruthy();
    expect(getCreateItem().className).toContain("pluto--hidden");
    fireEvent.click(await findAddButton());
    await waitFor(() => {
      const item = getCreateItem();
      expect(item.className).toContain("pluto--visible");
      expect(item.className).not.toContain("pluto--hidden");
    });
  });

  it("should keep the create form open while the color picker is open", async () => {
    await openEditModal();
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Search labels...")).toBeTruthy(),
    );
    fireEvent.click(await findAddButton());
    await waitFor(() => expect(getCreateItem().className).toContain("pluto--visible"));
    const swatch = document.querySelector<HTMLElement>(".pluto-color-swatch");
    if (swatch == null) throw new Error("color swatch not found");
    fireEvent.click(swatch);
    const picker = await waitFor(() => {
      const el = document.querySelector<HTMLElement>(".sketch-picker");
      if (el == null) throw new Error("color picker did not open");
      return el;
    });
    // Every element shares one rect in jsdom, so the row and the viewport need
    // their own for the click to land outside the row.
    getCreateItem().getBoundingClientRect = mockBoundingClientRect(0, 0, 50, 50);
    document.documentElement.getBoundingClientRect = mockBoundingClientRect(
      0,
      0,
      1000,
      1000,
    );
    fireEvent.pointerDown(picker, { clientX: 200, clientY: 200 });
    expect(getCreateItem().className).toContain("pluto--visible");
  });

  it("should persist a new label to the Core from the create form", async () => {
    await openEditModal();
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Search labels...")).toBeTruthy(),
    );
    fireEvent.click(await findAddButton());
    const name = uniqueName("label");
    // Every listed label renders its own name input, so the create form's has to be
    // reached through its row.
    const createItem = getCreateItem();
    const nameInput = within(createItem).getByPlaceholderText<HTMLInputElement>("Name");
    fireEvent.change(nameInput, { target: { value: name } });
    fireEvent.click(getIconButton(createItem, "check"));
    await waitFor(async () => {
      const found = await client.labels.retrieve({ names: [name] });
      expect(found.length).toBe(1);
    });
  });
});

describe("Label.useEditModal permissions", () => {
  const createSubject = async (grants: Grants) =>
    await createTestClientWithGrants(client, {
      ...grants,
      retrieve: [label.TYPE_ONTOLOGY_ID],
    });

  const createLabel = async () =>
    await client.labels.create({ name: uniqueName("label"), color: "#0000FF" });

  it("should withhold the delete button from a subject who cannot delete labels", async () => {
    const { name } = await createLabel();
    await openEditModal(await createSubject({ create: [label.TYPE_ONTOLOGY_ID] }));
    // The create grant resolving proves the permission queries answered, so the
    // missing delete button is the gate and not a check still in flight.
    await findAddButton();
    expect(queryIconButton(await findLabelRow(name), "delete")).toBeNull();
  });

  it("should offer the delete button to a subject who may delete labels", async () => {
    const { name } = await createLabel();
    await openEditModal(await createSubject({ delete: [label.TYPE_ONTOLOGY_ID] }));
    const row = await findLabelRow(name);
    await waitFor(() => expect(queryIconButton(row, "delete")).toBeTruthy());
    expect(queryAddButton()).toBeNull();
  });
});
