// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { type Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import {
  findButton,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { Range } from "@/platform/range";
import { createOffPageTestRange, createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";

const client = createTestClient();

interface OpenModalOptions {
  client?: Synnax | null;
  analytics?: Analytics.Sink;
}

const openModal = async (
  params?: Range.CreateModalParams,
  { client: c = null, analytics }: OpenModalOptions = {},
): Promise<ModalOpenerHandle<Promise<ranger.Key | null>>> => {
  const handle = await renderModalOpener(Range.useCreateModal, [params ?? {}], {
    client: c,
    analytics,
  });
  await waitFor(() => expect(findButton("Create")).toBeTruthy());
  return handle;
};

const isOpen = (): boolean => screen.queryAllByText("Create").length > 0;

const clickWhenEnabled = async (text: string): Promise<void> => {
  await waitFor(() => expect(findButton(text).className).not.toContain("disabled"));
  fireEvent.click(findButton(text));
};

describe("Range.useCreateModal", () => {
  it("should disable Create when there is no connected client", async () => {
    await openModal();
    expect(findButton("Create").className).toContain("lyra--disabled");
  });

  it("should prefill the form from the initial params", async () => {
    await openModal({ name: "Prefilled Range", timeRange: { start: 1000, end: 2000 } });
    expect(screen.getByDisplayValue("Prefilled Range")).toBeTruthy();
  });

  it("should keep the modal open when the name is empty", async () => {
    await openModal({ timeRange: { start: 1000, end: 2000 } }, { client });
    await clickWhenEnabled("Create");
    expect(isOpen()).toBe(true);
  });

  it("should create the range on the Core without adding it to the favorites", async () => {
    const name = uniqueName("persisted");
    const { store } = await openModal(
      { name, timeRange: { start: 1, end: 2 } },
      { client },
    );
    await clickWhenEnabled("Create");
    await waitFor(() => expect(isOpen()).toBe(false));
    await waitFor(async () => {
      const retrieved = await client.ranges.retrieve(name);
      expect(retrieved.name).toEqual(name);
    });
    expect(store.getState()[Session.Range.SLICE_NAME].ranges).toEqual([]);
  });

  it("should resolve with the key of the range it creates", async () => {
    const name = uniqueName("prompted");
    const { result } = await openModal(
      { name, timeRange: { start: 1, end: 2 } },
      { client },
    );
    await clickWhenEnabled("Create");
    const key = await result();
    expect(key).toEqual((await client.ranges.retrieve(name)).key);
  });

  it("should report the range it saved to the Core", async () => {
    const analytics = createTestSink();
    await openModal(
      { name: uniqueName("reported"), timeRange: { start: 1, end: 2 } },
      { client, analytics },
    );
    await clickWhenEnabled("Create");
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("range_created", {}),
    );
  });

  it("should close on save after the prefilled parent is cleared", async () => {
    const parent = await createOffPageTestRange(client);
    const name = uniqueName("orphaned");
    await openModal(
      {
        name,
        timeRange: { start: 1000, end: 2000 },
        parent: parent.key,
      },
      { client },
    );
    fireEvent.click(await screen.findByLabelText(`Clear ${parent.name}`));
    await waitFor(() => expect(screen.getByText("Parent range")).toBeTruthy());
    await clickWhenEnabled("Create");
    await waitFor(() => expect(isOpen()).toBe(false));
    const created = await client.ranges.retrieve(name);
    expect(await client.ranges.retrieveParent(created.key)).toBeNull();
  });

  it("should attach the parent and labels", async () => {
    const parent = await createTestRange(client);
    const label = await client.labels.create({
      name: uniqueName("label"),
      color: "#E774D0",
    });
    const childName = uniqueName("child");
    await openModal(
      {
        name: childName,
        timeRange: { start: 1, end: 2 },
        parent: parent.key,
        labels: [label.key],
      },
      { client },
    );
    await clickWhenEnabled("Create");
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
