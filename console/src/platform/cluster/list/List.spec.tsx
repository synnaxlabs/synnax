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

import { Cluster } from "@/platform/cluster";
import { renderClusterUI } from "@/platform/cluster/testutil";
import { Session } from "@/session";
import { createCluster, createClusterState } from "@/session/cluster/testutil";
import { getBySelector, stubClipboardWriteText } from "@/testutil";

describe("cluster List", () => {
  it("should call onChange with the clicked cluster's key", async () => {
    const onChange = vi.fn();
    await renderClusterUI(
      <Cluster.List value="a" onChange={onChange} />,
      createClusterState(
        [createCluster("a", { name: "Alpha" }), createCluster("b", { name: "Bravo" })],
        "a",
      ),
    );
    fireEvent.click(await screen.findByText("Bravo"));
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    expect(onChange.mock.lastCall?.[0]).toBe("b");
  });

  it("should open the connect modal from the empty action", async () => {
    const { container } = await renderClusterUI(
      <Cluster.List value={undefined} onChange={vi.fn()} />,
      createClusterState([]),
    );
    // The footer create button carries the same label, so scope to the list body.
    const items = getBySelector<HTMLElement>(container, ".console-cluster-list__items");
    fireEvent.click(await within(items).findByText("Add a Core"));
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Synnax Core")).toBeTruthy(),
    );
  });

  it("should open the connect modal in edit mode from the context menu", async () => {
    await renderClusterUI(
      <Cluster.List value="a" onChange={vi.fn()} />,
      createClusterState([createCluster("a", { name: "Alpha" })], "a"),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Edit"));
    await waitFor(() => {
      const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");
      expect(nameInput.value).toEqual("Alpha");
    });
  });

  it("should copy a link to the cluster from the context menu", async () => {
    const writeText = stubClipboardWriteText();
    await renderClusterUI(
      <Cluster.List value="a" onChange={vi.fn()} />,
      createClusterState([createCluster("a", { name: "Alpha" })], "a"),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Copy link"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText.mock.calls[0][0]).toBe("synnax://cluster/a");
  });

  it("should remove a cluster and reselect a sibling from the context menu", async () => {
    const onChange = vi.fn();
    const { store } = await renderClusterUI(
      <Cluster.List value="a" onChange={onChange} />,
      createClusterState(
        [createCluster("a", { name: "Alpha" }), createCluster("b", { name: "Bravo" })],
        "a",
      ),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Remove"));
    expect(onChange).toHaveBeenCalledWith("b");
    expect(Session.Cluster.selectState(store.getState(), "a")).toBeUndefined();
    expect(Session.Cluster.selectState(store.getState(), "b")).toBeDefined();
  });

  it("should refresh the connection and adopt the server's cluster key from the context menu", async () => {
    const { store } = await renderClusterUI(
      <Cluster.List value="a" onChange={vi.fn()} />,
      createClusterState([createCluster("a", { name: "Alpha" })], "a"),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Refresh connection"));
    await waitFor(() =>
      expect(Session.Cluster.selectState(store.getState(), "a")).toBeUndefined(),
    );
    const clusters = Session.Cluster.selectMany(store.getState());
    expect(clusters).toHaveLength(1);
    expect(clusters[0].name).toBe("Alpha");
    expect(clusters[0].key).not.toBe("a");
  });

  it("should rename a cluster to a new name from the context menu", async () => {
    const { store } = await renderClusterUI(
      <Cluster.List value="a" onChange={vi.fn()} />,
      createClusterState([createCluster("a", { name: "Alpha" })], "a"),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = screen.getByText("Alpha");
    await waitFor(() => expect(editable.getAttribute("contenteditable")).toBe("true"));
    editable.innerText = "Beta";
    fireEvent.keyDown(editable, { key: "Enter" });
    await waitFor(() =>
      expect(Session.Cluster.selectState(store.getState(), "a")?.name).toBe("Beta"),
    );
  });

  it("should reject renaming a cluster to a name already in use", async () => {
    const { store } = await renderClusterUI(
      <Cluster.List value="a" onChange={vi.fn()} />,
      createClusterState(
        [createCluster("a", { name: "Alpha" }), createCluster("b", { name: "Bravo" })],
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
    expect(Session.Cluster.selectState(store.getState(), "a")?.name).toBe("Alpha");
    expect(Session.Cluster.selectState(store.getState(), "b")?.name).toBe("Bravo");
  });

  it("should open the connect modal from the header add button", async () => {
    await renderClusterUI(
      <Cluster.List value="a" onChange={vi.fn()} />,
      createClusterState([createCluster("a", { name: "Alpha" })], "a"),
    );
    const addButton = screen
      .getAllByRole("button")
      .find((b) => b.querySelector("svg[aria-label*='add']"));
    if (addButton == null) throw new Error("add button not found");
    fireEvent.click(addButton);
    await waitFor(() => {
      const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");
      expect(nameInput.value).toEqual("");
    });
  });
});
