// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Version } from "@/platform/version";
import {
  callsOf,
  clearUpdater,
  createWrapper,
  mockUpdater,
} from "@/platform/version/testutil";

const renderBadge = async (): Promise<void> => {
  render(<Version.Badge />, { wrapper: await createWrapper() });
};

describe("version Badge", () => {
  beforeEach(() => {
    mocks.engine = "tauri";
  });

  afterEach(async () => {
    await clearUpdater();
    vi.clearAllMocks();
  });

  it("should render nothing while the Console is current", async () => {
    const ipc = mockUpdater({ served: "1.5.0" });
    await renderBadge();
    await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(1));
    await act(async () => {});
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("should prompt to update when an update is available", async () => {
    mockUpdater({ served: "1.5.1" });
    await renderBadge();
    expect(await screen.findByText("Update available")).toBeTruthy();
  });

  it("should open the info modal when the update prompt is clicked", async () => {
    mockUpdater({ version: "1.5.0", served: "1.5.1" });
    await renderBadge();
    fireEvent.click(await screen.findByText("Update available"));
    await waitFor(() => expect(screen.getByText("Console v1.5.0")).toBeTruthy());
  });
});
