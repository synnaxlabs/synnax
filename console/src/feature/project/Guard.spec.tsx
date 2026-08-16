// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { Session } from "@/session";
import { renderWithConsole } from "@/testutil";

describe("project/Guard", () => {
  const selected = "00000000-0000-0000-0000-000000000001";

  // The splash's own contents are permission-dependent, so identify it by its root
  // rather than by any action it offers.
  const querySplash = (container: HTMLElement): Element | null =>
    container.querySelector(".console-project-splash");

  it("should render the splash instead of children when no project is active", async () => {
    const { container } = await renderWithConsole(
      <Project.Guard>
        <div>protected content</div>
      </Project.Guard>,
    );
    expect(screen.queryByText("protected content")).toBeNull();
    expect(querySplash(container)).not.toBeNull();
  });

  it("should render children when a project is active", async () => {
    const { container } = await renderWithConsole(
      <Project.Guard>
        <div>protected content</div>
      </Project.Guard>,
      {
        preloadedState: {
          [Session.Project.SLICE_NAME]: {
            ...Session.Project.ZERO_SLICE_STATE,
            selected,
          },
        },
      },
    );
    expect(screen.getByText("protected content")).toBeDefined();
    expect(querySplash(container)).toBeNull();
  });

  it("should swap the splash for children when a project becomes active", async () => {
    const { container, store } = await renderWithConsole(
      <Project.Guard>
        <div>protected content</div>
      </Project.Guard>,
    );
    expect(screen.queryByText("protected content")).toBeNull();
    act(() => {
      store.dispatch(Session.Project.select(selected));
    });
    expect(screen.getByText("protected content")).toBeDefined();
    expect(querySplash(container)).toBeNull();
  });

  it("should fall back to the splash when the active project is cleared", async () => {
    const { container, store } = await renderWithConsole(
      <Project.Guard>
        <div>protected content</div>
      </Project.Guard>,
      {
        preloadedState: {
          [Session.Project.SLICE_NAME]: {
            ...Session.Project.ZERO_SLICE_STATE,
            selected,
          },
        },
      },
    );
    expect(screen.getByText("protected content")).toBeDefined();
    act(() => {
      store.dispatch(Session.Project.clearSelected());
    });
    expect(screen.queryByText("protected content")).toBeNull();
    expect(querySplash(container)).not.toBeNull();
  });
});
