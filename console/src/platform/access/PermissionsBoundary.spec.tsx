// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createProxiedTestClient,
  createTestClient,
  createTestClientWithRole,
  FAST_RETRY,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { act, fireEvent, type RenderOptions, screen } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { describe, expect, it } from "vitest";

import { Access } from "@/platform/access";
import { createConsoleWrapper, renderSuspended } from "@/testutil";

const root = createTestClient();

const renderBoundary = async (
  wrapper: FC<PropsWithChildren>,
  onCaughtError?: RenderOptions["onCaughtError"],
): Promise<void> => {
  await renderSuspended(
    <Access.PermissionsBoundary loading={<span>loading permissions</span>}>
      <span>workspace</span>
    </Access.PermissionsBoundary>,
    { wrapper, onCaughtError },
  );
};

describe("Access.PermissionsBoundary", () => {
  it("should replace the loading content with the children", async () => {
    const { wrapper } = await createConsoleWrapper({
      client: await createTestClientWithRole(root, "Operator"),
    });
    await renderBoundary(wrapper);
    expect(await screen.findByText("workspace")).toBeTruthy();
    expect(screen.queryByText("loading permissions")).toBeNull();
  });

  it("should load the permissions again on Retry after a failure", async () => {
    const { proxy, client } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onRetry: () => {},
    });
    const { wrapper } = await createConsoleWrapper({ client });
    await proxy.sever();
    await renderBoundary(wrapper, () => {});
    const retry = await screen.findByRole("button", { name: "Retry" });
    expect(screen.queryByText("workspace")).toBeNull();
    await proxy.restore();
    await waitForStreamLive(client.connection);
    await act(async () => {
      fireEvent.click(retry);
    });
    expect(await screen.findByText("workspace")).toBeTruthy();
  });
});
