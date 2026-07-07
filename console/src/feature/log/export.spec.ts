// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, DisconnectedError, log } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Log } from "@/feature/log";
import { createTestStore, uniqueName } from "@/testutil";

const client = createTestClient();

describe("log extractor", () => {
  it("should serialize the log with its layout type and version", async () => {
    const project = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const l = await client.logs.create(project.key, { name: uniqueName("log") });
    const store = await createTestStore();
    const extract = Log.EXTRACTORS[log.TYPE_ONTOLOGY_ID.type];
    const { data, name } = await extract(l.key, { client, store });
    expect(name).toBe(l.name);
    const parsed = JSON.parse(data);
    expect(parsed).toMatchObject({ key: l.key, type: log.TYPE_ONTOLOGY_ID.type });
    expect(parsed.version).toBe("2.0.0");
  });

  it("should reject with a disconnected error when no client exists", async () => {
    const store = await createTestStore();
    const extract = Log.EXTRACTORS[log.TYPE_ONTOLOGY_ID.type];
    await expect(extract("some_key", { client: null, store })).rejects.toSatisfy((e) =>
      DisconnectedError.matches(e),
    );
  });
});
