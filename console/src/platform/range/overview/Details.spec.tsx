// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, ranger, type Synnax as Client } from "@synnaxlabs/client";
import {
  createTestClient,
  RoleClients,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { TimeRange } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { searchAndClickLabelOption } from "@/platform/label/testutil";
import { Modals } from "@/platform/modals";
import { createActiveState } from "@/platform/project/testutil";
import { Range } from "@/platform/range";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createCore } from "@/session/core/testutil";
import {
  createConsoleWrapper,
  findTagCloseButton,
  getIconButton,
  renderSuspended,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const createRange = async (): Promise<ranger.Range> => await createTestRange(client);

const buttonWithIcon = (label: string): HTMLElement => {
  const button = screen
    .getAllByRole("button")
    .find((b) => b.querySelector(`svg.lyra-icon--${label}`));
  if (button == null) throw new Error(`button with icon ${label} not found`);
  return button;
};

const renderDetails = async (rangeKey: string, as: Client = client) => {
  const { wrapper, store } = await createConsoleWrapper({ client: as });
  const result = await renderSuspended(
    <>
      <Range.Details rangeKey={rangeKey} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { ...result, store };
};

describe("Range.Details", () => {
  it("should rename the range on the Core when the name field is edited", async () => {
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

  it("should show a label another client adds and add one beside it", async () => {
    const range = await createRange();
    const [added, remote] = await client.labels.create([
      { name: uniqueName("added"), color: "#FF0000" },
      { name: uniqueName("remote"), color: "#00FF00" },
    ]);
    await renderDetails(range.key);
    await screen.findByDisplayValue(range.name, {});
    await waitForStreamLive(client.connection);
    await createTestClient().labels.label(range.ontologyID, [remote.key]);
    await screen.findByText(remote.name);
    fireEvent.click(getIconButton(screen.getByLabelText("Labels"), "add"));
    await searchAndClickLabelOption(added.name);
    await waitFor(async () => {
      const labels = await client.labels.retrieve({ for: range.ontologyID });
      expect(labels.map(({ key }) => key).sort()).toEqual(
        [added.key, remote.key].sort(),
      );
    });
  });

  it("should remove a label when its tag is closed", async () => {
    const range = await createRange();
    const [removed, kept] = await client.labels.create([
      { name: uniqueName("removed"), color: "#FF0000" },
      { name: uniqueName("kept"), color: "#00FF00" },
    ]);
    await client.labels.label(range.ontologyID, [removed.key, kept.key]);
    await renderDetails(range.key);
    await screen.findByText(removed.name);
    fireEvent.click(findTagCloseButton(removed.name));
    await waitFor(async () => {
      const labels = await client.labels.retrieve({ for: range.ontologyID });
      expect(labels.map(({ key }) => key)).toEqual([kept.key]);
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
    });
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    await renderSuspended(
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
        [Session.Core.SLICE_NAME]: {
          version: 0,
          selected: "local",
          cores: {
            local: createCore("local", { name: "Local" }),
          },
        },
      },
    });
    await renderSuspended(<Range.Details rangeKey={range.key} />, { wrapper });
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

  describe("without permission to write the range", () => {
    it("should put the name field in preview rather than leaving it editable", async () => {
      const range = await createRange();
      const viewer = await roles.get("Viewer");
      await renderDetails(range.key, viewer);
      await waitFor(() => {
        const name = screen.getByText(range.name);
        expect(name.closest(".lyra-btn--preview")).not.toBeNull();
      });
      expect(screen.queryByDisplayValue(range.name)).toBeNull();
    });
  });
});
