// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, NotFoundError, type Synnax } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { color } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Label } from "@/feature/label";
import { Modals } from "@/platform/modals";
import {
  awaitGranted,
  awaitTextEditingElement,
  clickAndSettle,
  commitTextEdit,
  createConsoleWrapper,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const createLabel = async (name = uniqueName("label")): Promise<label.Label> =>
  await client.labels.create({ name, color: "#0000FF" });

const renderLabels = async (as: Synnax = client): Promise<void> => {
  const { wrapper: Console } = await createConsoleWrapper({ client: as });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      {children}
      <Modals.Stack />
    </Console>
  );
  await act(async () => {
    render(<Label.Settings />, { wrapper });
  });
};

// The Core holds labels from other tests, so each test searches until only its own
// labels are listed.
const search = async (term: string, count = 1): Promise<void> => {
  fireEvent.change(await screen.findByPlaceholderText("Search labels"), {
    target: { value: term },
  });
  await waitFor(() => expect(screen.getAllByRole("option")).toHaveLength(count));
};

const openMenu = async (name: string): Promise<void> => {
  await act(async () => {
    fireEvent.contextMenu(await screen.findByText(name));
  });
};

const deleted = async (key: label.Key): Promise<boolean> => {
  try {
    await client.labels.retrieve(key);
    return false;
  } catch (e) {
    return NotFoundError.matches(e);
  }
};

describe("Label.Settings", () => {
  it("should create a label from Create label", async () => {
    await renderLabels();
    await clickAndSettle("Create label");
    const name = uniqueName("label");
    fireEvent.change(await screen.findByPlaceholderText("Label name"), {
      target: { value: name },
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Create" }));
    });
    await expect
      .poll(async () => (await client.labels.retrieve({ name: [name] })).length)
      .toBe(1);
  });

  it("should rename a label from its menu", async () => {
    const l = await createLabel();
    await renderLabels();
    await search(l.name);
    await openMenu(l.name);
    await clickAndSettle("Rename");
    const name = uniqueName("renamed");
    commitTextEdit(await awaitTextEditingElement(), name);
    await expect
      .poll(async () => (await client.labels.retrieve(l.key)).name)
      .toBe(name);
  });

  it("should recolor a label from its swatch", async () => {
    const l = await createLabel();
    await renderLabels();
    await search(l.name);
    const line = screen.getByRole("option", { name: l.name });
    fireEvent.click(within(line).getByRole("button", { name: "Color" }));
    fireEvent.change(await screen.findByLabelText("Hex"), {
      target: { value: "123456" },
    });
    await act(async () => {
      fireEvent.keyDown(screen.getByLabelText("Hex"), { code: "Escape" });
    });
    await expect
      .poll(async () => color.hex((await client.labels.retrieve(l.key)).color))
      .toBe("#123456");
  });

  it("should delete every selected label at once after confirming", async () => {
    const prefix = uniqueName("bulk");
    const a = await createLabel(`${prefix}-a`);
    const b = await createLabel(`${prefix}-b`);
    await renderLabels();
    await search(prefix, 2);
    screen
      .getAllByRole("checkbox", { name: "Select" })
      .forEach((box) => fireEvent.click(box));
    await openMenu(a.name);
    expect(screen.queryByText("Rename")).toBeNull();
    await clickAndSettle("Delete");
    await act(async () => {
      fireEvent.click(await screen.findByRole("button", { name: "Delete" }));
    });
    await expect.poll(async () => await deleted(a.key)).toBe(true);
    await expect.poll(async () => await deleted(b.key)).toBe(true);
  });

  it("should show a rename from another client", async () => {
    const l = await createLabel();
    await renderLabels();
    await search(l.name);
    await waitForStreamLive(client.connection);
    const remote = createTestClient();
    await remote.labels.retrieve(l.key);
    const next = uniqueName("label");
    await remote.labels.dispatch(l.key, label.set({ path: ["name"], value: next }));
    expect(await screen.findByText(next)).toBeTruthy();
  });

  it("should drop a label that another client deletes", async () => {
    const l = await createLabel();
    await renderLabels();
    await search(l.name);
    await waitForStreamLive(client.connection);
    await createTestClient().labels.delete(l.key);
    expect(await screen.findByText("No labels")).toBeTruthy();
  });

  it("should not let a user who cannot edit labels change or create them", async () => {
    const l = await createLabel();
    const limited = await createTestClientWithPolicy(client, {
      name: uniqueName("policy"),
      objects: [label.TYPE_RESOURCE_ID],
      actions: ["retrieve", "delete"],
    });
    await awaitGranted(limited, label.TYPE_RESOURCE_ID, "delete");
    await renderLabels(limited);
    await search(l.name);
    const line = screen.getByRole("option", { name: l.name });
    await openMenu(l.name);
    // Delete proves the menu's grants answered, so the missing Rename is the gate.
    expect(await screen.findByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    const swatch = within(line).getByRole("button", { name: "Color" });
    expect(swatch.getAttribute("aria-disabled")).toBe("true");
    expect(screen.queryByRole("button", { name: "Create label" })).toBeNull();
  });
});
