// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeRange } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import { createActiveState } from "@/platform/project/testutil";
import { Range } from "@/platform/range";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createCluster } from "@/session/cluster/testutil";
import { createConsoleWrapper, stubClipboardWriteText, uniqueName } from "@/testutil";

const client = createTestClient();

const createRange = async (): Promise<ranger.Range> => await createTestRange(client);

const buttonWithIcon = (label: string): HTMLElement => {
  const button = screen
    .getAllByRole("button")
    .find((b) => b.querySelector(`svg[aria-label*='${label}']`));
  if (button == null) throw new Error(`button with icon ${label} not found`);
  return button;
};

const renderDetails = async (rangeKey: string) => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const result = render(
    <>
      <Range.Details rangeKey={rangeKey} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { ...result, store };
};

describe("Range.Details", () => {
  it("should rename the range on the cluster when the name field is edited", async () => {
    const range = await createRange();
    await renderDetails(range.key);
    const nameInput = await screen.findByDisplayValue(range.name, {});
    const next = uniqueRangeName("renamed");
    fireEvent.change(nameInput, { target: { value: next } });
    fireEvent.blur(nameInput);
    await waitFor(async () => {
      const retrieved = await client.ranges.retrieve(range.key);
      expect(retrieved.name).toEqual(next);
    });
  });

  it("should open the parent range as a tab when its button is clicked", async () => {
    const parent = await createRange();
    const child = await client.ranges.create({
      name: uniqueRangeName("child"),
      timeRange: new TimeRange(1, 2),
      parent,
    });
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    render(
      <>
        <Range.Details rangeKey={child.key} />
        <Modals.Stack />
      </>,
      { wrapper },
    );
    fireEvent.click(await screen.findByText(parent.name, {}));
    await waitFor(async () => {
      const panelKey = Session.Panel.selectSelected(store.getState());
      expect(panelKey).toBeDefined();
      const doc = await client.panels.retrieve(panelKey as string);
      const tab = panel.findTabByResource(doc.root, ranger.ontologyID(parent.key));
      expect(tab).not.toBeNull();
    });
  });

  it("should open the CSV download modal scoped to the range", async () => {
    const range = await createRange();
    await renderDetails(range.key);
    await screen.findByDisplayValue(range.name, {});
    fireEvent.click(buttonWithIcon("csv"));
    await waitFor(() =>
      expect(document.body.textContent).toContain(`Download data for ${range.name}`),
    );
  });

  it("should copy a link to the range to the clipboard", async () => {
    const writeText = stubClipboardWriteText();
    const range = await createRange();
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: {
        [Session.Cluster.SLICE_NAME]: {
          version: 0,
          selected: "local",
          clusters: {
            local: createCluster("local", { name: "Local" }),
          },
        },
      },
    });
    render(<Range.Details rangeKey={range.key} />, { wrapper });
    await screen.findByDisplayValue(range.name, {});
    fireEvent.click(buttonWithIcon("link"));
    await waitFor(() => expect(writeText).toHaveBeenCalled());
    expect(writeText.mock.calls[0][0]).toContain(range.key);
  });

  it("should copy Python retrieval code to the clipboard", async () => {
    const writeText = stubClipboardWriteText();
    const range = await createRange();
    await renderDetails(range.key);
    await screen.findByDisplayValue(range.name, {});
    fireEvent.click(buttonWithIcon("python"));
    await waitFor(() => expect(writeText).toHaveBeenCalled());
    const copied = writeText.mock.calls[0][0];
    expect(copied).toContain("client.ranges.retrieve");
    expect(copied).toContain(range.key);
  });

  it("should copy TypeScript retrieval code to the clipboard", async () => {
    const writeText = stubClipboardWriteText();
    const range = await createRange();
    await renderDetails(range.key);
    await screen.findByDisplayValue(range.name, {});
    fireEvent.click(buttonWithIcon("typescript"));
    await waitFor(() => expect(writeText).toHaveBeenCalled());
    const copied = writeText.mock.calls[0][0];
    expect(copied).toContain("client.ranges.retrieve");
    expect(copied).toContain(range.key);
  });
});
