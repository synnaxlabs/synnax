// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Label } from "@/platform/label";
import { searchAndClickLabel } from "@/platform/label/testutil";
import { Modals } from "@/platform/modals";
import {
  createConsoleWrapper,
  findDialogTrigger,
  getIconButton,
  stubGeometry,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const renderSelect = async (ui: ReactElement) => {
  const { wrapper } = await createConsoleWrapper({ client });
  return render(
    <>
      {ui}
      <Modals.Stack />
    </>,
    { wrapper },
  );
};

const openAddModal = async (): Promise<void> => {
  fireEvent.click(await findDialogTrigger());
  const addIcon = await waitFor(() => getIconButton(document.body, "add"));
  fireEvent.click(addIcon);
};

stubGeometry();

describe("Label.Select", () => {
  it("should pass the clicked label's key to onChange in SelectSingle", async () => {
    const label = await client.labels.create({
      name: uniqueName("label"),
      color: "#FF0000",
    });
    const onChange = vi.fn();
    await renderSelect(<Label.SelectSingle value={undefined} onChange={onChange} />);
    await searchAndClickLabel(label.name);
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    expect(onChange.mock.calls[0][0]).toBe(label.key);
  });

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

  it("should open the label editor from the SelectSingle add action", async () => {
    await renderSelect(<Label.SelectSingle value={undefined} onChange={vi.fn()} />);
    await openAddModal();
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Search labels...")).toBeTruthy(),
    );
  });

  it("should open the label editor from the SelectMultiple add action", async () => {
    await renderSelect(<Label.SelectMultiple value={[]} onChange={vi.fn()} />);
    await openAddModal();
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Search labels...")).toBeTruthy(),
    );
  });
});
