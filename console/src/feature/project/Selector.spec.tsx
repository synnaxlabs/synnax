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
  stubGeometry,
} from "@/testutil";

const client: Synnax = createTestClient();

// The selector's dialog list is virtualized and renders no rows under jsdom's
// zero-size layout.
stubGeometry();

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
    expect(container.querySelector(".console-trigger")).toBeNull();
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

    // The trigger renders the active project's avatar, not its name.
    const trigger = await waitFor(() => getBySelector(container, ".console-trigger"));
    fireEvent.click(trigger);
    // The dialog's list is virtualized and every project in the cluster is a
    // candidate, so search the target down rather than scrolling to it.
    const search = await screen.findByPlaceholderText("Search projects...");
    fireEvent.change(search, { target: { value: target.name } });
    fireEvent.click(await screen.findByText(target.name));

    await waitFor(() =>
      expect(Session.Project.selectSelected(store.getState())).toBe(target.key),
    );
  });
});
