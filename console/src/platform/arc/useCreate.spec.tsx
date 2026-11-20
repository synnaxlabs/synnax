// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/lyra/status";
import { id } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/platform/arc";
import { client } from "@/platform/arc/testutil";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper, resolveFocusedTab, selectTestProject } from "@/testutil";

const setup = async (props?: Arc.UseCreateProps) => {
  const { wrapper: Console, store } = await createConsoleWrapper({ client });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      {children}
      <Modals.Stack />
    </Console>
  );
  const { result } = renderHook(
    () => ({
      create: Arc.useCreate(props),
      statuses: Status.useNotifications().statuses,
    }),
    { wrapper },
  );
  // Modal content that suspends is discarded when it opens inside a synchronous act
  // scope, so the open needs an awaited one.
  const open = async () => {
    await act(async () => {
      result.current.create();
    });
  };
  return { store, open, statuses: () => result.current.statuses };
};

const createArcRack = async () =>
  await client.racks.create({
    name: uniqueName("rack"),
    integrations: [Arc.INTEGRATION_KEY],
  });

describe("arc useCreate", () => {
  it("should reject an empty name with a validation error", async () => {
    const { open } = await setup();
    await open();
    await waitFor(() => expect(screen.getByPlaceholderText("Arc name")).toBeTruthy());
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

    const input = await screen.findByPlaceholderText("Arc name");
    fireEvent.change(input, { target: { value: name } });

    const create = await screen.findByRole("button", { name: "Create" });
    await act(async () => {
      fireEvent.click(create);
    });

    const retrieved = await waitFor(async () => await client.arcs.retrieve({ name }));
    expect(retrieved.name).toBe(name);
    expect(retrieved.type).toBe("automation");
    expect(retrieved.source.mode).toBe("text");
  });

  it("should create a module when Module is selected", async () => {
    const name = id.create();
    const { store, open } = await setup();
    await selectTestProject(store, client);
    await open();

    const input = await screen.findByPlaceholderText("Arc name");
    fireEvent.change(input, { target: { value: name } });
    fireEvent.click(await screen.findByText("Module"));

    const create = await screen.findByRole("button", { name: "Create" });
    await act(async () => {
      fireEvent.click(create);
    });

    const retrieved = await waitFor(async () => await client.arcs.retrieve({ name }));
    expect(retrieved.type).toBe("module");
  });

  it("should bind a created automation to the given rack", async () => {
    const name = id.create();
    const rck = await createArcRack();
    const { store, open } = await setup({ rack: rck.key });
    await selectTestProject(store, client);
    await open();

    const input = await screen.findByPlaceholderText("Arc name");
    fireEvent.change(input, { target: { value: name } });

    const create = await screen.findByRole("button", { name: "Create" });
    await act(async () => {
      fireEvent.click(create);
    });

    const retrieved = await waitFor(async () => await client.arcs.retrieve({ name }));
    await waitFor(async () => {
      const tsk = await client.arcs.task.retrieve(retrieved.key);
      expect(tsk?.rack).toBe(rck.key);
    });
  });

  it("should create a module without binding it to the given rack", async () => {
    const name = id.create();
    const rck = await createArcRack();
    const { store, open, statuses } = await setup({ rack: rck.key });
    await selectTestProject(store, client);
    await open();

    const input = await screen.findByPlaceholderText("Arc name");
    fireEvent.change(input, { target: { value: name } });
    fireEvent.click(await screen.findByText("Module"));

    const create = await screen.findByRole("button", { name: "Create" });
    await act(async () => {
      fireEvent.click(create);
    });

    const retrieved = await waitFor(async () => await client.arcs.retrieve({ name }));
    await resolveFocusedTab(
      store,
      client,
      (t) => t.variant === "resource" && t.resource.key === retrieved.key,
    );
    expect(statuses()).not.toContainEqual(
      expect.objectContaining({ variant: "error" }),
    );
  });

  it("should reject a module name that is not an identifier", async () => {
    const { open } = await setup();
    await open();
    const input = await screen.findByPlaceholderText("Arc name");
    fireEvent.change(input, { target: { value: "bad name" } });
    fireEvent.click(await screen.findByText("Module"));
    const create = await screen.findByRole("button", { name: "Create" });
    await act(async () => {
      fireEvent.click(create);
    });
    await waitFor(() =>
      expect(screen.getByText(/must start with a letter or underscore/)).toBeTruthy(),
    );
  });
});
