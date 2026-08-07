// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, log } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Import } from "@/platform/import";
import { createFileIngesterContext } from "@/platform/import/testutil";
import { type Panel } from "@/platform/panel";
import { uniqueName } from "@/testutil";

const openedResource = (openTab: ReturnType<typeof vi.fn<Panel.OpenTab>>) => {
  expect(openTab).toHaveBeenCalledTimes(1);
  const [tab] = openTab.mock.calls[0];
  if (tab.variant !== "resource" || typeof tab.resource === "string")
    throw new Error("expected a resource tab");
  return tab.resource;
};

describe("ingestServer", () => {
  it("fails when disconnected", async () => {
    await expect(
      Import.ingestServer({ key: "x" }, createFileIngesterContext()),
    ).rejects.toThrow(DisconnectedError);
  });

  it("streams typed data to the Core and opens the created resource", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const original = await client.logs.create(proj.key, { name: uniqueName("log") });
    const stream = await client.imex.export(log.ontologyID(original.key), {
      encoding: "JSON",
    });
    const data = JSON.parse(await new Response(stream).text());
    const openTab = vi.fn<Panel.OpenTab>();
    await Import.ingestServer(
      data,
      createFileIngesterContext({ openTab, client, projectKey: proj.key }),
    );
    const created = await client.logs.retrieve({ key: openedResource(openTab).key });
    expect(created.name).toBe(original.name);
  });

  it("routes a typeless legacy state to the Core, naming it after the file", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const openTab = vi.fn<Panel.OpenTab>();
    // A legacy Console log state: version-stamped, no type, no name. The server
    // recognizes it by its frozen channels-array marker and names it after the file.
    const state = { version: "0.0.0", channels: [1, 2, 3], remoteCreated: false };
    await Import.ingestServer(
      state,
      createFileIngesterContext({
        openTab,
        client,
        projectKey: proj.key,
        fileName: "Legacy Log.json",
      }),
    );
    const created = await client.logs.retrieve({ key: openedResource(openTab).key });
    expect(created.name).toBe("Legacy Log");
  });

  it("migrates a legacy task config through the Core on import", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const openTab = vi.fn<Panel.OpenTab>();
    // A legacy Console task export: the camelCase config with only a type marker.
    const legacy = {
      type: "pagerduty_alert",
      routingKey: "R016395AF23B4E62B7A2BF7B24C1EF31",
      autoStart: true,
      alerts: [],
    };
    await Import.ingestServer(
      legacy,
      createFileIngesterContext({
        openTab,
        client,
        projectKey: proj.key,
        fileName: "PD Alerts.json",
      }),
    );
    const created = await client.tasks.retrieve({
      key: openedResource(openTab).key,
      schemas: {
        type: z.literal("pagerduty_alert"),
        config: z.looseObject({ routingKey: z.string(), autoStart: z.boolean() }),
        statusData: z.unknown(),
      },
    });
    expect(created.name).toBe("PD Alerts");
    expect(created.rack).toBe(0);
    expect(created.config.routingKey).toBe("R016395AF23B4E62B7A2BF7B24C1EF31");
    expect(created.config.autoStart).toBe(true);
  });
});
