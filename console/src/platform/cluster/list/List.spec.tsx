// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Cluster } from "@/platform/cluster";
import { cluster, clusterState, renderClusterUI } from "@/platform/cluster/testutil";
import { Session } from "@/session";

describe("cluster List", () => {
  it("should render every cluster's name and address", async () => {
    await renderClusterUI(
      <Cluster.List value="a" onChange={vi.fn()} />,
      clusterState(
        [
          cluster("a", { name: "Alpha", host: "alpha.host", port: 1111 }),
          cluster("b", { name: "Bravo", host: "bravo.host", port: 2222 }),
        ],
        "a",
      ),
    );
    await waitFor(() => expect(screen.getByText("Alpha")).toBeTruthy());
    expect(screen.getByText("Bravo")).toBeTruthy();
    expect(screen.getByText("alpha.host:1111")).toBeTruthy();
    expect(screen.getByText("bravo.host:2222")).toBeTruthy();
  });

  it("should show the empty action when there are no clusters", async () => {
    await renderClusterUI(
      <Cluster.List value={undefined} onChange={vi.fn()} />,
      clusterState([]),
    );
    await waitFor(() => expect(screen.getByText("No Cores added.")).toBeTruthy());
    expect(screen.getByText("Add a Core")).toBeTruthy();
  });

  it("should open the connect modal from the empty action", async () => {
    await renderClusterUI(
      <Cluster.List value={undefined} onChange={vi.fn()} />,
      clusterState([]),
    );
    fireEvent.click(await screen.findByText("Add a Core"));
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Synnax Core")).toBeTruthy(),
    );
  });

  it("should open the connect modal in edit mode from the context menu", async () => {
    await renderClusterUI(
      <Cluster.List value="a" onChange={vi.fn()} />,
      clusterState([cluster("a", { name: "Alpha" })], "a"),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Edit"));
    await waitFor(() => {
      const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");
      expect(nameInput.value).toEqual("Alpha");
    });
  });

  it("should copy a link to the cluster from the context menu", async () => {
    const writeText = vi.fn(async (_text: string) => {});
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    await renderClusterUI(
      <Cluster.List value="a" onChange={vi.fn()} />,
      clusterState([cluster("a", { name: "Alpha" })], "a"),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Copy link"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText.mock.calls[0][0]).toContain("a");
  });

  it("should remove a cluster and reselect a sibling from the context menu", async () => {
    const onChange = vi.fn();
    const { store } = await renderClusterUI(
      <Cluster.List value="a" onChange={onChange} />,
      clusterState(
        [cluster("a", { name: "Alpha" }), cluster("b", { name: "Bravo" })],
        "a",
      ),
    );
    fireEvent.contextMenu(await screen.findByText("Alpha"));
    fireEvent.click(await screen.findByText("Remove"));
    expect(onChange).toHaveBeenCalledWith("b");
    expect(Session.Cluster.selectState(store.getState(), "a")).toBeUndefined();
    expect(Session.Cluster.selectState(store.getState(), "b")).toBeDefined();
  });

  it("should rename a cluster to a new name from the context menu", async () => {
    const { store } = await renderClusterUI(
      <Cluster.List value="a" onChange={vi.fn()} />,
      clusterState([cluster("a", { name: "Alpha" })], "a"),
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
      clusterState(
        [cluster("a", { name: "Alpha" }), cluster("b", { name: "Bravo" })],
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
      clusterState([cluster("a", { name: "Alpha" })], "a"),
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
