// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  findButton,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { Range } from "@/platform/range";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import { Session } from "@/session";

const client = createTestClient();

interface OpenModalOptions {
  client?: Synnax | null;
}

const openModal = async (
  params?: Range.CreateModalParams,
  { client: c = null }: OpenModalOptions = {},
): Promise<ModalOpenerHandle<void>> => {
  const handle = await renderModalOpener(Range.useCreateModal, [params ?? {}], {
    client: c,
  });
  await waitFor(() => expect(screen.getByText("Save Locally")).toBeTruthy());
  return handle;
};

const clickWhenEnabled = async (text: string): Promise<void> => {
  await waitFor(() => expect(findButton(text).className).not.toContain("disabled"));
  fireEvent.click(findButton(text));
};

describe("Range.useCreateModal", () => {
  it("should disable Save to Synnax when there is no connected client", async () => {
    await openModal();
    expect(findButton("Save to Synnax").className).toContain("pluto--disabled");
  });

  it("should prefill the form from the initial params", async () => {
    await openModal({ name: "Prefilled Range", timeRange: { start: 1000, end: 2000 } });
    expect(screen.getByDisplayValue("Prefilled Range")).toBeTruthy();
  });

  it("should add a non-persisted range to the slice and close on Save Locally", async () => {
    const { store } = await openModal({
      name: "Local Range",
      timeRange: { start: 1000, end: 2000 },
    });
    await clickWhenEnabled("Save Locally");
    await waitFor(() => expect(screen.queryByText("Save Locally")).toBeNull());
    const ranges = Session.Range.selectMultiple(store.getState());
    const created = ranges.find((r) => r.name === "Local Range");
    expect(created).toBeDefined();
    expect(created?.persisted).toBe(false);
    expect(created?.variant).toBe("static");
  });

  it("should keep the modal open and add nothing when the name is empty", async () => {
    const { store } = await openModal({ timeRange: { start: 1000, end: 2000 } });
    const before = Session.Range.selectMultiple(store.getState()).length;
    await clickWhenEnabled("Save Locally");
    expect(screen.getByText("Save Locally")).toBeTruthy();
    expect(Session.Range.selectMultiple(store.getState()).length).toBe(before);
  });

  it("should persist the range to the cluster and favorite it on Save to Synnax", async () => {
    const name = uniqueRangeName("persisted");
    const { store } = await openModal(
      { name, timeRange: { start: 1, end: 2 } },
      { client },
    );
    await clickWhenEnabled("Save to Synnax");
    await waitFor(() => {
      const created = Session.Range.selectMultiple(store.getState()).find(
        (r) => r.name === name,
      );
      expect(created).toBeDefined();
      expect(created?.persisted).toBe(true);
    });
    await waitFor(() => expect(screen.queryByText("Save Locally")).toBeNull());
    await waitFor(async () => {
      const retrieved = await client.ranges.retrieve(name);
      expect(retrieved.name).toEqual(name);
    });
  });

  it("should prefill from and update the existing range when rangeKey is provided", async () => {
    const existing = await createTestRange(client);
    await openModal({ rangeKey: existing.key }, { client });
    await waitFor(() => expect(screen.getByDisplayValue(existing.name)).toBeTruthy());
    const renamed = uniqueRangeName("renamed");
    fireEvent.change(screen.getByDisplayValue(existing.name), {
      target: { value: renamed },
    });
    await clickWhenEnabled("Save to Synnax");
    await waitFor(() => expect(screen.queryByText("Save Locally")).toBeNull());
    await waitFor(async () => {
      const updated = await client.ranges.retrieve(existing.key);
      expect(updated.name).toEqual(renamed);
    });
  });

  it("should attach the parent and labels when saving to Synnax", async () => {
    const parent = await createTestRange(client);
    const label = await client.labels.create({
      name: uniqueRangeName("label"),
      color: "#E774D0",
    });
    const childName = uniqueRangeName("child");
    await openModal(
      {
        name: childName,
        timeRange: { start: 1, end: 2 },
        parent: parent.key,
        labels: [label.key],
      },
      { client },
    );
    await clickWhenEnabled("Save to Synnax");
    let childKey = "";
    await waitFor(async () => {
      childKey = (await client.ranges.retrieve(childName)).key;
    });
    await waitFor(async () => {
      const retrievedParent = await client.ranges.retrieveParent(childKey);
      expect(retrievedParent?.key).toEqual(parent.key);
    });
    await waitFor(async () => {
      const created = await client.ranges.retrieve(childName);
      const labels = await created.retrieveLabels();
      expect(labels.map((l) => l.key)).toContain(label.key);
    });
  });
});
