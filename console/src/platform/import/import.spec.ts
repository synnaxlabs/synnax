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
import { ZodError } from "zod";

import { Import } from "@/platform/import";
import { createFileIngesterContext } from "@/platform/import/testutil";
import { type Panel } from "@/platform/panel";
import { uniqueName } from "@/testutil";

const ctx = createFileIngesterContext();

const openedResource = (openTab: ReturnType<typeof vi.fn<Panel.OpenTab>>) => {
  expect(openTab).toHaveBeenCalledTimes(1);
  const [tab] = openTab.mock.calls[0];
  if (tab.variant !== "resource" || typeof tab.resource === "string")
    throw new Error("expected a resource tab");
  return tab.resource;
};

describe("ingestComponent", () => {
  it("dispatches typed data to the ingester matching its type", async () => {
    const logIngest = vi.fn();
    const table = vi.fn();
    const data = { type: "log", key: "abc" };
    await Import.ingestComponent(data, { log: logIngest, table }, ctx);
    expect(logIngest).toHaveBeenCalledTimes(1);
    expect(logIngest).toHaveBeenCalledWith(data, ctx);
    expect(table).not.toHaveBeenCalled();
  });

  it("tries each ingester in turn for untyped data, stopping at the first success", async () => {
    const first = vi.fn().mockRejectedValue(new ZodError([]));
    const second = vi.fn().mockResolvedValue(undefined);
    const third = vi.fn();
    const data = { key: "no-type-field" };
    await Import.ingestComponent(data, { first, second, third }, ctx);
    expect(first).toHaveBeenCalledWith(data, ctx);
    expect(second).toHaveBeenCalledWith(data, ctx);
    expect(third).not.toHaveBeenCalled();
  });

  it("rethrows a non-Zod error raised by an untyped ingester without trying the rest", async () => {
    const boom = new Error("disk on fire");
    const first = vi.fn().mockRejectedValue(boom);
    const second = vi.fn();
    await expect(
      Import.ingestComponent({ key: "x" }, { first, second }, ctx),
    ).rejects.toThrow("disk on fire");
    expect(second).not.toHaveBeenCalled();
  });

  it("reaches the server fallback for unclaimed untyped data, failing when disconnected", async () => {
    const first = vi.fn().mockRejectedValue(new ZodError([]));
    await expect(Import.ingestComponent({ key: "x" }, { first }, ctx)).rejects.toThrow(
      DisconnectedError,
    );
  });

  it("streams typed data with no client-side ingester to the server", async () => {
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
    const task = vi.fn();
    const openTab = vi.fn<Panel.OpenTab>();
    await Import.ingestComponent(
      data,
      { some_task: task },
      createFileIngesterContext({ openTab, client, projectKey: proj.key }),
    );
    expect(task).not.toHaveBeenCalled();
    const created = await client.logs.retrieve({ key: openedResource(openTab).key });
    expect(created.name).toBe(original.name);
  });

  it("routes a typeless legacy state to the server after client ingesters decline", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const task = vi.fn().mockRejectedValue(new ZodError([]));
    const openTab = vi.fn<Panel.OpenTab>();
    const serverCtx = createFileIngesterContext({
      openTab,
      client,
      projectKey: proj.key,
      fileName: "Legacy Log.json",
    });
    // A legacy Console log state: version-stamped, no type, no name. The server
    // recognizes it by its frozen channels-array marker and names it after the file.
    const state = { version: "0.0.0", channels: [1, 2, 3], remoteCreated: false };
    await Import.ingestComponent(state, { task }, serverCtx);
    expect(task).toHaveBeenCalledWith(state, serverCtx);
    const created = await client.logs.retrieve({ key: openedResource(openTab).key });
    expect(created.name).toBe("Legacy Log");
  });
});
