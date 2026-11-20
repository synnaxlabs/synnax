// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Label } from "@/platform/label";
import { searchAndClickLabel } from "@/platform/label/testutil";
import { Modals } from "@/platform/modals";
import {
  createConsoleWrapper,
  createTestClientWithGrants,
  findDialogTrigger,
  getIconButton,
  type Grants,
  uniqueToken,
} from "@/testutil";

const client = createTestClient();

const openAndSearch = async (term: string): Promise<void> => {
  fireEvent.click(await findDialogTrigger());
  fireEvent.change(await screen.findByPlaceholderText("Search labels..."), {
    target: { value: term },
  });
};

const renderSelect = async (ui: ReactElement, as: Client = client) => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  return render(
    <>
      {ui}
      <Modals.Stack />
    </>,
    { wrapper },
  );
};

describe("Label.Select", () => {
  it("should pass the clicked label's key to onChange in SelectMultiple", async () => {
    const label = await client.labels.create({
      name: uniqueName("label"),
      color: { type: "fixed", value: "#00FF00" },
    });
    const onChange = vi.fn();
    await renderSelect(<Label.SelectMultiple value={[]} onChange={onChange} />);
    await searchAndClickLabel(label.name);
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    expect(onChange.mock.calls[0][0]).toEqual([label.key]);
  });

  it("should show the labels of its value before it first opens", async () => {
    const label = await client.labels.create({
      name: uniqueName("label"),
      color: { type: "fixed", value: "#FF0000" },
    });
    await renderSelect(<Label.SelectMultiple value={[label.key]} onChange={vi.fn()} />);
    expect(await screen.findByText(label.name)).toBeTruthy();
    expect(screen.queryByPlaceholderText("Search labels...")).toBeNull();
  });

  it("should remove a label from its tag before it first opens", async () => {
    const label = await client.labels.create({
      name: uniqueName("label"),
      color: { type: "fixed", value: "#FF0000" },
    });
    const onChange = vi.fn();
    await renderSelect(
      <Label.SelectMultiple value={[label.key]} onChange={onChange} />,
    );
    fireEvent.click(await screen.findByLabelText(`Remove ${label.name}`));
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    expect(onChange.mock.calls[0][0]).toEqual([]);
  });

  it("should keep the tag that first opens it, so it can open beside it", async () => {
    const label = await client.labels.create({
      name: uniqueName("label"),
      color: { type: "fixed", value: "#FF0000" },
    });
    await renderSelect(<Label.SelectMultiple value={[label.key]} onChange={vi.fn()} />);
    const tag = await screen.findByText(label.name);
    fireEvent.click(tag);
    await screen.findByPlaceholderText("Search labels...");
    expect(tag.isConnected).toBe(true);
  });

  it("should keep its class name before it first opens", async () => {
    const { container } = await renderSelect(
      <Label.SelectMultiple value={[]} onChange={vi.fn()} className="labels" />,
    );
    expect(container.querySelector(".labels")).not.toBeNull();
  });

  it("should add a created label to the value the field holds when it closes", async () => {
    const createdBefore = await client.labels.create({
      name: uniqueName("label"),
      color: { type: "fixed", value: "#00FF00" },
    });
    const addedMeanwhile = await client.labels.create({
      name: uniqueName("label"),
      color: { type: "fixed", value: "#0000FF" },
    });
    const onChange = vi.fn();
    const renderField = (value: label.Key[]): ReactElement => (
      <>
        <Label.SelectMultiple value={value} onChange={onChange} />
        <Modals.Stack />
      </>
    );
    const { wrapper } = await createConsoleWrapper({ client });
    const { rerender } = render(renderField([createdBefore.key]), { wrapper });
    fireEvent.click(getIconButton(await findDialogTrigger(), "add"));
    fireEvent.click(await screen.findByText("New label"));
    const name = uniqueName("label");
    fireEvent.change(await screen.findByPlaceholderText("Label name"), {
      target: { value: name },
    });
    rerender(renderField([createdBefore.key, addedMeanwhile.key]));
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Create" }));
    });
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    const [created] = await client.labels.retrieve({ name: [name] });
    expect(onChange.mock.calls[0][0]).toEqual([
      createdBefore.key,
      addedMeanwhile.key,
      created.key,
    ]);
  });

  it("should add a label created from the field to the selection", async () => {
    const existing = await client.labels.create({
      name: uniqueName("label"),
      color: { type: "fixed", value: "#00FF00" },
    });
    const onChange = vi.fn();
    await renderSelect(
      <Label.SelectMultiple value={[existing.key]} onChange={onChange} />,
    );
    // A trigger that holds labels opens through its add button.
    fireEvent.click(getIconButton(await findDialogTrigger(), "add"));
    fireEvent.click(await screen.findByText("New label"));
    const name = uniqueName("label");
    fireEvent.change(await screen.findByPlaceholderText("Label name"), {
      target: { value: name },
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Create" }));
    });
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    const [created] = await client.labels.retrieve({ name: [name] });
    expect(onChange.mock.calls[0][0]).toEqual([existing.key, created.key]);
  });

  it("should create a label named after the search text in the create modal", async () => {
    const existing = await client.labels.create({
      name: uniqueName("label"),
      color: { type: "fixed", value: "#00FF00" },
    });
    const onChange = vi.fn();
    await renderSelect(
      <Label.SelectMultiple value={[existing.key]} onChange={onChange} />,
    );
    fireEvent.click(getIconButton(await findDialogTrigger(), "add"));
    const name = uniqueName("label");
    fireEvent.change(await screen.findByPlaceholderText("Search labels..."), {
      target: { value: name },
    });
    fireEvent.click(await screen.findByText(`Create label "${name}"`));
    const input = await screen.findByPlaceholderText("Label name");
    expect(input).toHaveProperty("value", name);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Create" }));
    });
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    const [created] = await client.labels.retrieve({ name: [name] });
    expect(onChange.mock.calls[0][0]).toEqual([existing.key, created.key]);
  });

  it("should hide the create option while the search matches a label name in any case", async () => {
    const existing = await client.labels.create({
      name: uniqueName("label"),
      color: { type: "fixed", value: "#00FF00" },
    });
    await renderSelect(<Label.SelectMultiple value={[]} onChange={vi.fn()} />);
    await openAndSearch(existing.name.toUpperCase());
    await screen.findByText(existing.name);
    expect(screen.queryByText(/^Create label/)).toBeNull();
  });
});

describe("Label.Select permissions", () => {
  const createSubject = async (grants: Grants) =>
    await createTestClientWithGrants(client, {
      ...grants,
      retrieve: [label.TYPE_RESOURCE_ID],
    });

  // A search that lists a label but names none shows the create option to a subject who
  // may create, so its absence proves the gate and not a list still in flight.
  const renderPicker = async (as: Client): Promise<string> => {
    const token = uniqueToken();
    const { name } = await client.labels.create({
      name: `${token} listed`,
      color: { type: "fixed", value: "#0000FF" },
    });
    await renderSelect(<Label.SelectMultiple value={[]} onChange={vi.fn()} />, as);
    await openAndSearch(token);
    await screen.findByText(name);
    return token;
  };

  it("should withhold the create option from a subject who cannot create labels", async () => {
    const token = await renderPicker(await createSubject({}));
    expect(screen.queryByText(`Create label "${token}"`)).toBeNull();
    // A footer that renders null still counts as a footer, which moves the border
    // off the list onto an empty wrapper.
    expect(document.querySelector(".lyra-select__body")).toBeNull();
  });

  it("should offer the create option to a subject who may create labels", async () => {
    const token = await renderPicker(
      await createSubject({ create: [label.TYPE_RESOURCE_ID] }),
    );
    expect(await screen.findByText(`Create label "${token}"`)).toBeTruthy();
  });
});
