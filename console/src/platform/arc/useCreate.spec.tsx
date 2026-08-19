// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/platform/arc";
import { client } from "@/platform/arc/testutil";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper, selectTestProject } from "@/testutil";

const setup = async () => {
  const { wrapper: Console, store } = await createConsoleWrapper({ client });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      {children}
      <Modals.Stack />
    </Console>
  );
  const { result } = renderHook(Arc.useCreate, { wrapper });
  // Modal content that suspends is discarded when it opens inside a synchronous act
  // scope, so the open needs an awaited one.
  const open = async () => {
    await act(async () => {
      result.current();
    });
  };
  return { store, open };
};

describe("arc useCreate", () => {
  it("should reject an empty name with a validation error", async () => {
    const { open } = await setup();
    await open();
    await waitFor(() => expect(screen.getByPlaceholderText("Name")).toBeTruthy());
    const create = await screen.findByRole("button", { name: "Create" });
    await act(async () => {
      fireEvent.click(create);
    });
    await waitFor(() => expect(screen.getByText("Name is required")).toBeTruthy());
  });

  it("should create the arc on the server after the modal completes", async () => {
    const name = id.create();
    const { store, open } = await setup();
    await selectTestProject(store, client);
    await open();

    const input = await screen.findByPlaceholderText("Name");
    fireEvent.change(input, { target: { value: name } });

    const create = await screen.findByRole("button", { name: "Create" });
    await act(async () => {
      fireEvent.click(create);
    });

    const retrieved = await waitFor(async () => await client.arcs.retrieve({ name }));
    expect(retrieved.name).toBe(name);
  });
});
