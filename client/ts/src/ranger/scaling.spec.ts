// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, WebSocketClient } from "@synnaxlabs/freighter";
import { binary, TimeRange, TimeStamp, url, uuid } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { type z } from "zod";

import { framer } from "@/framer";
import { label } from "@/label";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { ranger } from "@/ranger";

// Regression pin for SY-4751: one range fetch cost (ranges in the response) x
// (relationships cached), because every write-through recomposed against full
// scans of the relationships table. The whole client stack here is production
// code; only the network is synthetic.

const LABELS_PER_RANGE = 3;
const BATCH = 100;
const BATCHES = 30;

const labelPool = Array.from({ length: 40 }, () => ({
  key: uuid.create(),
  name: "label",
  color: "#000000",
}));

describe("range write-through scaling", () => {
  it("should keep fetch cost flat as the relationship table grows", async () => {
    let nextRange = 0;
    let batchSize = BATCH;
    const makeRange = () => {
      const i = nextRange++;
      return {
        key: uuid.create(),
        name: `range-${i}`,
        timeRange: new TimeRange({
          start: TimeStamp.seconds(i),
          end: TimeStamp.seconds(i + 1),
        }),
        labels: labelPool.slice(i % 20, (i % 20) + LABELS_PER_RANGE),
        parent: {
          key: uuid.create(),
          name: `parent-${i}`,
          timeRange: new TimeRange({
            start: TimeStamp.seconds(0),
            end: TimeStamp.seconds(1e6),
          }),
        },
      };
    };
    const unary: UnaryClient = {
      use: () => {},
      send: async <RQ extends z.ZodType, RS extends z.ZodType>(
        target: string,
        _req: z.input<RQ> | z.infer<RQ>,
        _reqZ: RQ,
        resZ: RS,
      ): Promise<z.infer<RS>> => {
        if (target === "/range/retrieve")
          return resZ.parse({
            ranges: Array.from({ length: batchSize }, makeRange),
          });
        return resZ.parse({});
      },
    };
    const cache = new query.Cache({ openStreamer: null, onError: () => {} });
    const ontologyClient = new ontology.Client({ unary, cache });
    const labels = new label.Client({ unary, cache, ontology: ontologyClient });
    const frameClient = new framer.Client({
      stream: new WebSocketClient(
        new url.URL({ host: "localhost", port: 9090 }),
        binary.JSON_CODEC,
      ),
      unary,
      retrieveChannels: async () => [],
    });
    const ranges = new ranger.Client({
      unary,
      cache,
      labels,
      ontology: ontologyClient,
      framer: frameClient,
      channels: async () => [],
    });
    const relationships = ontologyClient.cache.relationships;

    const fetchOnce = async (b: number): Promise<number> => {
      const start = performance.now();
      await ranges.retrieve({
        overlapsWith: new TimeRange({
          start: TimeStamp.seconds(b * 1000),
          end: TimeStamp.seconds(b * 1000 + 1),
        }),
      });
      return performance.now() - start;
    };
    const durations: number[] = [];
    for (let b = 0; b < BATCHES; b++) durations.push(await fetchOnce(b));

    expect(relationships.get().length).toBeGreaterThan(10_000);

    // The write-through path must never scan the relationships table: every
    // lookup goes through the byTo/byFrom indexes. A predicate get is a scan.
    const spy = vi.spyOn(relationships, "get");
    batchSize = 500;
    await fetchOnce(BATCHES);
    for (const [arg] of spy.mock.calls) expect(typeof arg).not.toBe("function");
    spy.mockRestore();

    // The relationships table grew ~30x between the first fetches and the
    // last; before the fix the last fetch cost >20x the first. The wide bound
    // and floor keep timer noise from flaking the assertion.
    const baseline = Math.max(Math.min(...durations.slice(0, 5)), 5);
    const last = durations[durations.length - 1];
    expect(last).toBeLessThan(baseline * 10);
  }, 120_000);
});
