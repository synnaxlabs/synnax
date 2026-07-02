// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";

import { renderWithModals } from "@/platform/modals/testutil";
import { Ontology } from "@/platform/ontology";

interface HarnessProps {
  items: { name: string } | { name: string }[];
  type?: string;
  description?: string;
}

const Harness = ({
  items,
  type = "Channel",
  description,
}: HarnessProps): ReactElement => {
  const confirm = Ontology.useConfirmDelete({ type, description });
  const [result, setResult] = useState<string>("pending");
  return (
    <button onClick={() => void confirm(items).then((r) => setResult(String(r)))}>
      confirm:{result}
    </button>
  );
};
Harness.displayName = "Harness";

const clickConfirm = (): void => {
  fireEvent.click(screen.getByText(/^confirm:/));
};

const clickLabel = async (label: string): Promise<void> => {
  const button = screen
    .getAllByText(label)
    .map((el) => el.closest<HTMLButtonElement>("button"))
    .find((b) => b != null);
  if (button == null) throw new Error(`${label} button not found`);
  await act(async () => {
    fireEvent.click(button);
  });
};

describe("useConfirmDelete", () => {
  it("should prompt with the resource name for a single item", async () => {
    renderWithModals(<Harness items={{ name: "Sensor A" }} />);
    clickConfirm();
    await waitFor(() =>
      expect(
        screen.getByText("Are you sure you want to delete Sensor A?"),
      ).toBeTruthy(),
    );
  });

  it("should prompt with a pluralized count for multiple items", async () => {
    renderWithModals(<Harness items={[{ name: "a" }, { name: "b" }, { name: "c" }]} />);
    clickConfirm();
    await waitFor(() =>
      expect(
        screen.getByText("Are you sure you want to delete 3 channels?"),
      ).toBeTruthy(),
    );
  });

  it("should show the default irreversibility description", async () => {
    renderWithModals(<Harness items={{ name: "Sensor A" }} />);
    clickConfirm();
    await waitFor(() =>
      expect(screen.getByText("This action cannot be undone.")).toBeTruthy(),
    );
  });

  it("should use a custom description when provided", async () => {
    renderWithModals(
      <Harness items={{ name: "Sensor A" }} description="Custom warning." />,
    );
    clickConfirm();
    await waitFor(() => expect(screen.getByText("Custom warning.")).toBeTruthy());
  });

  it("should resolve true when the user confirms", async () => {
    renderWithModals(<Harness items={{ name: "Sensor A" }} />);
    clickConfirm();
    await screen.findByText("Are you sure you want to delete Sensor A?");
    await clickLabel("Delete");
    await waitFor(() => expect(screen.getByText("confirm:true")).toBeTruthy());
  });

  it("should resolve false when the user cancels", async () => {
    renderWithModals(<Harness items={{ name: "Sensor A" }} />);
    clickConfirm();
    await screen.findByText("Are you sure you want to delete Sensor A?");
    await clickLabel("Cancel");
    await waitFor(() => expect(screen.getByText("confirm:false")).toBeTruthy());
  });
});
