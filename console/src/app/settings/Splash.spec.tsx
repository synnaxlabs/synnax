// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Settings } from "@/app/settings";
import { Project } from "@/feature/project";
import { Session } from "@/session";
import { renderWithConsole } from "@/testutil";

describe("Settings.Splash", () => {
  const querySplash = (container: HTMLElement): Element | null =>
    container.querySelector(".console-project-splash");

  it("should open Settings in place of the splash and go back", async () => {
    const { container } = await renderWithConsole(
      <Project.Guard Settings={Settings.Splash}>
        <div>protected content</div>
      </Project.Guard>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    expect(await screen.findByRole("navigation", { name: "Settings" })).toBeDefined();
    expect(querySplash(container)).toBeNull();
    expect(screen.queryByRole("button", { name: "Settings" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    expect(screen.queryByRole("navigation", { name: "Settings" })).toBeNull();
    expect(querySplash(container)).not.toBeNull();
  });

  it("should show the splash, not Settings, when the next project closes", async () => {
    const { container, store } = await renderWithConsole(
      <Project.Guard Settings={Settings.Splash}>
        <div>protected content</div>
      </Project.Guard>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    expect(await screen.findByRole("navigation", { name: "Settings" })).toBeDefined();
    act(() => {
      store.dispatch(Session.Project.select("00000000-0000-0000-0000-000000000001"));
    });
    expect(screen.getByText("protected content")).toBeDefined();
    act(() => {
      store.dispatch(Session.Project.clearSelected());
    });
    expect(screen.queryByRole("navigation", { name: "Settings" })).toBeNull();
    expect(querySplash(container)).not.toBeNull();
  });

  it("should leave the gear off a splash with no Settings", async () => {
    await renderWithConsole(
      <Project.Guard>
        <div>protected content</div>
      </Project.Guard>,
    );
    expect(await screen.findByText("Projects")).toBeDefined();
    expect(screen.queryByRole("button", { name: "Settings" })).toBeNull();
  });
});
