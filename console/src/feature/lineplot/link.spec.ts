// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { Session } from "@/session";
import { renderLinkHook } from "@/testutil";

const client = createTestClient();

describe("LinePlot.useLink", () => {
  it("should place a line plot layout for the retrieved line plot", async () => {
    const project = await client.projects.create({ name: id.create(), layout: {} });
    const linePlot = await client.lineplots.create(project.key, {
      name: "Tank Pressure",
    });
    const { handler, store } = await renderLinkHook(LinePlot.useLink);
    await handler({ client, key: linePlot.key });
    expect(Session.Layout.select(store.getState(), linePlot.key)?.name).toBe(
      "Tank Pressure",
    );
  });
});
