// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, type RenderResult, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/platform/channel";
import { renderWithModals } from "@/platform/modals/testutil";

const Harness = (): ReactElement => {
  const open = Channel.useCalculatedModal();
  return <button onClick={() => open()}>open</button>;
};
Harness.displayName = "Harness";

const openModal = (): RenderResult => {
  const result = renderWithModals(<Harness />);
  fireEvent.click(screen.getByRole("button", { name: "open" }));
  return result;
};

describe("useCalculatedModal", () => {
  describe("create mode", () => {
    it("should render the calculated create header", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Calculated")).toBeTruthy());
      expect(screen.getByText("Channel")).toBeTruthy();
    });

    it("should render the operation type buttons", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Derivative")).toBeTruthy());
      expect(screen.getByText("None")).toBeTruthy();
      expect(screen.getByText("Min")).toBeTruthy();
      expect(screen.getByText("Max")).toBeTruthy();
      expect(screen.getByText("Average")).toBeTruthy();
    });

    it("should show the window and reset-channel fields for non-derivative operations", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Window")).toBeTruthy());
      expect(screen.getByText("Reset Channel")).toBeTruthy();
    });

    it("should hide the window and reset-channel fields for the derivative operation", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Derivative")).toBeTruthy());
      fireEvent.click(screen.getByText("Derivative"));
      await waitFor(() => expect(screen.queryByText("Window")).toBeNull());
      expect(screen.queryByText("Reset Channel")).toBeNull();
    });

    it("should restore the window and reset-channel fields when switching back to a windowed operation", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Derivative")).toBeTruthy());
      fireEvent.click(screen.getByText("Derivative"));
      await waitFor(() => expect(screen.queryByText("Window")).toBeNull());
      fireEvent.click(screen.getByText("Average"));
      await waitFor(() => expect(screen.getByText("Window")).toBeTruthy());
      expect(screen.getByText("Reset Channel")).toBeTruthy();
    });
  });
});
