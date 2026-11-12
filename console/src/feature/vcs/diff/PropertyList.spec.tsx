// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { type Change } from "@/feature/vcs/diff/change";
import { PropertyList } from "@/feature/vcs/diff/PropertyList";
import { Link } from "@/platform/link";
import { createConsoleWrapper, renderSuspended } from "@/testutil";

const client = createTestClient();

const DEVICE = "3c7d1f2a-9b4e-4f6a-8c21-5e0d7b9a1f34";

const PROPERTY: Change.Property = {
  path: ["Device"],
  kind: "changed",
  before: [{ text: "Empty" }],
  after: [{ text: "NI 9237 strain module", resource: { type: "device", key: DEVICE } }],
};

const renderList = async (handlers: Link.Handlers): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  await renderSuspended(
    <Link.Provider handlers={handlers}>
      <PropertyList properties={[PROPERTY]} />
    </Link.Provider>,
    { wrapper },
  );
};

describe("PropertyList", () => {
  it("should open the resource of a reference on click", async () => {
    const handler = vi.fn<Link.Handler>(async () => {});
    await renderList({ device: handler });
    fireEvent.click(screen.getByRole("link", { name: "NI 9237 strain module" }));
    await waitFor(() => expect(handler).toHaveBeenCalledWith({ client, key: DEVICE }));
  });

  it("should open the resource of a reference on Enter", async () => {
    const handler = vi.fn<Link.Handler>(async () => {});
    await renderList({ device: handler });
    fireEvent.keyDown(screen.getByRole("link"), { key: "Enter" });
    await waitFor(() => expect(handler).toHaveBeenCalledWith({ client, key: DEVICE }));
  });

  it("should show a reference as text when no link opens its type", async () => {
    await renderList({});
    expect(screen.getByText("NI 9237 strain module")).toBeTruthy();
    expect(screen.queryByRole("link")).toBeNull();
  });
});
