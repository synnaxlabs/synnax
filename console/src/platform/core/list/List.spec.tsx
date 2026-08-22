// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Core } from "@/platform/core";
import { renderCoreUI } from "@/platform/core/testutil";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import { getBySelector, stubClipboardWriteText } from "@/testutil";

describe("Core List", () => {
  it("should call onChange with the clicked Core's key", async () => {
    const onChange = vi.fn();
    await renderCoreUI(
      <Core.List value="a" onChange={onChange} />,
      createCoreState(
        [createCore("a", { name: "Alpha" }), createCore("b", { name: "Bravo" })],
        "a",
      ),
    );
    fireEvent.click(await screen.findByText("Bravo"));
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    expect(onChange.mock.lastCall?.[0]).toBe("b");
  });

  it("should open the connect modal from the empty action", async () => {
    const { container } = await renderCoreUI(
      <Core.List value={undefined} onChange={vi.fn()} />,
      createCoreState([]),
    );
    // The footer create button carries the same label, so scope to the list body.
    const items = getBySelector<HTMLElement>(container, ".console-core-list__items");
    fireEvent.click(await within(items).findByText("Add Core"));
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Synnax Core")).toBeTruthy(),
    );
  });

  it("should open the connect modal in edit mode from the context menu", async () => {
    await renderCoreUI(
      <Core.List value="a" onChange={vi.fn()} />,
      createCoreState([createCore("a", { name: "Alpha" })], "a"),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Edit"));
    await waitFor(() => {
      const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");
      expect(nameInput.value).toEqual("Alpha");
    });
  });

  it("should copy a link to the Core from the context menu", async () => {
    const writeText = stubClipboardWriteText();
    await renderCoreUI(
      <Core.List value="a" onChange={vi.fn()} />,
      createCoreState([createCore("a", { name: "Alpha" })], "a"),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Copy link"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText.mock.calls[0][0]).toBe("synnax://Core/a");
  });

  it("should remove a Core and reselect a sibling from the context menu", async () => {
    const onChange = vi.fn();
    const { store } = await renderCoreUI(
      <Core.List value="a" onChange={onChange} />,
      createCoreState(
        [createCore("a", { name: "Alpha" }), createCore("b", { name: "Bravo" })],
        "a",
      ),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Remove"));
    expect(onChange).toHaveBeenCalledWith("b");
    expect(Session.Core.selectState(store.getState(), "a")).toBeUndefined();
    expect(Session.Core.selectState(store.getState(), "b")).toBeDefined();
  });

  it("should refresh the connection and adopt the server's Core key from the context menu", async () => {
    const { store } = await renderCoreUI(
      <Core.List value="a" onChange={vi.fn()} />,
      createCoreState([createCore("a", { name: "Alpha" })], "a"),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Refresh connection"));
    await waitFor(() =>
      expect(Session.Core.selectState(store.getState(), "a")).toBeUndefined(),
    );
    const cores = Session.Core.selectMany(store.getState());
    expect(cores).toHaveLength(1);
    expect(cores[0].name).toBe("Alpha");
    expect(cores[0].key).not.toBe("a");
  });

  it("should rename a Core to a new name from the context menu", async () => {
    const { store } = await renderCoreUI(
      <Core.List value="a" onChange={vi.fn()} />,
      createCoreState([createCore("a", { name: "Alpha" })], "a"),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = screen.getByText("Alpha");
    await waitFor(() => expect(editable.getAttribute("contenteditable")).toBe("true"));
    editable.innerText = "Beta";
    fireEvent.keyDown(editable, { key: "Enter" });
    await waitFor(() =>
      expect(Session.Core.selectState(store.getState(), "a")?.name).toBe("Beta"),
    );
  });

  it("should reject renaming a Core to a name already in use", async () => {
    const { store } = await renderCoreUI(
      <Core.List value="a" onChange={vi.fn()} />,
      createCoreState(
        [createCore("a", { name: "Alpha" }), createCore("b", { name: "Bravo" })],
        "a",
      ),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = screen.getByText("Alpha");
    await waitFor(() => expect(editable.getAttribute("contenteditable")).toBe("true"));
    editable.innerText = "Bravo";
    fireEvent.keyDown(editable, { key: "Enter" });
    await waitFor(() => expect(editable.getAttribute("contenteditable")).toBe("false"));
    expect(Session.Core.selectState(store.getState(), "a")?.name).toBe("Alpha");
    expect(Session.Core.selectState(store.getState(), "b")?.name).toBe("Bravo");
  });

  it("should open the connect modal from the header add button", async () => {
    await renderCoreUI(
      <Core.List value="a" onChange={vi.fn()} />,
      createCoreState([createCore("a", { name: "Alpha" })], "a"),
    );
    const addButton = screen
      .getAllByRole("button")
      .find((b) => b.querySelector("svg.pluto-icon--add"));
    if (addButton == null) throw new Error("add button not found");
    fireEvent.click(addButton);
    await waitFor(() => {
      const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");
      expect(nameInput.value).toEqual("");
    });
  });
});
