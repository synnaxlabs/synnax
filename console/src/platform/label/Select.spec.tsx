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
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
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
  uniqueName,
} from "@/testutil";

const client = createTestClient();

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
      color: "#00FF00",
    });
    const onChange = vi.fn();
    await renderSelect(<Label.SelectMultiple value={[]} onChange={onChange} />);
    await searchAndClickLabel(label.name);
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    expect(onChange.mock.calls[0][0]).toEqual([label.key]);
  });

  it("should add a created label to the value the field holds when it closes", async () => {
    const createdBefore = await client.labels.create({
      name: uniqueName("label"),
      color: "#00FF00",
    });
    const addedMeanwhile = await client.labels.create({
      name: uniqueName("label"),
      color: "#0000FF",
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
    fireEvent.click(await screen.findByText("Create label"));
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
      color: "#00FF00",
    });
    const onChange = vi.fn();
    await renderSelect(
      <Label.SelectMultiple value={[existing.key]} onChange={onChange} />,
    );
    // A trigger that holds labels opens through its add button.
    fireEvent.click(getIconButton(await findDialogTrigger(), "add"));
    fireEvent.click(await screen.findByText("Create label"));
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
});

describe("Label.Select permissions", () => {
  const createSubject = async (grants: Grants) =>
    await createTestClientWithGrants(client, {
      ...grants,
      retrieve: [label.TYPE_RESOURCE_ID],
    });

  const renderPicker = async (as: Client): Promise<string> => {
    const { name } = await client.labels.create({
      name: uniqueName("label"),
      color: "#0000FF",
    });
    await renderSelect(<Label.SelectMultiple value={[]} onChange={vi.fn()} />, as);
    fireEvent.click(await findDialogTrigger());
    const search = await screen.findByPlaceholderText("Search labels...");
    fireEvent.change(search, { target: { value: name } });
    return name;
  };

  it("should withhold Create label from a subject who cannot create labels", async () => {
    const name = await renderPicker(await createSubject({}));
    // The listed label proves the picker resolved its data, so the missing action
    // is the gate and not a list still in flight.
    await screen.findByText(name);
    expect(screen.queryByText("Create label")).toBeNull();
    // A footer that renders null still counts as a footer, which moves the border
    // off the list onto an empty wrapper.
    expect(document.querySelector(".lyra-select__body")).toBeNull();
  });

  it("should offer Create label to a subject who may create labels", async () => {
    await renderPicker(await createSubject({ create: [label.TYPE_RESOURCE_ID] }));
    expect(await screen.findByText("Create label")).toBeTruthy();
  });
});
