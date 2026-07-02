// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  type RenderResult,
  screen,
  waitFor,
} from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/platform/channel";
import { Modals } from "@/platform/modals";
import { renderWithModals } from "@/platform/modals/testutil";
import { createConsoleWrapper } from "@/testutil";

const TIMEOUT = { timeout: 5000 };

const Harness = (): ReactElement => {
  const open = Channel.useCreateModal();
  return <button onClick={() => open()}>open</button>;
};
Harness.displayName = "Harness";

const openModal = (): RenderResult => {
  const result = renderWithModals(<Harness />);
  fireEvent.click(screen.getByRole("button", { name: "open" }));
  return result;
};

const getSwitch = (label: string): HTMLInputElement => {
  const text = screen.getByText(label);
  const input = text
    .closest("*")
    ?.parentElement?.querySelector<HTMLInputElement>("input[type='checkbox']");
  if (input == null) throw new Error(`switch ${label} not found`);
  return input;
};

describe("useCreateModal", () => {
  describe("static render", () => {
    it("should render the create header", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Channel")).toBeTruthy());
    });

    it("should render the name, virtual and index toggles", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Virtual")).toBeTruthy());
      expect(screen.getByText("Is index")).toBeTruthy();
      expect(screen.getByText("Data type")).toBeTruthy();
    });

    it("should render the create-more toggle and create button", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Create More")).toBeTruthy());
      expect(screen.getByRole("button", { name: "Create" })).toBeTruthy();
    });
  });

  describe("index / virtual interaction", () => {
    it("should disable the index toggle once virtual is enabled", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Virtual")).toBeTruthy());
      const isIndex = getSwitch("Is index");
      expect(isIndex.disabled).toBe(false);
      fireEvent.click(getSwitch("Virtual"));
      await waitFor(() => expect(getSwitch("Is index").disabled).toBe(true));
    });

    it("should disable the virtual toggle once is-index is enabled", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Is index")).toBeTruthy());
      const virtual = getSwitch("Virtual");
      expect(virtual.disabled).toBe(false);
      fireEvent.click(getSwitch("Is index"));
      await waitFor(() => expect(getSwitch("Virtual").disabled).toBe(true));
    });
  });

  describe("with test client", () => {
    it("should create an index channel through the modal", async () => {
      const client = createTestClient();
      const { wrapper } = await createConsoleWrapper({ client });

      render(
        <>
          <Harness />
          <Modals.Stack />
        </>,
        { wrapper },
      );
      fireEvent.click(screen.getByRole("button", { name: "open" }));

      const name = `test_ch_${id.create().replace(/-/g, "_")}`;
      const nameInput = await screen.findByPlaceholderText("Name");
      fireEvent.change(nameInput, { target: { value: name } });

      fireEvent.click(getSwitch("Is index"));

      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: "Create" }));
      });

      await waitFor(async () => {
        const created = await client.channels.retrieve(name);
        expect(created.name).toEqual(name);
        expect(created.isIndex).toBe(true);
      }, TIMEOUT);
    });
  });
});
