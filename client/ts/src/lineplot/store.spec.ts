// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { afterAll, describe, expect, it } from "vitest";

import { rename } from "@/lineplot/actions.gen";
import { STORE_KEY } from "@/lineplot/store";
import { type Key, type LinePlot } from "@/lineplot/types.gen";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();
afterAll(() => client.close());

const seed = async (): Promise<LinePlot> => {
  const project = await client.projects.create({ name: `lp-${id.create()}` });
  const plot = await client.lineplots.create(project.key, {
    name: `plot-${id.create()}`,
  });
  client.cache.engine.store<Key, LinePlot>(STORE_KEY).set(plot.key, plot);
  return plot;
};

describe("lineplot store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.cache.engine.ensureStreaming();
    const plot = await seed();
    await client.lineplots.delete(plot.key);
    await expect
      .poll(() => client.lineplots.store.status(plot.key), { timeout: 5000 })
      .toBe("tombstoned");
    expect(client.lineplots.store.getTombstone(plot.key)?.corpse.name).toEqual(
      plot.name,
    );
  }, 20000);

  it("reduces broadcast dispatch frames into the cached document", async () => {
    await client.cache.engine.ensureStreaming();
    const plot = await seed();
    const name = `renamed-${id.create()}`;
    await client.lineplots.dispatch(plot.key, uuid.create(), [rename({ name })]);
    await expect
      .poll(() => client.lineplots.store.get(plot.key)?.name, { timeout: 5000 })
      .toBe(name);
  }, 20000);
});
