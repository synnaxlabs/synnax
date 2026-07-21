// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { afterAll, describe, expect, it } from "vitest";

import { idToString } from "@/ontology/payload";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();
afterAll(() => client.close());

describe("ontology store", () => {
  it("caches resource sets and corpses deletes from live signals", async () => {
    await client.cache.ensureStreaming();
    const label = await client.labels.create({
      name: `cache-test-${id.create()}`,
      color: "#E774D0",
    });
    const resourceKey = idToString({ type: "label", key: label.key });
    await expect
      .poll(() => client.ontology.resources.get(resourceKey), { timeout: 5000 })
      .toBeDefined();
    expect(client.ontology.resources.status(resourceKey)).toBe("present");
    await client.labels.delete(label.key);
    await expect
      .poll(() => client.ontology.resources.status(resourceKey), { timeout: 5000 })
      .toBe("tombstoned");
    const tombstone = client.ontology.resources.getTombstone(resourceKey);
    expect(tombstone?.corpse.name).toEqual(label.name);
  }, 20000);

  it("stays a detached, local-only cache when caching is disabled", async () => {
    const disabled = createTestClient({ cache: false });
    expect(() => disabled.ontology.resources).not.toThrow();
    expect(disabled.cache.epoch).toBe(0);
    // Detached: ensureStreaming never opens a change stream, so the epoch
    // never advances past 0.
    await disabled.cache.ensureStreaming();
    expect(disabled.cache.epoch).toBe(0);
    disabled.close();
  });
});
