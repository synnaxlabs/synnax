// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type Synnax } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import { Project } from "@/platform/project";
import { Session } from "@/session";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const client: Synnax = createTestClient();

const TIMEOUT = { timeout: 5000 };

const Harness = (): ReactElement => {
  const open = Project.useCreateModal();
  return <button onClick={() => open()}>open</button>;
};
Harness.displayName = "Harness";

describe("Project.useCreateModal", () => {
  it("renders the create form and disables Create without a client", async () => {
    await renderWithConsole(
      <>
        <Harness />
        <Modals.Stack />
      </>,
    );
    fireEvent.click(screen.getByRole("button", { name: "open" }));
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Project Name")).toBeTruthy(),
    );
    const createBtn = screen.getByRole("button", { name: "Create" });
    expect(createBtn.className).toContain("pluto--disabled");
  });

  it("creates the project on the cluster, activates it, and closes", async () => {
    const { wrapper, store } = await createConsoleWrapper({ client });
    render(
      <>
        <Harness />
        <Modals.Stack />
      </>,
      { wrapper },
    );
    fireEvent.click(screen.getByRole("button", { name: "open" }));
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Project Name")).toBeTruthy(),
    );

    const name = `proj-${id.create()}`;
    fireEvent.change(screen.getByPlaceholderText("Project Name"), {
      target: { value: name },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => {
      const active = Session.Project.selectOptionalSelected(store.getState());
      expect(active).not.toBeUndefined();
    }, TIMEOUT);
    await waitFor(
      () => expect(screen.queryByPlaceholderText("Project Name")).toBeNull(),
      TIMEOUT,
    );

    const active = Session.Project.selectSelected(store.getState());
    const created = await client.projects.retrieve(active);
    expect(created.name).toBe(name);
  });
});
