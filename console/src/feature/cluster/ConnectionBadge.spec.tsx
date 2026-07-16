// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Cluster } from "@/feature/cluster";
import { hoverConnectionBadge } from "@/feature/cluster/testutil";
import { createConnectedConsoleWrapper, renderWithConsole } from "@/testutil";

describe("ConnectionBadge", () => {
  it("should report disconnected when no cluster connection exists", async () => {
    const { container } = await renderWithConsole(<Cluster.ConnectionBadge />);
    hoverConnectionBadge(container);
    expect((await screen.findAllByText("Disconnected")).length).toBeGreaterThan(0);
  });

  it("should report connected once the provider reaches the cluster", async () => {
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: {
        host: "localhost",
        port: 9090,
        username: "synnax",
        password: "seldon",
        secure: false,
      },
    });
    const { container } = render(<Cluster.ConnectionBadge />, { wrapper });
    hoverConnectionBadge(container);
    expect(await screen.findByText("Connected")).toBeTruthy();
  });
});
