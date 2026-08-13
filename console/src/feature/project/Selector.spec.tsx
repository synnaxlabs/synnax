// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  getBySelector,
  renderWithConsole,
  uniqueName,
} from "@/testutil";

const client: Synnax = createTestClient();

const TRIGGER = ".console-project-selector__trigger";

describe("Project.Selector", () => {
  it("renders nothing when the user lacks retrieve permission", async () => {
    const { container } = await renderWithConsole(<Project.Selector />, {
      preloadedState: {
        [Session.Project.SLICE_NAME]: {
          ...Session.Project.ZERO_SLICE_STATE,
          selected: id.create(),
        },
      },
    });
    expect(container.querySelector(TRIGGER)).toBeNull();
  });

  it("switches the active project on selection", async () => {
    const active: project.Project = await client.projects.create({
      name: `proj-active-${id.create()}`,
      layout: {},
    });
    const target: project.Project = await client.projects.create({
      name: `proj-target-${id.create()}`,
      layout: {},
    });
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(active) },
    });
    const { container } = render(<Project.Selector />, { wrapper });

    const trigger = await waitFor(() => getBySelector(container, TRIGGER));
    fireEvent.click(trigger);
    const search = await screen.findByPlaceholderText("Search projects...");
    fireEvent.change(search, { target: { value: target.name } });
    fireEvent.click(await screen.findByText(target.name));

    await waitFor(() =>
      expect(Session.Project.selectSelected(store.getState())).toBe(target.key),
    );
  });

  it("gives numbered siblings different avatar initials", async () => {
    const active: project.Project = await client.projects.create({
      name: `proj-active-${id.create()}`,
      layout: {},
    });
    const prefix = uniqueName("stand");
    await client.projects.create({ name: `${prefix}_1`, layout: {} });
    await client.projects.create({ name: `${prefix}_2`, layout: {} });
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(active) },
    });
    const { container } = render(<Project.Selector />, { wrapper });

    const trigger = await waitFor(() => getBySelector(container, TRIGGER));
    fireEvent.click(trigger);
    const search = await screen.findByPlaceholderText("Search projects...");
    fireEvent.change(search, { target: { value: prefix } });
    await screen.findByText(`${prefix}_1`);

    expect(await screen.findByText("S1")).toBeTruthy();
    expect(await screen.findByText("S2")).toBeTruthy();
  });
});
