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
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { User } from "@/platform/user";
import { createCore, createCoreState } from "@/session/core/testutil";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const CORE_KEY = "core-key";

const createStateWithUser = (username: string) =>
  createCoreState([createCore("Local", { key: CORE_KEY, username })], CORE_KEY);

describe("User.Info", () => {
  it("should never show the Core username", async () => {
    await renderWithConsole(<User.Info />, {
      preloadedState: createStateWithUser("Core-user"),
    });
    expect(screen.queryByText("Core-user")).toBeNull();
  });

  it("should name the subject's role", async () => {
    const root = createTestClient();
    const viewer = await createTestClientWithRole(root, "Viewer");
    const { wrapper } = await createConsoleWrapper({
      client: viewer,
      preloadedState: createStateWithUser("fallback_user"),
    });
    render(<User.Info />, { wrapper });
    expect(await screen.findByText("Viewer")).toBeTruthy();
  });

  it("should show the subject's name and not the Core username", async () => {
    const root = createTestClient();
    const other = await createTestClientWithRole(root, "Viewer");
    const { wrapper } = await createConsoleWrapper({
      client: other,
      preloadedState: createStateWithUser("core_user"),
    });
    render(<User.Info />, { wrapper });
    expect(await screen.findByText("test")).toBeTruthy();
    expect(screen.queryByText("core_user")).toBeNull();
  });
});
