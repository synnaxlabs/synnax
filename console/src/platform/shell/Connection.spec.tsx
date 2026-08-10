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

import { Shell } from "@/platform/shell";
import { CONNECTION_PARAMS } from "@/session/cluster/testutil";
import { createConnectedConsoleWrapper } from "@/testutil";

const CLUSTER = { name: "Local", ...CONNECTION_PARAMS };

describe("Connection", () => {
  it("should report the live client status for the connected Core", async () => {
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: CONNECTION_PARAMS,
    });
    render(<Shell.Connection cluster={CLUSTER} />, { wrapper });
    expect(await screen.findByText("Connected")).toBeTruthy();
  });

  it("should check a Core that shares the host and port but not the TLS mode", async () => {
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: CONNECTION_PARAMS,
    });
    render(<Shell.Connection cluster={{ ...CLUSTER, secure: true }} />, { wrapper });
    expect(await screen.findByText("Unreachable")).toBeTruthy();
  });
});
