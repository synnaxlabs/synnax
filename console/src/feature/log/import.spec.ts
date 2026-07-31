// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it, vi } from "vitest";

import { Log } from "@/feature/log";
import { createFileIngesterContext } from "@/platform/import/testutil";
import { type Panel } from "@/platform/panel";
import { assertDefined, awaitGranted, uniqueName } from "@/testutil";

describe("ingest", () => {
  it("should create the log on the cluster and open it as a tab", async () => {
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
    await awaitGranted(client, log.TYPE_ONTOLOGY_ID, "update");
    const openTab = vi.fn<Panel.OpenTab>();
    const id = await Log.ingest(
      data,
      createFileIngesterContext({ openTab, client, projectKey: proj.key }),
    );
    assertDefined(id, "ingest returned no id");
    expect(openTab).toHaveBeenCalledWith({ variant: "resource", resource: id });
    const created = await client.logs.retrieve({ key: id.key });
    expect(created.name).toBe(original.name);
  });
});
