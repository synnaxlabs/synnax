// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { Session } from "@/session";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const client = createTestClient();

describe("project/Splash", () => {
  it("should offer no log out and name no Core", async () => {
    const { store } = await renderWithConsole(<Project.Splash />);
    act(() => {
      store.dispatch(Session.Core.select(Session.Core.LOCAL_KEY));
    });
    const name = Session.Core.selectSelected(store.getState())?.name;
    if (name == null) throw new Error("no Core is selected");
    expect(screen.getByText("Projects")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Log out" })).toBeNull();
    expect(screen.queryByText(name)).toBeNull();
  });

  describe("sample project", () => {
    let created: project.Key | undefined;

    afterEach(async () => {
      if (created == null) return;
      const [panel] = await client.panels.retrieve({
        parent: project.ontologyID(created),
      });
      const { root } = panel;
      if (root.variant !== "split" || root.last.variant !== "split")
        throw new Error("expected a split panel");
      const leaf = root.last.last;
      if (leaf.variant !== "leaf") throw new Error("expected a leaf");
      await client.arcs.delete(
        leaf.tabs.flatMap((tab) =>
          tab.variant === "resource" && tab.resource.type === "arc"
            ? [tab.resource.key]
            : [],
        ),
      );
      await client.projects.delete(created);
      created = undefined;
    });

    it("should create and open the sample project", async () => {
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(<Project.Splash />, { wrapper });
      fireEvent.click(await screen.findByText("Sample project"));
      await waitFor(
        () =>
          expect(
            Session.Project.selectOptionalSelected(store.getState()),
          ).toBeDefined(),
        { timeout: 10_000 },
      );
      const key = Session.Project.selectSelected(store.getState());
      created = key;
      expect((await client.projects.retrieve({ key })).name).toBe("Sample project");
    });
  });
});
