// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";
import { renderPalette } from "@/platform/palette/testutil";
import { Session } from "@/session";
import { resolveFocusedTab, stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("HTTP palette commands", () => {
  it("should open the connect modal from the connect server command", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: HTTP.Device.COMMANDS,
      client,
    });
    await openCommandPalette("Connect an HTTP");
    await selectCommand("Connect an HTTP server");
    expect(await screen.findByPlaceholderText("www.example.com")).toBeTruthy();
  });

  it("should open the read task view from the create read task command", async () => {
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: HTTP.Task.COMMANDS,
      client,
    });
    store.dispatch(Session.Project.select(proj.key));
    await openCommandPalette("Create an HTTP Read");
    await selectCommand("Create an HTTP Read Task");
    expect(await resolveFocusedTab(store, client)).toMatchObject({
      variant: "view",
      type: HTTP.Task.READ_TYPE,
    });
  });

  it("should open the write task view from the create write task command", async () => {
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: HTTP.Task.COMMANDS,
      client,
    });
    store.dispatch(Session.Project.select(proj.key));
    await openCommandPalette("Create an HTTP Write");
    await selectCommand("Create an HTTP Write Task");
    expect(await resolveFocusedTab(store, client)).toMatchObject({
      variant: "view",
      type: HTTP.Task.WRITE_TYPE,
    });
  });
});
