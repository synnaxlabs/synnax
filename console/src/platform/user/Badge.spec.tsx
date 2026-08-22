// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { User } from "@/platform/user";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const createStateWithUser = (username: string, selected = "LOCAL") =>
  createCoreState([createCore("LOCAL", { name: "Local", username })], selected);

const getTrigger = (container: ParentNode): HTMLElement => {
  const trigger = container.querySelector<HTMLElement>(".pluto-dialog__trigger");
  if (trigger == null) throw new Error("badge trigger not found");
  return trigger;
};

describe("User.Badge", () => {
  it("should fall back to the Core username when no user is loaded", async () => {
    await renderWithConsole(<User.Badge />, {
      preloadedState: createStateWithUser("Core-user"),
    });
    expect(screen.getByText("Core-user")).toBeTruthy();
  });

  it("should prefer the retrieved user's identity over the Core fallback", async () => {
    const client = createTestClient();
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: createStateWithUser("fallback_user"),
    });
    render(<User.Badge />, { wrapper });
    await waitFor(() => expect(screen.getByText("synnax")).toBeTruthy());
    expect(screen.queryByText("fallback_user")).toBeNull();
  });

  it("should name the subject's role in the dialog", async () => {
    const root = createTestClient();
    const viewer = await createTestClientWithRole(root, "Viewer");
    const { wrapper } = await createConsoleWrapper({
      client: viewer,
      preloadedState: createStateWithUser("fallback_user"),
    });
    const { container } = render(<User.Badge />, { wrapper });
    fireEvent.click(getTrigger(container));
    const role = await waitFor(() => {
      const el = document.body.querySelector(".console-user-badge__roles");
      if (el == null) throw new Error("role not rendered");
      return el;
    });
    expect(role.textContent).toContain("Viewer");
  });

  it("should show the subject's full name and username in the dialog", async () => {
    const root = createTestClient();
    const other = await createTestClientWithRole(root, "Viewer");
    const { wrapper } = await createConsoleWrapper({
      client: other,
      preloadedState: createStateWithUser("fallback_user"),
    });
    const { container } = render(<User.Badge />, { wrapper });
    fireEvent.click(getTrigger(container));
    expect(await screen.findByText("test test")).toBeTruthy();
    expect(await screen.findByText(other.auth?.user?.username ?? "")).toBeTruthy();
  });

  it("should log out of the active Core when Log out is clicked", async () => {
    const { store } = await renderWithConsole(<User.Badge />, {
      preloadedState: createStateWithUser("Core-user"),
    });
    expect(Session.Core.selectSelectedKey(store.getState())).toBe("LOCAL");
    fireEvent.click(screen.getByText("Core-user"));
    const logout = await screen.findByText("Log out");
    fireEvent.click(logout);
    await waitFor(() =>
      expect(Session.Core.selectSelectedKey(store.getState())).toBeUndefined(),
    );
  });
});
