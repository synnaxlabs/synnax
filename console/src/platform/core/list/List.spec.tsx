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
import { getCoreRow, renderCoreUI } from "@/platform/core/testutil";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import { getBySelector, getIconButton, stubClipboardWriteText } from "@/testutil";

const ALPHA = createCore("Alpha", { clusterKey: "cluster-alpha" });
const BRAVO = createCore("Bravo", { port: 9099, clusterKey: undefined });

describe("Core List", () => {
  it("should call onChange with the clicked Core's key", async () => {
    const onChange = vi.fn();
    await renderCoreUI(
      <Core.List value={ALPHA.key} onChange={onChange} />,
      createCoreState([ALPHA, BRAVO], ALPHA.key),
    );
    fireEvent.click(await screen.findByText("Bravo"));
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    expect(onChange.mock.lastCall?.[0]).toBe(BRAVO.key);
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
      <Core.List value={ALPHA.key} onChange={vi.fn()} />,
      createCoreState([ALPHA], ALPHA.key),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Edit"));
    await waitFor(() => {
      const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");
      expect(nameInput.value).toEqual("Alpha");
    });
  });

  // The link names the cluster, not the record, so it opens on a machine whose own
  // record for that cluster carries a different key.
  it("should copy a link to the Core's cluster from the context menu", async () => {
    const writeText = stubClipboardWriteText();
    await renderCoreUI(
      <Core.List value={ALPHA.key} onChange={vi.fn()} />,
      createCoreState([ALPHA], ALPHA.key),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Copy link"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText.mock.calls[0][0]).toBe("synnax://cluster/cluster-alpha");
  });

  it("should copy no link for a Core that has never connected", async () => {
    const writeText = stubClipboardWriteText();
    await renderCoreUI(
      <Core.List value={BRAVO.key} onChange={vi.fn()} />,
      createCoreState([BRAVO], BRAVO.key),
    );
    fireEvent.contextMenu(await screen.findByText("Bravo"));
    fireEvent.click(await screen.findByText("Copy link"));
    await waitFor(() => expect(screen.queryByText("Copy link")).toBeNull());
    expect(writeText).not.toHaveBeenCalled();
  });

  // The row is itself a select target, so the copy button must not switch Cores.
  it("should copy a Core's address without selecting its row", async () => {
    const writeText = stubClipboardWriteText();
    const onChange = vi.fn();
    const { container } = await renderCoreUI(
      <Core.List value={ALPHA.key} onChange={onChange} />,
      createCoreState([ALPHA, BRAVO], ALPHA.key),
    );
    fireEvent.click(getIconButton(getCoreRow(container, "Bravo"), "copy"));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("localhost:9099"));
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should remove a Core and reselect a sibling from the context menu", async () => {
    const onChange = vi.fn();
    const { store } = await renderCoreUI(
      <Core.List value={ALPHA.key} onChange={onChange} />,
      createCoreState([ALPHA, BRAVO], ALPHA.key),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Remove"));
    expect(onChange).toHaveBeenCalledWith(BRAVO.key);
    expect(Session.Core.selectState(store.getState(), ALPHA.key)).toBeUndefined();
    expect(Session.Core.selectState(store.getState(), BRAVO.key)).toBeDefined();
  });

  it("should leave the Core's key alone when refreshing the connection", async () => {
    const { store } = await renderCoreUI(
      <Core.List value={ALPHA.key} onChange={vi.fn()} />,
      createCoreState([ALPHA], ALPHA.key),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Refresh connection"));
    await waitFor(() => expect(screen.queryByText("Refresh connection")).toBeNull());
    // The address is the key, so a connection tells the session nothing new about it.
    const cores = Session.Core.selectMany(store.getState());
    expect(cores).toHaveLength(1);
    expect(cores[0].key).toBe(ALPHA.key);
  });

  it("should rename a Core to a new name from the context menu", async () => {
    const { store } = await renderCoreUI(
      <Core.List value={ALPHA.key} onChange={vi.fn()} />,
      createCoreState([ALPHA], ALPHA.key),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = screen.getByText("Alpha");
    await waitFor(() => expect(editable.getAttribute("contenteditable")).toBe("true"));
    editable.innerText = "Beta";
    fireEvent.keyDown(editable, { key: "Enter" });
    await waitFor(() =>
      expect(Session.Core.selectState(store.getState(), ALPHA.key)?.name).toBe("Beta"),
    );
  });

  // A name is a label, not an identity, so two Cores may share one.
  it("should accept renaming a Core to a name another Core already has", async () => {
    const { store } = await renderCoreUI(
      <Core.List value={ALPHA.key} onChange={vi.fn()} />,
      createCoreState([ALPHA, BRAVO], ALPHA.key),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = screen.getByText("Alpha");
    await waitFor(() => expect(editable.getAttribute("contenteditable")).toBe("true"));
    editable.innerText = "Bravo";
    fireEvent.keyDown(editable, { key: "Enter" });
    await waitFor(() =>
      expect(Session.Core.selectState(store.getState(), ALPHA.key)?.name).toBe("Bravo"),
    );
    expect(Session.Core.selectState(store.getState(), BRAVO.key)?.name).toBe("Bravo");
  });

  it("should accept renaming a Core to the name it already has", async () => {
    const { store } = await renderCoreUI(
      <Core.List value={ALPHA.key} onChange={vi.fn()} />,
      createCoreState([ALPHA, BRAVO], ALPHA.key),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = screen.getByText("Alpha");
    await waitFor(() => expect(editable.getAttribute("contenteditable")).toBe("true"));
    editable.innerText = "Alpha ";
    fireEvent.keyDown(editable, { key: "Enter" });
    await waitFor(() => expect(editable.getAttribute("contenteditable")).toBe("false"));
    expect(Session.Core.selectState(store.getState(), ALPHA.key)?.name).toBe("Alpha");
  });

  it("should open the connect modal from the header add button", async () => {
    await renderCoreUI(
      <Core.List value={ALPHA.key} onChange={vi.fn()} />,
      createCoreState([ALPHA], ALPHA.key),
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
