// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, staticWindow, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterAll, describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { framer } from "@/framer";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();
afterAll(async () => await client.close());

const POLL = { timeout: 10000, interval: 50 };

// Creates a calculation of twice a stored channel, with three samples written ten
// seconds ago.
const createCalculation = async () => {
  const time = await client.channels.create({
    name: id.create(),
    dataType: "timestamp",
    isIndex: true,
  });
  const data = await client.channels.create({
    name: id.create(),
    dataType: "float64",
    index: time.key,
  });
  const calc = await client.calculations.create({
    name: id.create(),
    expression: `return ${data.name} * 2`,
  });
  const start = TimeStamp.now().sub(TimeSpan.seconds(10));
  const stamps = [0, 1, 2].map((i) => start.add(TimeSpan.milliseconds(i)));
  await client.write(start, { [time.key]: stamps, [data.key]: [1, 2, 3] });
  const tr = new TimeRange(start, start.add(TimeSpan.seconds(1)));
  return { calc, tr };
};

const points = (key: channel.Key, tr: TimeRange): framer.PointsRequest => ({
  keys: [key],
  window: staticWindow(tr),
  limit: { variant: "points", count: 2 ** 20, aggregation: "min_max" },
});

const createReader = async () =>
  await createTestClientWithPolicy(client, {
    name: id.create(),
    objects: [channel.TYPE_RESOURCE_ID, framer.TYPE_RESOURCE_ID],
    actions: ["retrieve"],
  });

describe("framer cache on live calculation outputs", () => {
  it("should read the newest value of a hidden output", async () => {
    const { calc } = await createCalculation();
    const reader = await createReader();
    const feed = reader.openFeed();
    const sub = feed.subscribe({
      keys: [calc.output],
      limit: { variant: "latest", count: 1 },
    });
    try {
      await expect
        .poll(() => Array.from(sub.value(calc.output).series), POLL)
        .toEqual([6]);
    } finally {
      sub.close();
      await feed.close();
      await reader.close();
    }
  });

  it("should stop drawing a hidden output once its calculation goes", async () => {
    const { calc, tr } = await createCalculation();
    const reader = await createReader();
    const feed = reader.openFeed();
    const sub = feed.subscribe(points(calc.output, tr));
    try {
      await expect
        .poll(() => Array.from(sub.value(calc.output).series), POLL)
        .toEqual([2, 4, 6]);
      await client.calculations.delete(calc.key);
      await expect
        .poll(() => sub.value(calc.output).status.variant, POLL)
        .toBe("error");
      expect(sub.value(calc.output).status.description).toContain(
        "has no samples to draw",
      );
    } finally {
      sub.close();
      await feed.close();
      await reader.close();
    }
  });

  it("should stop drawing an output once its calculation is deleted", async () => {
    const { calc, tr } = await createCalculation();
    const feed = client.openFeed();
    const sub = feed.subscribe(points(calc.output, tr));
    try {
      await expect
        .poll(() => Array.from(sub.value(calc.output).series), POLL)
        .toEqual([2, 4, 6]);
      await client.calculations.delete(calc.key);
      await expect
        .poll(() => sub.value(calc.output).status.variant, POLL)
        .toBe("error");
      expect(sub.value(calc.output).status.description).toContain(
        "has no samples to draw",
      );
    } finally {
      sub.close();
      await feed.close();
    }
  });

  it("should stop drawing an output that another client's delete reaches", async () => {
    const { calc, tr } = await createCalculation();
    const other = createTestClient();
    const feed = client.openFeed();
    const sub = feed.subscribe(points(calc.output, tr));
    try {
      await expect
        .poll(() => Array.from(sub.value(calc.output).series), POLL)
        .toEqual([2, 4, 6]);
      await other.calculations.delete(calc.key);
      await expect
        .poll(() => sub.value(calc.output).status.description ?? "", POLL)
        .toContain("has no samples to draw");
    } finally {
      sub.close();
      await feed.close();
      await other.close();
    }
  });

  it("should draw an output again once a new calculation writes it", async () => {
    const { calc, tr } = await createCalculation();
    const other = createTestClient();
    const feed = client.openFeed();
    const sub = feed.subscribe(points(calc.output, tr));
    try {
      await expect
        .poll(() => Array.from(sub.value(calc.output).series), POLL)
        .toEqual([2, 4, 6]);
      await other.calculations.delete(calc.key);
      await expect
        .poll(() => sub.value(calc.output).status.description ?? "", POLL)
        .toContain("has no samples to draw");
      await other.calculations.create({
        output: calc.output,
        expression: calc.expression,
      });
      await expect
        .poll(() => Array.from(sub.value(calc.output).series), POLL)
        .toEqual([2, 4, 6]);
      expect(sub.value(calc.output).status.variant).toBe("success");
    } finally {
      sub.close();
      await feed.close();
      await other.close();
    }
  });
});
