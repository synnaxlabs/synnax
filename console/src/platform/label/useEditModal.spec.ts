// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { mockBoundingClientRect } from "@synnaxlabs/pluto/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Label } from "@/platform/label";
import { openModal } from "@/platform/modals/testutil";
import { getIconButton, uniqueName } from "@/testutil";

const getAddButton = (): HTMLButtonElement => {
  const btn = document.querySelector<HTMLButtonElement>(".console-label__create");
  if (btn == null) throw new Error("create button not found");
  return btn;
};

const getCreateItem = (): HTMLElement => {
  const item = document.querySelector<HTMLElement>(
    ".console-label__list-item.console--create",
  );
  if (item == null) throw new Error("create item not found");
  return item;
};

const openEditModal = async () => {
  const client = createTestClient();
  return { ...(await openModal(Label.useEditModal, { client })), client };
};

describe("Label.useEditModal", () => {
  it("should reveal the create form when the add button is clicked", async () => {
    await openEditModal();
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Search labels...")).toBeTruthy(),
    );
    expect(screen.getByText("Edit")).toBeTruthy();
    expect(getCreateItem().className).toContain("pluto--hidden");
    fireEvent.click(getAddButton());
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
    fireEvent.click(getAddButton());
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

  it("should persist a new label to the cluster from the create form", async () => {
    const { client } = await openEditModal();
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Search labels...")).toBeTruthy(),
    );
    fireEvent.click(getAddButton());
    const name = uniqueName("label");
    const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Name");
    fireEvent.change(nameInput, { target: { value: name } });
    fireEvent.click(getIconButton(document.body, "check"));
    await waitFor(async () => {
      const found = await client.labels.retrieve({ names: [name] });
      expect(found.length).toBe(1);
    });
  });
});
