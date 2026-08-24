// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  findButton,
  findDismissButton,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { useConfirmDelete } from "@/platform/modals/useConfirmDelete";

interface OpenConfirmParams {
  items: { name: string } | { name: string }[];
  type?: string;
  title?: string;
  description?: string;
}

const openConfirm = async ({
  items,
  type = "Channel",
  title,
  description,
}: OpenConfirmParams): Promise<ModalOpenerHandle<Promise<boolean>>> =>
  await renderModalOpener(
    () => useConfirmDelete({ type, title, description }),
    [items],
  );

// The header splits a dotted title into one breadcrumb segment per part, so the whole
// string never appears as a single node.
const headerText = (): string =>
  document.querySelector(".console-modal__header .pluto-breadcrumb")?.textContent ?? "";

describe("useConfirmDelete", () => {
  it("should prompt with the resource name and resolve true when confirmed", async () => {
    const handle = await openConfirm({ items: { name: "Sensor A" } });
    await screen.findByText("Are you sure you want to delete Sensor A?");
    fireEvent.click(findButton("Delete"));
    await expect(handle.result()).resolves.toBe(true);
    await waitFor(() =>
      expect(
        screen.queryByText("Are you sure you want to delete Sensor A?"),
      ).toBeNull(),
    );
  });

  it("should prompt with a pluralized count and resolve false when cancelled", async () => {
    const handle = await openConfirm({
      items: [{ name: "a" }, { name: "b" }, { name: "c" }],
    });
    await screen.findByText("Are you sure you want to delete 3 channels?");
    fireEvent.click(findButton("Cancel"));
    await expect(handle.result()).resolves.toBe(false);
  });

  it("should show the default irreversibility description", async () => {
    await openConfirm({ items: { name: "Sensor A" } });
    await screen.findByText("This action cannot be undone.");
  });

  it("should use a custom description when provided", async () => {
    await openConfirm({
      items: { name: "Sensor A" },
      description: "Custom warning.",
    });
    await screen.findByText("Custom warning.");
    expect(screen.queryByText("This action cannot be undone.")).toBeNull();
  });

  it("should resolve false when dismissed without an answer", async () => {
    const handle = await openConfirm({ items: { name: "Sensor A" } });
    await screen.findByText("Are you sure you want to delete Sensor A?");
    fireEvent.click(findDismissButton());
    await expect(handle.result()).resolves.toBe(false);
  });

  it("should title the modal after the resource type by default", async () => {
    await openConfirm({ items: { name: "Sensor A" }, type: "Range" });
    await screen.findByText("Range");
    expect(headerText()).toEqual("RangeDelete");
  });

  it("should use a custom title when provided", async () => {
    await openConfirm({ items: { name: "Sensor A" }, title: "Range.Remove" });
    await screen.findByText("Remove");
    expect(headerText()).toEqual("RangeRemove");
  });

  it("should pluralize an irregular resource type", async () => {
    await openConfirm({ items: [{ name: "a" }, { name: "b" }], type: "Policy" });
    await screen.findByText("Are you sure you want to delete 2 policies?");
  });
});
