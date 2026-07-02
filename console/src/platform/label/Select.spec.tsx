// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Label } from "@/platform/label";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper } from "@/testutil";

const TIMEOUT = { timeout: 5000 };

const renderSelect = async (ui: ReactElement) => {
  const client = createTestClient();
  const { wrapper } = await createConsoleWrapper({ client });
  return render(
    <>
      {ui}
      <Modals.Stack />
    </>,
    { wrapper },
  );
};

const getTrigger = async (): Promise<HTMLElement> =>
  await waitFor(() => {
    const el = document.querySelector<HTMLElement>(".pluto-dialog__trigger");
    if (el == null) throw new Error("select trigger not found");
    return el;
  }, TIMEOUT);

const openAddModal = async (): Promise<void> => {
  fireEvent.click(await getTrigger());
  const addIcon = await waitFor(() => {
    const el = screen.getByLabelText("pluto-icon--add").closest("button");
    if (el == null) throw new Error("add button not found");
    return el;
  }, TIMEOUT);
  fireEvent.click(addIcon);
};

describe("Label.Select", () => {
  it("should render a SelectSingle trigger", async () => {
    await renderSelect(<Label.SelectSingle value={undefined} onChange={vi.fn()} />);
    expect(await getTrigger()).toBeTruthy();
  });

  it("should render a SelectMultiple trigger", async () => {
    await renderSelect(<Label.SelectMultiple value={[]} onChange={vi.fn()} />);
    expect(await getTrigger()).toBeTruthy();
  });

  it("should open the label editor from the SelectSingle add action", async () => {
    await renderSelect(<Label.SelectSingle value={undefined} onChange={vi.fn()} />);
    await openAddModal();
    await waitFor(
      () => expect(screen.getByText("Search labels")).toBeTruthy(),
      TIMEOUT,
    );
  });

  it("should open the label editor from the SelectMultiple add action", async () => {
    await renderSelect(<Label.SelectMultiple value={[]} onChange={vi.fn()} />);
    await openAddModal();
    await waitFor(
      () => expect(screen.getByText("Search labels")).toBeTruthy(),
      TIMEOUT,
    );
  });
});
